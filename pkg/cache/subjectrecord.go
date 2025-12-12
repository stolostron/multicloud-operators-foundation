package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
	"k8s.io/klog/v2"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
	rbacv1informers "k8s.io/client-go/informers/rbac/v1"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

// SubjectRecord is a cache record for the set of resources a subject can access
type SubjectRecord struct {
	Subject string
	Names   sets.String
}

// subjectRecordKeyFn is a key func for SubjectRecord objects
func subjectRecordKeyFn(obj interface{}) (string, error) {
	SubjectRecord, ok := obj.(*SubjectRecord)
	if !ok {
		return "", fmt.Errorf("expected SubjectRecord")
	}
	return SubjectRecord.Subject, nil
}

// LastSyncResourceVersioner is any object that can divulge a LastSyncResourceVersion
type LastSyncResourceVersioner interface {
	LastSyncResourceVersion() string
}

type unionLastSyncResourceVersioner []LastSyncResourceVersioner

func (u unionLastSyncResourceVersioner) LastSyncResourceVersion() string {
	resourceVersions := []string{}
	for _, versioner := range u {
		resourceVersions = append(resourceVersions, versioner.LastSyncResourceVersion())
	}
	return strings.Join(resourceVersions, "")
}

type skipSynchronizer interface {
	// SkipSynchronize returns true if if its safe to skip synchronization of the cache based on provided token from previous observation
	SkipSynchronize(prevState string, versionedObjects ...LastSyncResourceVersioner) (skip bool, currentState string)
}

type statelessSkipSynchronizer struct{}

func (rs *statelessSkipSynchronizer) SkipSynchronize(prevState string, versionedObjects ...LastSyncResourceVersioner) (skip bool, currentState string) {
	resourceVersions := []string{}
	for i := range versionedObjects {
		resourceVersions = append(resourceVersions, versionedObjects[i].LastSyncResourceVersion())
	}
	currentState = strings.Join(resourceVersions, ",")
	skip = currentState == prevState
	return skip, currentState
}

type SyncedClusterRoleLister interface {
	rbacv1listers.ClusterRoleLister
	LastSyncResourceVersioner
}

type SyncedClusterRoleBindingLister interface {
	rbacv1listers.ClusterRoleBindingLister
	LastSyncResourceVersioner
}

type syncedClusterRoleLister struct {
	rbacv1listers.ClusterRoleLister
	versioner LastSyncResourceVersioner
}

func (l syncedClusterRoleLister) LastSyncResourceVersion() string {
	return l.versioner.LastSyncResourceVersion()
}

type syncedClusterRoleBindingLister struct {
	rbacv1listers.ClusterRoleBindingLister
	versioner LastSyncResourceVersioner
}

func (l syncedClusterRoleBindingLister) LastSyncResourceVersion() string {
	return l.versioner.LastSyncResourceVersion()
}

type AuthCache struct {
	// the known items are used to get deleted items
	knownResources           sets.String
	knownUsers               sets.String
	knownGroups              sets.String
	clusterRoleLister        SyncedClusterRoleLister
	clusterRolebindingLister SyncedClusterRoleBindingLister

	lastSyncResourceVersioner       LastSyncResourceVersioner
	policyLastSyncResourceVersioner LastSyncResourceVersioner
	skip                            skipSynchronizer
	lastState                       string

	userSubjectRecordStore  cache.Store
	groupSubjectRecordStore cache.Store

	syncResources                   func() (sets.String, error)
	getResourceNamesFromClusterRole func(*rbacv1.ClusterRole, string, string) (sets.String, bool)

	group    string
	resource string

	watchers    []CacheWatcher
	watcherLock sync.Mutex
}

