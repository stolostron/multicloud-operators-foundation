package objectsource

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
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

func (s *NodeSource) Start(ctx context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	s.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.handler.Create(ctx, event.CreateEvent{}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if !nodeEventAffectsClusterClaim(oldObj, newObj) {
				return
			}
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

// nodeEventAffectsClusterClaim reports whether a node update can change
// schedulable.open-cluster-management.io or related cluster inventory data.
// Routine kubelet heartbeats and lease refreshes are ignored.
func nodeEventAffectsClusterClaim(oldObj, newObj interface{}) bool {
	oldNode, ok := oldObj.(*corev1.Node)
	if !ok {
		return true
	}
	newNode, ok := newObj.(*corev1.Node)
	if !ok {
		return true
	}

	if oldNode.Spec.Unschedulable != newNode.Spec.Unschedulable {
		return true
	}
	if !apiequality.Semantic.DeepEqual(oldNode.Spec.Taints, newNode.Spec.Taints) {
		return true
	}
	if !apiequality.Semantic.DeepEqual(oldNode.Status.Capacity, newNode.Status.Capacity) {
		return true
	}
	if nodeConditionStatus(oldNode, corev1.NodeReady) != nodeConditionStatus(newNode, corev1.NodeReady) {
		return true
	}
	if nodeConditionStatus(oldNode, corev1.NodeNetworkUnavailable) != nodeConditionStatus(newNode, corev1.NodeNetworkUnavailable) {
		return true
	}
	return false
}

func nodeConditionStatus(node *corev1.Node, conditionType corev1.NodeConditionType) corev1.ConditionStatus {
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return ""
}

// nodeEventHandler maps any event to an empty request
type nodeEventHandler struct{}

func newNodeEventHandler() *nodeEventHandler {
	return &nodeEventHandler{}
}

func (e *nodeEventHandler) Create(ctx context.Context, evt event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	q.Add(reconcile.Request{})
}
