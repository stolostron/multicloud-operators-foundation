package imageregistry

import (
	"fmt"
	"testing"

	"github.com/stolostron/multicloud-operators-foundation/pkg/apis/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	fakekube "k8s.io/client-go/kubernetes/fake"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

var (
	scheme = runtime.NewScheme()
)

const imageRegistryTemplate = `{
    "apiVersion": "imageregistry.open-cluster-management.io/v1alpha1",
    "kind": "ManagedClusterImageRegistry",
    "metadata": {
        "name": "imageRegistry",
        "namespace": "default"
    },
    "spec": {
        "placementRef": {
            "group": "cluster.open-cluster-management.io",
            "name": "placement",
            "resource": "placements"
        },
        "pullSecret": {
            "name": "pullSecret"
        },
        "registry": "quay.io/image"
    }
}`

func newImageRegistryObj(name, namespace, registryAddress, pullSecret string) *unstructured.Unstructured {
	obj := unstructured.Unstructured{}
	_ = obj.UnmarshalJSON([]byte(imageRegistryTemplate))
	_ = unstructured.SetNestedField(obj.Object, namespace, "metadata", "namespace")
	_ = unstructured.SetNestedField(obj.Object, name, "metadata", "name")
	_ = unstructured.SetNestedField(obj.Object, registryAddress, "spec", "registry")
	_ = unstructured.SetNestedField(obj.Object, pullSecret, "spec", "pullSecret", "name")

	return &obj
}

func fakeDynamicClient(cluster *clusterv1.ManagedCluster, secret *corev1.Secret, imageRegistry *unstructured.Unstructured) Client {
	fakeClusterClient := clusterfake.NewSimpleClientset(cluster)
	fakeKubeClient := fakekube.NewSimpleClientset(secret)
	fakeDynamicClient := dynamicfake.NewSimpleDynamicClient(scheme, imageRegistry)

	return &DynamicClient{
		clusterClient: fakeClusterClient,
		dynamicClient: fakeDynamicClient,
		kubeClient:    fakeKubeClient,
	}
}

func Test_DynamicClientPullSecret(t *testing.T) {
	testCases := []struct {
		name               string
		clusterName        string
		cluster            *clusterv1.ManagedCluster
		imageRegistry      *unstructured.Unstructured
		pullSecret         *corev1.Secret
		expectedErr        error
		expectedPullSecret *corev1.Secret
	}{
		{
			name:               "get correct pullSecret",
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"}),
			imageRegistry:      newImageRegistryObj("imageRegistry1", "ns1", "registryAddress1", "pullSecret1"),
			pullSecret:         newPullSecret("pullSecret1", "ns1", []byte("abc")),
			expectedErr:        nil,
			expectedPullSecret: newPullSecret("pullSecret1", "ns1", []byte("abc")),
		},
		{
			name:               "get cluster without ClusterImageRegistryLabel",
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", map[string]string{}),
			imageRegistry:      newImageRegistryObj("imageRegistry1", "ns1", "registryAddress1", "pullSecret1"),
			pullSecret:         newPullSecret("pullSecret1", "ns1", []byte("abc")),
			expectedErr:        nil,
			expectedPullSecret: nil,
		},
		{
			name:               "failed to get imageRegistry",
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"}),
			imageRegistry:      newImageRegistryObj("imageRegistry2", "ns2", "registryAddress1", "pullSecret1"),
			pullSecret:         newPullSecret("pullSecret1", "ns1", []byte("abc")),
			expectedErr:        fmt.Errorf("managedclusterimageregistries.imageregistry.open-cluster-management.io \"imageRegistry1\" not found"),
			expectedPullSecret: nil,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			client := fakeDynamicClient(c.cluster, c.pullSecret, c.imageRegistry)
			pullSecret, err := client.Cluster(c.clusterName).PullSecret()
			if c.expectedErr != nil && err == nil {
				t.Errorf("expected err %v, got nil", c.expectedErr)
			}
			if c.expectedErr == nil && err != nil {
				t.Errorf("expected no err, got %v", err)
			}

			if pullSecret == nil && c.expectedPullSecret != nil {
				t.Errorf("expected pullSecretData %+v,but got %+v", c.expectedPullSecret, pullSecret)
			}
			if pullSecret != nil {
				pullSecret.SetResourceVersion("")
				if !equality.Semantic.DeepEqual(c.expectedPullSecret, pullSecret) {
					t.Errorf("expected pullSecretData %#v,but got %#v", c.expectedPullSecret, pullSecret)
				}
			}
		})
	}
}

func Test_DynamicClientImageOverride(t *testing.T) {
	testCases := []struct {
		name          string
		image         string
		clusterName   string
		cluster       *clusterv1.ManagedCluster
		imageRegistry *unstructured.Unstructured
		expectedErr   error
		expectedImage string
	}{
		{
			name:          "override image successfully",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster1", map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"}),
			imageRegistry: newImageRegistryObj("imageRegistry1", "ns1", "192.163.1.1:5000/", "pullSecret1"),
			image:         "quay.io/acm-d/registry@SHA256aabc",
			expectedErr:   nil,
			expectedImage: "192.163.1.1:5000/registry@SHA256aabc",
		},
		{
			name:          "no registryAddress",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster1", map[string]string{}),
			imageRegistry: newImageRegistryObj("imageRegistry1", "ns1", "registryAddress1", "pullSecret1"),
			image:         "quay.io/acm-d/registry@SHA256aabc",
			expectedImage: "quay.io/acm-d/registry@SHA256aabc",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			client := fakeDynamicClient(c.cluster, &corev1.Secret{}, c.imageRegistry)
			newImage, err := client.Cluster(c.clusterName).ImageOverride(c.image)
			if c.expectedErr != nil && err == nil {
				t.Errorf("expected err %v, got nil", c.expectedErr)
			}
			if c.expectedErr == nil && err != nil {
				t.Errorf("expected no err, got %v", err)
			}
			if newImage != c.expectedImage {
				t.Errorf("execpted image %v, but got %v", c.expectedImage, newImage)
			}
		})
	}
}
