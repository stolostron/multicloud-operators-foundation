package objectsource

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func readyNode() *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("8"),
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue, LastHeartbeatTime: metav1.Now()},
			},
		},
	}
}

func TestNodeEventAffectsClusterClaim(t *testing.T) {
	heartbeatOld := readyNode()
	heartbeatNew := readyNode()
	heartbeatNew.Status.Conditions[0].LastHeartbeatTime = metav1.Now()

	unschedulable := readyNode()
	unschedulable.Spec.Unschedulable = true

	notReady := readyNode()
	notReady.Status.Conditions[0].Status = corev1.ConditionFalse

	tainted := readyNode()
	tainted.Spec.Taints = []corev1.Taint{{Key: corev1.TaintNodeNotReady, Effect: corev1.TaintEffectNoExecute}}

	capacityChanged := readyNode()
	capacityChanged.Status.Capacity[corev1.ResourceCPU] = resource.MustParse("16")

	tests := []struct {
		name     string
		oldObj   interface{}
		newObj   interface{}
		expected bool
	}{
		{
			name:     "ignore kubelet heartbeat",
			oldObj:   heartbeatOld,
			newObj:   heartbeatNew,
			expected: false,
		},
		{
			name:     "reconcile unschedulable change",
			oldObj:   readyNode(),
			newObj:   unschedulable,
			expected: true,
		},
		{
			name:     "reconcile ready condition change",
			oldObj:   readyNode(),
			newObj:   notReady,
			expected: true,
		},
		{
			name:     "reconcile taint change",
			oldObj:   readyNode(),
			newObj:   tainted,
			expected: true,
		},
		{
			name:     "reconcile capacity change",
			oldObj:   readyNode(),
			newObj:   capacityChanged,
			expected: true,
		},
		{
			name:     "reconcile unexpected object type",
			oldObj:   "not-a-node",
			newObj:   readyNode(),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nodeEventAffectsClusterClaim(tt.oldObj, tt.newObj); got != tt.expected {
				t.Errorf("nodeEventAffectsClusterClaim() = %v, want %v", got, tt.expected)
			}
		})
	}
}
