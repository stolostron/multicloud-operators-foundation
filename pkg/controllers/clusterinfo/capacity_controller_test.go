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
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func newTestCapacityReconciler(existingObjs ...runtime.Object) (*CapacityReconciler, client.Client) {
	s := kubescheme.Scheme
	s.AddKnownTypes(clusterv1beta1.SchemeGroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	client := fake.NewFakeClientWithScheme(s, existingObjs...)
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
			existinClusterInfo: newClusterInfo("bar", nil, 1),
			expectedNotFound:   true,
		},
		{
			name:               "ManagedClusterInfoNotFound",
			existingCluster:    newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 1}),
			existinClusterInfo: newClusterInfo("bar", nil, 1),
			expectedCapacity:   newCapacity(map[clusterv1.ResourceName]int64{"cpu": 1}),
		},
		{
			name:               "UpdateManagedClusterCapacity",
			existingCluster:    newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 1}),
			existinClusterInfo: newClusterInfo(ManagedClusterName, map[string]bool{"node1": false}, 2),
			expectedCapacity:   newCapacity(map[clusterv1.ResourceName]int64{"cpu": 1, "socket_worker": 0, "core_worker": 0}),
		},
		{
			name:               "UpdateManagedClusterCapacityWithWorker",
			existingCluster:    newCluster(ManagedClusterName, map[clusterv1.ResourceName]int64{"cpu": 1}),
			existinClusterInfo: newClusterInfo(ManagedClusterName, map[string]bool{"node1": false, "node2": true}, 2),
			expectedCapacity:   newCapacity(map[clusterv1.ResourceName]int64{"cpu": 1, "socket_worker": 2, "core_worker": 2}),
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
			Capacity: newCapacity(resources),
		},
	}
}

func newClusterInfo(name string, resources map[string]bool, val int64) *clusterv1beta1.ManagedClusterInfo {
	info := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
		Status: clusterv1beta1.ClusterInfoStatus{
			NodeList: []clusterv1beta1.NodeStatus{},
		},
	}

	for name, isworker := range resources {
		node := clusterv1beta1.NodeStatus{
			Name: name,
			Capacity: clusterv1beta1.ResourceList{
				"cpu":    *resource.NewQuantity(val, resource.DecimalSI),
				"socket": *resource.NewQuantity(val, resource.DecimalSI),
			},
		}
		if isworker {
			node.Labels = map[string]string{
				"node-role.kubernetes.io/worker": "",
			}
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
