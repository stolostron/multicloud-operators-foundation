package cache

import (
	"errors"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/storage"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

type CacheWatcher interface {
	// GroupMembershipChanged is called serially for all changes for all watchers.  This method MUST NOT BLOCK.
	// The serial nature makes reasoning about the code easy, but if you block in this method you will doom all watchers.
	GroupMembershipChanged(names, users, groups sets.String)
}

type WatchableCache interface {
	// RemoveWatcher removes a watcher
	RemoveWatcher(CacheWatcher)

	ListObjects(user user.Info) (runtime.Object, error)

	Get(name string) (runtime.Object, error)

	ConvertResource(name string) runtime.Object
}

// cacheWatcher converts a native etcd watch to a watch.Interface.
type cacheWatcher struct {
	user user.Info
	// cacheIncoming is a buffered channel used for notification to watcher.  If the buffer fills up,
	// then the watcher will be removed and the connection will be broken.
	cacheIncoming chan watch.Event
	// cacheError is a cached channel that is put to serially.  In theory, only one item will
	// ever be placed on it.
	cacheError chan error

	// outgoing is the unbuffered `ResultChan` use for the watch.  Backups of this channel will block
	// the default `emit` call.  That's why cacheError is a buffered channel.
	outgoing chan watch.Event
	// userStop lets a user stop his watch.
	userStop chan struct{}

	// stopLock keeps parallel stops from doing crazy things
	stopLock sync.Mutex

	// Injectable for testing. Send the event down the outgoing channel.
	emit func(watch.Event)

	nsLister  corev1listers.NamespaceLister
	authCache WatchableCache

	initialResources []runtime.Object
	// knownResources maps name to resourceVersion
	knownResources map[string]string
}

var (
	// watchChannelHWM tracks how backed up the most backed up channel got.  This mirrors etcd watch behavior and allows tuning
	// of channel depth.
	watchChannelHWM storage.HighWaterMark
)

func NewCacheWatcher(user user.Info, authCache WatchableCache, includeAllExistingResources bool) *cacheWatcher {
	objectList, _ := authCache.ListObjects(user)
	objs, _ := meta.ExtractList(objectList)
	knownResources := map[string]string{}
	for _, object := range objs {
		accessor, _ := meta.Accessor(object)
		knownResources[accessor.GetName()] = accessor.GetResourceVersion()
	}

	// this is optional.  If they don't request it, don't include it.
	initialResources := []runtime.Object{}
	if includeAllExistingResources {
		initialResources = append(initialResources, objs...)
	}

	w := &cacheWatcher{
		user:          user,
		cacheIncoming: make(chan watch.Event, 1000),
		cacheError:    make(chan error, 1),
		outgoing:      make(chan watch.Event),
		userStop:      make(chan struct{}),

		authCache:        authCache,
		initialResources: initialResources,
		knownResources:   knownResources,
	}
	w.emit = func(e watch.Event) {
		select {
		case w.outgoing <- e:
		case <-w.userStop:
		}
	}
	return w
}

func (w *cacheWatcher) GroupMembershipChanged(names, users, groups sets.String) {
	hasAccess := users.Has(w.user.GetName()) || groups.HasAny(w.user.GetGroups()...)
	if !hasAccess {
		return
	}
	for name := range w.knownResources {
		if !names.Has(name) {
			delete(w.knownResources, name)
			select {
			case w.cacheIncoming <- watch.Event{
				Type:   watch.Deleted,
				Object: w.authCache.ConvertResource(name),
			}:
			default:
				// remove the watcher so that we wont' be notified again and block
				w.authCache.RemoveWatcher(w)
				w.cacheError <- errors.New("delete notification timeout")
			}
		}
	}

	for _, name := range names.List() {
		object, err := w.authCache.Get(name)
		if err != nil {
			utilruntime.HandleError(err)
			continue
		}

		event := watch.Event{
			Type:   watch.Added,
			Object: object,
		}
		accessor, _ := meta.Accessor(object)

		// if we already have this in our list, then we're getting notified because the object changed
		if lastResourceVersion, known := w.knownResources[name]; known {
			event.Type = watch.Modified

			// if we've already notified for this particular resourceVersion, there's no work to do
			if lastResourceVersion == accessor.GetResourceVersion() {
				continue
			}
		}
		w.knownResources[name] = accessor.GetResourceVersion()

		select {
		case w.cacheIncoming <- event:
		default:
			// remove the watcher so that we won't be notified again and block
			w.authCache.RemoveWatcher(w)
			w.cacheError <- errors.New("add notification timeout")
		}
	}
}

// Watch pulls stuff from etcd, converts, and pushes out the outgoing channel. Meant to be
// called as a goroutine.
func (w *cacheWatcher) Watch() {
	defer close(w.outgoing)
	defer func() {
		// when the watch ends, always remove the watcher from the cache to avoid leaking.
		w.authCache.RemoveWatcher(w)
	}()
	defer utilruntime.HandleCrash()

	// start by emitting all the `initialResources`
	for i := range w.initialResources {
		// keep this check here to sure we don't keep this open in the case of failures
		select {
		case err := <-w.cacheError:
			w.emit(makeErrorEvent(err))
			return
		default:
		}

		w.emit(watch.Event{
			Type:   watch.Added,
			Object: w.initialResources[i].DeepCopyObject(),
		})
	}

	for {
		select {
		case err := <-w.cacheError:
			w.emit(makeErrorEvent(err))
			return

		case <-w.userStop:
			return

		case event := <-w.cacheIncoming:
			if curLen := int64(len(w.cacheIncoming)); watchChannelHWM.Update(curLen) {
				// Monitor if this gets backed up, and how much.
				klog.V(2).Infof("watch: %v objects queued in managedCluster cache watching channel.", curLen)
			}

			w.emit(event)
		}
	}
}

func makeErrorEvent(err error) watch.Event {
	return watch.Event{
		Type: watch.Error,
		Object: &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: err.Error(),
		},
	}
}

// ResultChan implements watch.Interface.
func (w *cacheWatcher) ResultChan() <-chan watch.Event {
	return w.outgoing
}

// Stop implements watch.Interface.
func (w *cacheWatcher) Stop() {
	// lock access so we don't race past the channel select
	w.stopLock.Lock()
	defer w.stopLock.Unlock()

	// Prevent double channel closes.
	select {
	case <-w.userStop:
		return
	default:
	}
	close(w.userStop)
}
