package objectsource

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

// NewClusterClaimSource returns an event source for cluster claims
func NewClusterClaimSource(clusterInformers clusterv1alpha1informer.ClusterClaimInformer) source.Source {
	return &clusterClaimSource{
		claimInformer: clusterInformers.Informer(),
		handler:       newClusterClaimEventHandler(),
	}
}

// clusterClaimSource is the event source of cluster claims on managed cluster.
type clusterClaimSource struct {
	claimInformer cache.SharedIndexInformer
	handler       handler.EventHandler
}

var _ source.SyncingSource = &clusterClaimSource{}

func (s *clusterClaimSource) Start(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	// all predicates are ignored
	s.claimInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.handler.Create(ctx, event.CreateEvent{
				Object: obj.(*clusterv1alpha1.ClusterClaim),
			}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.handler.Update(ctx, event.UpdateEvent{
				ObjectOld: oldObj.(*clusterv1alpha1.ClusterClaim),
				ObjectNew: newObj.(*clusterv1alpha1.ClusterClaim),
			}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			s.handler.Delete(ctx, event.DeleteEvent{
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

// newClusterClaimEventHandler maps any event to an empty request
func newClusterClaimEventHandler() *clusterClaimEventHandler {
	return &clusterClaimEventHandler{}
}

type clusterClaimEventHandler struct{}

var _ handler.EventHandler = &clusterClaimEventHandler{}

func (e *clusterClaimEventHandler) Create(_ context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Update(_ context.Context, evt event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Delete(_ context.Context, evt event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}

func (e *clusterClaimEventHandler) Generic(_ context.Context, evt event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}
