package objectsource

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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
			if !nodeChanged(oldObj, newObj) {
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

// nodeChanged reports whether a node update should trigger a reconcile.
//
// The kubelet updates node status often (conditions heartbeats, images, volumes),
// and none of that affects what the consumers of this source compute. Only
// enqueue when a field they read changes:
//   - labels (region claim)
//   - spec.unschedulable, spec.taints, Ready and NetworkUnavailable condition
//     status (schedulable claim)
//   - spec.providerID, status.nodeInfo.architecture (node info claims)
//
// Informer resyncs (same resourceVersion) are always passed through, so the
// periodic refresh of claims that do not depend on nodes is kept.
func nodeChanged(oldObj, newObj interface{}) bool {
	oldNode, ok := oldObj.(*corev1.Node)
	if !ok {
		return true
	}
	newNode, ok := newObj.(*corev1.Node)
	if !ok {
		return true
	}

	if oldNode.ResourceVersion == newNode.ResourceVersion {
		return true
	}

	if !equality.Semantic.DeepEqual(oldNode.Labels, newNode.Labels) {
		return true
	}
	if oldNode.Spec.Unschedulable != newNode.Spec.Unschedulable ||
		oldNode.Spec.ProviderID != newNode.Spec.ProviderID ||
		oldNode.Status.NodeInfo.Architecture != newNode.Status.NodeInfo.Architecture {
		return true
	}
	if !equality.Semantic.DeepEqual(oldNode.Spec.Taints, newNode.Spec.Taints) {
		return true
	}
	for _, t := range []corev1.NodeConditionType{corev1.NodeReady, corev1.NodeNetworkUnavailable} {
		if conditionStatus(oldNode, t) != conditionStatus(newNode, t) {
			return true
		}
	}
	return false
}

// conditionStatus returns the status of the given node condition, or "" if the
// condition is not set.
func conditionStatus(node *corev1.Node, t corev1.NodeConditionType) corev1.ConditionStatus {
	for _, c := range node.Status.Conditions {
		if c.Type == t {
			return c.Status
		}
	}
	return ""
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