func NewAuthCache(clusterRoleInformer rbacv1informers.ClusterRoleInformer,
	clusterRolebindingInformer rbacv1informers.ClusterRoleBindingInformer,
	group, resource string,
	lastSyncResourceVersioner LastSyncResourceVersioner,
	syncResourcesFunc func() (sets.String, error),
	getResourceNamesFromClusterRole func(*rbacv1.ClusterRole, string, string) (sets.String, bool),
) *AuthCache {
	scrLister := syncedClusterRoleLister{
		clusterRoleInformer.Lister(),
		clusterRoleInformer.Informer(),
	}
	scrbLister := syncedClusterRoleBindingLister{
		clusterRolebindingInformer.Lister(),
		clusterRolebindingInformer.Informer(),
	}
	result := &AuthCache{
		clusterRoleLister:               scrLister,
		clusterRolebindingLister:        scrbLister,
		syncResources:                   syncResourcesFunc,
		lastSyncResourceVersioner:       lastSyncResourceVersioner,
		policyLastSyncResourceVersioner: unionLastSyncResourceVersioner{scrLister, scrbLister},

		group:    group,
		resource: resource,

		userSubjectRecordStore:  cache.NewStore(subjectRecordKeyFn),
		groupSubjectRecordStore: cache.NewStore(subjectRecordKeyFn),
		skip:                    &statelessSkipSynchronizer{},

		getResourceNamesFromClusterRole: getResourceNamesFromClusterRole,

		watchers: []CacheWatcher{},
	}

	return result
}

// synchronize runs a a full synchronization over the cache data.  it must be run in a single-writer model, it's not thread-safe by design.
func (ac *AuthCache) synchronize() {
	startTime := time.Now()
	// if none of our internal reflectors changed, then we can skip reviewing the cache
	skip, currentState := ac.skip.SkipSynchronize(ac.lastState, ac.lastSyncResourceVersioner, ac.policyLastSyncResourceVersioner)
	if skip {
		return
	}

	userSubjectRecordStore := ac.userSubjectRecordStore
	groupSubjectRecordStore := ac.groupSubjectRecordStore

	resources, err := ac.syncResources()
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	ac.knownResources = resources
	ac.synchronizeClusterRoleBindings(userSubjectRecordStore, groupSubjectRecordStore)
	ac.lastState = currentState
	klog.V(2).Infof("synchronize: resources=%d users=%d groups=%d took=%v",
		len(ac.knownResources), len(ac.knownUsers), len(ac.knownGroups), time.Since(startTime))
	klog.V(5).Infoln("synchronize...", ac.knownResources, ac.knownUsers, ac.knownGroups)
}

// synchronizeRoleBindings synchronizes access over each clusterRoleBinding
// List all of users/groups in each clusterRoleBinding and their resources in each clusterRole.
// update all of user-resources to the userSubjectRecordStore
// update all of group-resources to the groupSubjectRecordStore
// delete all of user records in userSubjectRecordStore if user is not found
// delete all of group records in groupSubjectRecordStore if group is not found
func (ac *AuthCache) synchronizeClusterRoleBindings(userSubjectRecordStore cache.Store, groupSubjectRecordStore cache.Store) {
	roleBindings, err := ac.clusterRolebindingLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	newAllUsers := sets.String{}
	newAllGroups := sets.String{}
	userToResources := map[string]sets.String{}
	groupToResources := map[string]sets.String{}

	for _, roleBinding := range roleBindings {
		clusterRole, err := ac.clusterRoleLister.Get(roleBinding.RoleRef.Name)
		if err != nil {
			continue
		}
		resources, all := ac.getResourceNamesFromClusterRole(clusterRole, ac.group, ac.resource)
		if all {
			resources = ac.knownResources
		}
		if len(resources) == 0 {
			continue
		}

		users, groups := authorizationutil.RBACSubjectsToUsersAndGroups(roleBinding.Subjects, "")
		for _, user := range users {
			newAllUsers.Insert(user)
			for _, resource := range resources.List() {
				if !ac.knownResources.Has(resource) {
					continue
				}
				if userToResources[user] == nil {
					userToResources[user] = sets.String{}
				}
				userToResources[user].Insert(resource)
			}
		}
		for _, group := range groups {
			newAllGroups.Insert(group)
			for _, resource := range resources.List() {
				if !ac.knownResources.Has(resource) {
					continue
				}
				if groupToResources[group] == nil {
					groupToResources[group] = sets.String{}
				}
				groupToResources[group].Insert(resource)
			}
		}
	}

	for updatedUser, updatedResources := range userToResources {
		updateResourcesToSubject(userSubjectRecordStore, updatedUser, updatedResources)
		ac.notifyWatchers(updatedResources, sets.NewString(updatedUser), sets.NewString())
	}

	for updatedGroup, updatedResources := range groupToResources {
		updateResourcesToSubject(groupSubjectRecordStore, updatedGroup, updatedResources)
		ac.notifyWatchers(updatedResources, sets.NewString(), sets.NewString(updatedGroup))
	}

	for deletedUser := range ac.knownUsers.Difference(newAllUsers) {
		deleteSubject(userSubjectRecordStore, deletedUser)
		ac.notifyWatchers(sets.NewString(), sets.NewString(deletedUser), sets.NewString())
	}
	for deletedGroup := range ac.knownGroups.Difference(newAllGroups) {
		deleteSubject(groupSubjectRecordStore, deletedGroup)
		ac.notifyWatchers(sets.NewString(), sets.NewString(), sets.NewString(deletedGroup))
	}

	ac.knownUsers = newAllUsers
	ac.knownGroups = newAllGroups
}

