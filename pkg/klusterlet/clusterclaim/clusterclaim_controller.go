package clusterclaim

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	clusterclientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterinformers "github.com/open-cluster-management/api/client/cluster/informers/externalversions"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ListClusterClaimsFunc func() ([]*clusterv1alpha1.ClusterClaim, error)

// ClusterClaimReconciler reconciles cluster claim objects
type ClusterClaimReconciler struct {
	Log               logr.Logger
	ListClusterClaims ListClusterClaimsFunc
	ClusterClient     clusterclientset.Interface
}

func (r *ClusterClaimReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	err := r.syncClaims()
	return ctrl.Result{}, err
}

func (r *ClusterClaimReconciler) syncClaims() error {
	r.Log.V(4).Info("Sync cluster claims")
	claims, err := r.ListClusterClaims()
	if err != nil {
		return err
	}

	errs := []error{}
	for _, claim := range claims {
		if err := r.createOrUpdate(claim); err != nil {
			errs = append(errs, err)
		}
	}
	return utils.NewMultiLineAggregate(errs)
}

func (r *ClusterClaimReconciler) createOrUpdate(newClaim *clusterv1alpha1.ClusterClaim) error {
	oldClaim, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(), newClaim.Name, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		_, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Create(context.Background(), newClaim, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("unable to create ClusterClaim: %v, %w", newClaim, err)
		}
	case err != nil:
		return fmt.Errorf("unable to get ClusterClaim %q: %w", newClaim.Name, err)
	case !reflect.DeepEqual(oldClaim.Spec, newClaim.Spec):
		oldClaim.Spec = newClaim.Spec
		_, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Update(context.Background(), oldClaim, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update ClusterClaim %q: %w", oldClaim.Name, err)
		}
	}
	return nil
}

func (r *ClusterClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// create or update the claims before controller starts
	if err := r.syncClaims(); err != nil {
		return err
	}

	// setup a controller for ClusterClaim with customized event source and handler
	c, err := controller.New("ClusterClaim", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	source := newClusterClaimSource(r.ClusterClient)
	if err := c.Watch(source, &clusterClaimEventHandler{}); err != nil {
		return err
	}
	return nil
}

// newClusterClaimSource returns an event source for cluster claims
func newClusterClaimSource(clusterClient clusterclientset.Interface) source.Source {
	informerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	return &clusterClaimSource{
		informerFactory: informerFactory,
		claimInformer:   informerFactory.Cluster().V1alpha1().ClusterClaims().Informer(),
	}
}

// clusterClaimSource is the event source of cluster claims on managed cluster.
type clusterClaimSource struct {
	informerFactory clusterinformers.SharedInformerFactory
	claimInformer   cache.SharedIndexInformer
}

var _ source.SyncingSource = &clusterClaimSource{}

func (s *clusterClaimSource) Start(handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	// all predicates are ignored
	s.claimInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handler.Create(event.CreateEvent{}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handler.Update(event.UpdateEvent{}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			handler.Delete(event.DeleteEvent{}, queue)
		},
	})

	return nil
}

func (s *clusterClaimSource) WaitForSync(stop <-chan struct{}) error {
	go s.informerFactory.Start(stop)

	if ok := cache.WaitForCacheSync(stop, s.claimInformer.HasSynced); !ok {
		return fmt.Errorf("Never achieved initial sync")
	}
	return nil
}

// clusterClaimEventHandler maps any event to an empty request
type clusterClaimEventHandler struct{}

var _ handler.EventHandler = &clusterClaimEventHandler{}

func (e *clusterClaimEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
