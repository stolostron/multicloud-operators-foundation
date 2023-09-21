package clusterclaim

import (
	"context"
	"fmt"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func (r *clusterClaimReconciler) SetupWithManager(mgr ctrl.Manager, clusterInformer clusterv1alpha1informer.ClusterClaimInformer) error {
	// setup a controller for ClusterClaim with customized event source and handler
	c, err := controller.New("ClusterClaim", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	claimSource := NewClusterClaimSource(clusterInformer)
	if err := c.Watch(claimSource, NewClusterClaimEventHandler()); err != nil {
		return err
	}

	// There are 2 use cases for this watch:
	// 1. when labels of the managedclusterinfo changed, we need to sync labels to clusterclaims
	// 2. at the beginning of the pod, we need this watch to trigger the reconcile of all clusterclaims
	if err := c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta1.ManagedClusterInfo{}), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

// NewClusterClaimSource returns an event source for cluster claims
func NewClusterClaimSource(clusterInformers clusterv1alpha1informer.ClusterClaimInformer) source.Source {
	return &clusterClaimSource{
		claimInformer: clusterInformers.Informer(),
	}
}

// clusterClaimSource is the event source of cluster claims on managed cluster.
type clusterClaimSource struct {
	claimInformer cache.SharedIndexInformer
}

var _ source.SyncingSource = &clusterClaimSource{}

func (s *clusterClaimSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	// all predicates are ignored
	s.claimInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handler.Create(ctx, event.CreateEvent{
				Object: obj.(*clusterv1alpha1.ClusterClaim),
			}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handler.Update(ctx, event.UpdateEvent{
				ObjectOld: oldObj.(*clusterv1alpha1.ClusterClaim),
				ObjectNew: newObj.(*clusterv1alpha1.ClusterClaim),
			}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			handler.Delete(ctx, event.DeleteEvent{
				Object: obj.(*clusterv1alpha1.ClusterClaim),
			}, queue)
		},
	})

	return nil
}

func (s *clusterClaimSource) WaitForSync(ctx context.Context) error {
	if ok := cache.WaitForCacheSync(ctx.Done(), s.claimInformer.HasSynced); !ok {
		return fmt.Errorf("Never achieved initial sync")
	}
	return nil
}

// ClusterClaimEventHandler maps any event to an empty request
func NewClusterClaimEventHandler() *clusterClaimEventHandler {
	return &clusterClaimEventHandler{}
}

type clusterClaimEventHandler struct{}

var _ handler.EventHandler = &clusterClaimEventHandler{}

func (e *clusterClaimEventHandler) Create(_ context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Update(_ context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Delete(_ context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Generic(_ context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