func (ac *AuthCache) GetUserSubjectRecord() []*SubjectRecord {
	if ac == nil || ac.userSubjectRecordStore == nil {
		return []*SubjectRecord{}
	}
	subjectRecordStore := ac.userSubjectRecordStore.List()
	var returnSubjectRecord []*SubjectRecord
	for _, subjectRecord := range subjectRecordStore {
		s := subjectRecord.(*SubjectRecord)
		returnSubjectRecord = append(returnSubjectRecord, s)
	}
	return returnSubjectRecord
}

func (ac *AuthCache) GetGroupSubjectRecord() []*SubjectRecord {
	if ac == nil || ac.userSubjectRecordStore == nil {
		return []*SubjectRecord{}
	}
	subjectRecordStore := ac.groupSubjectRecordStore.List()
	var returnSubjectRecord []*SubjectRecord
	for _, subjectRecord := range subjectRecordStore {
		s := subjectRecord.(*SubjectRecord)
		returnSubjectRecord = append(returnSubjectRecord, s)
	}
	return returnSubjectRecord
}

func (ac *AuthCache) listNames(userInfo user.Info) sets.String {
	keys := sets.String{}
	user := userInfo.GetName()
	groups := userInfo.GetGroups()

	obj, exists, _ := ac.userSubjectRecordStore.GetByKey(user)
	if exists {
		SubjectRecord := obj.(*SubjectRecord)
		keys.Insert(SubjectRecord.Names.List()...)
	}

	for _, group := range groups {
		obj, exists, _ := ac.groupSubjectRecordStore.GetByKey(group)
		if exists {
			SubjectRecord := obj.(*SubjectRecord)
			keys.Insert(SubjectRecord.Names.List()...)
		}
	}

	return keys
}

func (ac *AuthCache) AddWatcher(watcher CacheWatcher) {
	ac.watcherLock.Lock()
	defer ac.watcherLock.Unlock()

	ac.watchers = append(ac.watchers, watcher)
}

func (ac *AuthCache) RemoveWatcher(watcher CacheWatcher) {
	ac.watcherLock.Lock()
	defer ac.watcherLock.Unlock()

	lastIndex := len(ac.watchers) - 1
	for i := 0; i < len(ac.watchers); i++ {
		if ac.watchers[i] == watcher {
			if i < lastIndex {
				// if we're not the last element, shift
				copy(ac.watchers[i:], ac.watchers[i+1:])
			}
			ac.watchers = ac.watchers[:lastIndex]
			break
		}
	}
}

func (ac *AuthCache) notifyWatchers(names, users, groups sets.String) {
	ac.watcherLock.Lock()
	defer ac.watcherLock.Unlock()
	for _, watcher := range ac.watchers {
		watcher.GroupMembershipChanged(names, users, groups)
	}
}

func updateResourcesToSubject(subjectRecordStore cache.Store, subject string, names sets.String) {
	var item *SubjectRecord
	obj, exists, _ := subjectRecordStore.GetByKey(subject)
	if exists {
		item = obj.(*SubjectRecord)
		item.Names = names
	} else {
		item = &SubjectRecord{Subject: subject, Names: names}
		if err := subjectRecordStore.Add(item); err != nil {
			klog.Warningf("failed to add %v: %v", item, err)
		}
	}
	return
}

func deleteSubject(subjectRecordStore cache.Store, subject string) {
	obj, exists, _ := subjectRecordStore.GetByKey(subject)
	if exists {
		SubjectRecord := obj.(*SubjectRecord)
		if err := subjectRecordStore.Delete(SubjectRecord); err != nil {
			klog.Warningf("failed to delete %v: %v", SubjectRecord, err)
		}
	}

	return
}
