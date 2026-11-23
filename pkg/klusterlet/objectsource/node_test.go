package objectsource

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestNode() *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "node1",
			ResourceVersion: "1",
			Labels:          map[string]string{corev1.LabelTopologyRegion: "us-east-1"},
		},
		Spec: corev1.NodeSpec{ProviderID: "aws:///us-east-1a/i-1"},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{Architecture: "amd64"},
			Conditions: []corev1.NodeCondition{
				{
					Type:              corev1.NodeReady,
					Status:            corev1.ConditionTrue,
					LastHeartbeatTime: metav1.Unix(100, 0),
				},
				{
					Type:              corev1.NodeMemoryPressure,
					Status:            corev1.ConditionFalse,
					LastHeartbeatTime: metav1.Unix(100, 0),
				},
			},
		},
	}
}

func TestNodeChanged(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(n *corev1.Node)
		// resync keeps the resource version, like an informer resync event.
		resync bool
		want   bool
	}{
		{
			name:   "resync with same resource version",
			mutate: func(n *corev1.Node) {},
			resync: true,
			want:   true,
		},
		{
			name: "heartbeat only",
			mutate: func(n *corev1.Node) {
				for i := range n.Status.Conditions {
					n.Status.Conditions[i].LastHeartbeatTime = metav1.Unix(200, 0)
				}
			},
			want: false,
		},
		{
			name: "images and allocatable change",
			mutate: func(n *corev1.Node) {
				n.Status.Images = []corev1.ContainerImage{{Names: []string{"quay.io/foo:v1"}}}
				n.Annotations = map[string]string{"foo": "bar"}
			},
			want: false,
		},
		{
			name: "non-schedulability condition flips",
			mutate: func(n *corev1.Node) {
				n.Status.Conditions[1].Status = corev1.ConditionTrue
			},
			want: false,
		},
		{
			name: "ready condition flips",
			mutate: func(n *corev1.Node) {
				n.Status.Conditions[0].Status = corev1.ConditionFalse
			},
			want: true,
		},
		{
			name: "network unavailable condition added",
			mutate: func(n *corev1.Node) {
				n.Status.Conditions = append(n.Status.Conditions, corev1.NodeCondition{
					Type:   corev1.NodeNetworkUnavailable,
					Status: corev1.ConditionTrue,
				})
			},
			want: true,
		},
		{
			name:   "cordoned",
			mutate: func(n *corev1.Node) { n.Spec.Unschedulable = true },
			want:   true,
		},
		{
			name: "taint added",
			mutate: func(n *corev1.Node) {
				n.Spec.Taints = []corev1.Taint{{Key: corev1.TaintNodeUnreachable, Effect: corev1.TaintEffectNoExecute}}
			},
			want: true,
		},
		{
			name:   "label changed",
			mutate: func(n *corev1.Node) { n.Labels[corev1.LabelTopologyRegion] = "us-west-2" },
			want:   true,
		},
		{
			name:   "provider id changed",
			mutate: func(n *corev1.Node) { n.Spec.ProviderID = "aws:///us-east-1a/i-2" },
			want:   true,
		},
		{
			name:   "architecture changed",
			mutate: func(n *corev1.Node) { n.Status.NodeInfo.Architecture = "arm64" },
			want:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			oldNode := newTestNode()
			newNode := oldNode.DeepCopy()
			c.mutate(newNode)
			if !c.resync {
				newNode.ResourceVersion = "2"
			}
			if got := nodeChanged(oldNode, newNode); got != c.want {
				t.Errorf("nodeChanged() = %v, want %v", got, c.want)
			}
		})
	}

	if !nodeChanged("not-a-node", newTestNode()) {
		t.Errorf("nodeChanged() should enqueue for unexpected object types")
	}
}
