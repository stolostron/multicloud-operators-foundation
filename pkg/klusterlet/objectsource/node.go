package objectsource

import (
	"context"
	"fmt"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NodeSource is the event source of nodes on managed cluster.
type NodeSource struct {
	nodeInformer cache.SharedIndexInformer
	handler      handler.EventHandler
}

// NewNodeSource returns an event source for node source
func NewNodeSource(nodeInformer coreinformers.NodeInformer) source.Source {
	return &NodeSource{
		nodeInformer: nodeInformer.Informer(),
		handler:      newNodeEventHandler(),
	}
}

var _ source.SyncingSource = &NodeSource{}

func (s *NodeSource) Start(ctx context.Context, queue workqueue.RateLimitingInterface) error {
	// all predicates are ignored
	s.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.handler.Create(ctx, event.CreateEvent{}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.handler.Update(ctx, event.UpdateEvent{}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			s.handler.Delete(ctx, event.DeleteEvent{}, queue)
		},
	})

	return nil
}

func (s *NodeSource) WaitForSync(ctx context.Context) error {
	if ok := cache.WaitForCacheSync(ctx.Done(), s.nodeInformer.HasSynced); !ok {
		return fmt.Errorf("never achieved initial sync")
	}
	return nil
}

// nodeEventHandler maps any event to an empty request
type nodeEventHandler struct{}

func newNodeEventHandler() *nodeEventHandler {
	return &nodeEventHandler{}
}

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
