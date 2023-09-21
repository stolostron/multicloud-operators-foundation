package controllers

import (
	"context"
	"fmt"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	claimSource := clusterclaim.NewClusterClaimSource(r.ClaimInformer)
	nodeSource := &nodeSource{nodeInformer: r.NodeInformer.Informer()}
	return ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(claimSource, clusterclaim.NewClusterClaimEventHandler()).
		WatchesRawSource(nodeSource, &nodeEventHandler{}).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}

// nodeSource is the event source of nodes on managed cluster.
type nodeSource struct {
	nodeInformer cache.SharedIndexInformer
}

var _ source.SyncingSource = &nodeSource{}

func (s *nodeSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	// all predicates are ignored
	s.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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

func (s *nodeSource) WaitForSync(ctx context.Context) error {
	if ok := cache.WaitForCacheSync(ctx.Done(), s.nodeInformer.HasSynced); !ok {
		return fmt.Errorf("never achieved initial sync")
	}
	return nil
}

// nodeEventHandler maps any event to an empty request
type nodeEventHandler struct{}

var _ handler.EventHandler = &nodeEventHandler{}

func (e *nodeEventHandler) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
