package denynamespace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

var (
	scheme = runtime.NewScheme()
)

func newManagedCluster(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.open-cluster-management.io/v1",
			"kind":       "ManagedCluster",
			"metadata": map[string]interface{}{
				"name": name,
			},
		},
	}
}
func newManifestWork(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "work.open-cluster-management.io/v1",
			"kind":       "ManifestWork",
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

func newClusterDeployment(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "hive.openshift.io/v1",
			"kind":       "ClusterDeployment",
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

func Test_ShouldDenyDeleteNamespace(t *testing.T) {
	gvrToListKind := map[schema.GroupVersionResource]string{
		clusterDeploymentsGVR: "ClusterDeploymentsList",
		managedClustersGVR:    "ManagedClustersList",
		manifestWorksGVR:      "ManifestWorkList",
	}
	tests := []struct {
		name           string
		namespace      string
		dynamicClient  dynamic.Interface
		expectedResult bool
		expectdMsg     string
	}{
		{
			name:           "case: ns is null",
			namespace:      "",
			expectedResult: false,
			expectdMsg:     "",
		},
		{
			name:           "case: there is no managedcluster",
			namespace:      "cluster1",
			dynamicClient:  dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, newManifestWork("cluster1", "work")),
			expectedResult: false,
			expectdMsg:     "",
		},
		{
			name:           "case: there is no resource in ns",
			namespace:      "cluster1",
			dynamicClient:  dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, newManagedCluster("cluster1")),
			expectedResult: false,
			expectdMsg:     "",
		},
		{
			name:      "case: there is only manifestwork",
			namespace: "cluster2",
			dynamicClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind,
				newManagedCluster("cluster2"), newManifestWork("cluster2", "work")),
			expectedResult: true,
			expectdMsg:     "deny deleting namespace cluster2 since the manifestworks exist",
		},
		{
			name:      "case: there is only clusterdeployment",
			namespace: "cluster3",
			dynamicClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind,
				newManagedCluster("cluster3"), newClusterDeployment("cluster3", "clusterdeployment")),
			expectedResult: true,
			expectdMsg:     "deny deleting namespace cluster3 since the clusterdeployments exist",
		},
		{
			name:      "case: there are both manifestwork and clusterdeployment",
			namespace: "cluster4",
			dynamicClient: dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind,
				newManagedCluster("cluster4"), newManifestWork("cluster4", "work"), newClusterDeployment("cluster4", "clusterdeployment")),
			expectedResult: true,
			expectdMsg:     "deny deleting namespace cluster4 since the manifestworks and clusterdeployments exist",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, msg := ShouldDenyDeleteNamespace(test.namespace, test.dynamicClient)
			assert.Equal(t, test.expectedResult, result)
			assert.Equal(t, test.expectdMsg, msg)
		})
	}
}
