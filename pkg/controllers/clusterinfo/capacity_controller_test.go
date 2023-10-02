package clusterinfo

import (
	"context"
	"testing"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func newTestCapacityReconciler(existingObjs ...client.Object) (*CapacityReconciler, client.Client) {
	s := kubescheme.Scheme
	s.AddKnownTypes(clusterv1beta1.SchemeGroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	client := fake.NewClientBuilder().
		WithObjects(existingObjs...).WithStatusSubresource(existingObjs...).
		WithScheme(s).Build()
	return &CapacityReconciler{
		client: client,
		scheme: scheme,
	}, client
}

func TestCapacityReconcile(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name               string
		existingCluster    *clusterv1.ManagedCluster
		existinClusterInfo *clusterv1beta1.ManagedClusterInfo
		expectedCapacity   clusterv1.ResourceList
		expectedNotFound   bool
	}{
		{
			name:               "ManagedClusterNotFound",
			existingCluster:    newCluster("bar", nil),
			existinClusterInfo: newClusterInfo("bar", true, nil, 1),
			expectedNotFound:   true,
		},
		{
			name:               "ManagedClusterInfoNotFound",
			existingCluster:    newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 1}),
			existinClusterInfo: newClusterInfo("bar", true, nil, 1),
			expectedCapacity:   newCapacity(map[clusterv1.ResourceName]int64{"cpu": 1}),
		},
		{
			name:            "Do not update ocp Capacity",
			existingCluster: newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 1}),
			existinClusterInfo: newClusterInfo(ManagedClusterName, true,
				map[string]string{"node1": LabelNodeRoleOldControlPlane, "node2": LabelNodeRoleControlPlane}, 2),
			expectedCapacity: newCapacity(map[clusterv1.ResourceName]int64{"cpu": 1, "socket_worker": 0, "core_worker": 0}),
		},
		{
			name:            "Update ocp Capacity",
			existingCluster: newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 1}),
			existinClusterInfo: newClusterInfo(ManagedClusterName, true,
				map[string]string{"node1": LabelNodeRoleControlPlane, "node2": LabelNodeRoleInfra,
					"node3": "node-role.kubernetes.io/worker", "node4": ""}, 2),
			expectedCapacity: newCapacity(map[clusterv1.ResourceName]int64{"cpu": 1, "socket_worker": 4, "core_worker": 4}),
		},

		{
			name:            "Update non-ocp Capacity",
			existingCluster: newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 8}),
			existinClusterInfo: newClusterInfo(ManagedClusterName, false,
				map[string]string{"node1": LabelNodeRoleControlPlane, "node2": LabelNodeRoleInfra,
					"node3": "node-role.kubernetes.io/worker", "node4": ""}, 2),
			expectedCapacity: newCapacity(map[clusterv1.ResourceName]int64{"cpu": 8, "socket_worker": 0, "core_worker": 8}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc, client := newTestCapacityReconciler(test.existinClusterInfo, test.existingCluster)
			svrc.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: ManagedClusterName, Name: ManagedClusterName}})
			actualCluster := &clusterv1.ManagedCluster{}
			err := client.Get(context.Background(), types.NamespacedName{Name: ManagedClusterName}, actualCluster)
			switch {
			case errors.IsNotFound(err):
				if !test.expectedNotFound {
					t.Errorf("unexpected err %v", err)
				}
			case err != nil:
				t.Errorf("unexpected err %v", err)
			}
			if !apiequality.Semantic.DeepEqual(actualCluster.Status.Capacity, test.expectedCapacity) {
				t.Errorf("unexpected capacity %v, %v", actualCluster.Status.Capacity, test.expectedCapacity)
			}
		})
	}
}

func newCluster(name string, resources map[clusterv1.ResourceName]int64) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: clusterv1.ManagedClusterStatus{
			Conditions: []metav1.Condition{
				{
					Type:   clusterv1.ManagedClusterConditionAvailable,
					Status: metav1.ConditionTrue,
				},
			},
			Capacity: newCapacity(resources),
		},
	}
}

func newClusterInfo(name string, isOCP bool, nodes map[string]string, val int64) *clusterv1beta1.ManagedClusterInfo {
	info := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
		Status: clusterv1beta1.ClusterInfoStatus{
			NodeList: []clusterv1beta1.NodeStatus{},
		},
	}
	if isOCP {
		info.Status.DistributionInfo.Type = clusterv1beta1.DistributionTypeOCP
	}

	for nodeName, nodeRole := range nodes {
		node := clusterv1beta1.NodeStatus{
			Name: nodeName,
			Capacity: clusterv1beta1.ResourceList{
				"cpu":    *resource.NewQuantity(val, resource.DecimalSI),
				"socket": *resource.NewQuantity(val, resource.DecimalSI),
			},
		}
		node.Labels = map[string]string{
			nodeRole: "",
		}
		info.Status.NodeList = append(info.Status.NodeList, node)
	}

	return info
}

func newCapacity(resources map[clusterv1.ResourceName]int64) clusterv1.ResourceList {
	r := clusterv1.ResourceList{}

	for k, v := range resources {
		r[k] = *resource.NewQuantity(v, resource.DecimalSI)
	}

	return r
}

func Test_IsWorker(t *testing.T) {
	tests := []struct {
		name     string
		node     clusterv1beta1.NodeStatus
		expected bool
	}{
		{
			name: "no label",
			node: clusterv1beta1.NodeStatus{
				Name:   "",
				Labels: nil,
			},
			expected: true,
		},
		{
			name: "SNO",
			node: clusterv1beta1.NodeStatus{
				Name:   "",
				Labels: map[string]string{LabelNodeRoleOldControlPlane: "", LabelNodeRoleWorker: ""},
			},
			expected: true,
		},
		{
			name: "no worker label",
			node: clusterv1beta1.NodeStatus{
				Name:   "",
				Labels: map[string]string{"myLabel": ""},
			},
			expected: true,
		},
		{
			name: "infra node",
			node: clusterv1beta1.NodeStatus{
				Name:   "",
				Labels: map[string]string{LabelNodeRoleInfra: ""},
			},
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expected != isWorker(test.node) {
				t.Errorf("expected %v, but got %v", test.expected, isWorker(test.node))
			}
		})
	}
}
