package clusterclaim

import (
	"context"
	"fmt"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func (r *ClusterClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// setup a controller for ClusterClaim with customized event source and handler
	c, err := controller.New("ClusterClaim", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	claimSource := NewClusterClaimSource(r.ClusterInformers)
	if err := c.Watch(claimSource, &ClusterClaimEventHandler{}); err != nil {
		return err
	}

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
			handler.Create(ctx, event.CreateEvent{}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handler.Update(ctx, event.UpdateEvent{}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			handler.Delete(ctx, event.DeleteEvent{}, queue)
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
type ClusterClaimEventHandler struct{}

var _ handler.EventHandler = &ClusterClaimEventHandler{}

func (e *ClusterClaimEventHandler) Create(_ context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *ClusterClaimEventHandler) Update(_ context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *ClusterClaimEventHandler) Delete(_ context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *ClusterClaimEventHandler) Generic(_ context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
