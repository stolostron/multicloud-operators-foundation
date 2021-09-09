package imageregistry

import (
	"fmt"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newCluster(name string, labels map[string]string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

func newImageRegistry(name, namespace, registryAddress, pullSecret string) *v1alpha1.ManagedClusterImageRegistry {
	return &v1alpha1.ManagedClusterImageRegistry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ImageRegistrySpec{
			Registry:     registryAddress,
			PullSecret:   corev1.LocalObjectReference{Name: pullSecret},
			PlacementRef: v1alpha1.PlacementRef{},
		},
	}
}

func newPullSecret(name, namespace string, data []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: data,
		},
		StringData: nil,
		Type:       corev1.SecretTypeDockerConfigJson,
	}
}

func Test_DefaultClientPullSecret(t *testing.T) {
	testCases := []struct {
		name               string
		clusterName        string
		cluster            *clusterv1.ManagedCluster
		imageRegistry      *v1alpha1.ManagedClusterImageRegistry
		pullSecret         *corev1.Secret
		expectedErr        error
		expectedPullSecret *corev1.Secret
	}{
		{
			name:               "get correct pullSecret",
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"}),
			imageRegistry:      newImageRegistry("imageRegistry1", "ns1", "registryAddress1", "pullSecret1"),
			pullSecret:         newPullSecret("pullSecret1", "ns1", []byte("abc")),
			expectedErr:        nil,
			expectedPullSecret: newPullSecret("pullSecret1", "ns1", []byte("abc")),
		},
		{
			name:               "get cluster without ClusterImageRegistryLabel",
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", map[string]string{}),
			imageRegistry:      newImageRegistry("imageRegistry1", "ns1", "registryAddress1", "pullSecret1"),
			pullSecret:         newPullSecret("pullSecret1", "ns1", []byte("abc")),
			expectedErr:        nil,
			expectedPullSecret: nil,
		},
		{
			name:               "failed to get imageRegistry",
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"}),
			imageRegistry:      newImageRegistry("imageRegistry2", "ns2", "registryAddress1", "pullSecret1"),
			pullSecret:         newPullSecret("pullSecret1", "ns1", []byte("abc")),
			expectedErr:        fmt.Errorf("managedclusterimageregistries.imageregistry.open-cluster-management.io \"imageRegistry1\" not found"),
			expectedPullSecret: nil,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Secret{})
			_ = AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder()
			client := NewDefaultClient(fakeClient.WithScheme(scheme).WithRuntimeObjects(c.imageRegistry, c.cluster, c.pullSecret).Build())
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

func Test_DefaultClientImageOverride(t *testing.T) {
	testCases := []struct {
		name          string
		image         string
		clusterName   string
		cluster       *clusterv1.ManagedCluster
		imageRegistry *v1alpha1.ManagedClusterImageRegistry
		expectedErr   error
		expectedImage string
	}{
		{
			name:          "override image successfully",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster1", map[string]string{v1alpha1.ClusterImageRegistryLabel: "ns1.imageRegistry1"}),
			imageRegistry: newImageRegistry("imageRegistry1", "ns1", "192.163.1.1:5000/", "pullSecret1"),
			image:         "quay.io/acm-d/registry@SHA256aabc",
			expectedErr:   nil,
			expectedImage: "192.163.1.1:5000/registry@SHA256aabc",
		},
		{
			name:          "no registryAddress",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster1", map[string]string{}),
			imageRegistry: newImageRegistry("imageRegistry1", "ns1", "registryAddress1", "pullSecret1"),
			image:         "quay.io/acm-d/registry@SHA256aabc",
			expectedImage: "quay.io/acm-d/registry@SHA256aabc",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = AddToScheme(scheme)
			fakeClient := fake.NewClientBuilder()
			client := NewDefaultClient(fakeClient.WithScheme(scheme).WithRuntimeObjects(c.cluster, c.imageRegistry).Build())
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
