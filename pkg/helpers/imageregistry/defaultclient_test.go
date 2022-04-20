package imageregistry

import (
	"fmt"
	"testing"

	"github.com/stolostron/multicloud-operators-foundation/pkg/apis/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_DefaultClientPullSecret(t *testing.T) {
	testCases := []struct {
		name               string
		pullSecret         *corev1.Secret
		clusterName        string
		cluster            *clusterv1.ManagedCluster
		expectedErr        error
		expectedPullSecret *corev1.Secret
	}{
		{
			name:               "get correct pullSecret",
			pullSecret:         newPullSecret("ns1", "pullSecret"),
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", newAnnotationRegistries(nil, "ns1.pullSecret")),
			expectedErr:        nil,
			expectedPullSecret: newPullSecret("ns1", "pullSecret"),
		},
		{
			name:               "failed to get pullSecret without annotation",
			pullSecret:         newPullSecret("ns1", "pullSecret"),
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", ""),
			expectedErr:        fmt.Errorf("wrong pullSecret format  in the annotation %s", v1alpha1.ClusterImageRegistriesAnnotation),
			expectedPullSecret: nil,
		},
		{
			name:               "failed to get pullSecret with wrong annotation",
			pullSecret:         newPullSecret("ns1", "pullSecret"),
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", "abc"),
			expectedErr:        fmt.Errorf("invalid character 'a' looking for beginning of value"),
			expectedPullSecret: nil,
		},
		{
			name:               "failed to get pullSecret with wrong cluster",
			pullSecret:         newPullSecret("ns1", "pullSecret"),
			clusterName:        "cluster1",
			cluster:            newCluster("cluster2", ""),
			expectedErr:        fmt.Errorf(`managedclusters.cluster.open-cluster-management.io "cluster1" not found`),
			expectedPullSecret: nil,
		},
		{
			name:               "failed to get pullSecret without pullSecret",
			pullSecret:         newPullSecret("ns1", "pullSecret"),
			clusterName:        "cluster1",
			cluster:            newCluster("cluster1", newAnnotationRegistries(nil, "ns.test")),
			expectedErr:        fmt.Errorf("secrets \"test\" not found"),
			expectedPullSecret: nil,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Secret{})
			_ = AddToScheme(scheme)
			client := NewDefaultClient(fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c.cluster, c.pullSecret).Build())
			pullSecret, err := client.Cluster(c.clusterName).PullSecret()
			if err != nil && c.expectedErr != nil {
				if err.Error() != c.expectedErr.Error() {
					t.Errorf("expected err %v, but got %v", c.expectedErr, err)
				}
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
		expectedImage string
		expectedErr   error
	}{
		{
			name:        "override rhacm2 image ",
			clusterName: "cluster1",
			cluster: newCluster("cluster1", newAnnotationRegistries([]v1alpha1.Registries{
				{Source: "registry.redhat.io/rhacm2", Mirror: "quay.io/rhacm2"},
				{Source: "registry.redhat.io/multicluster-engine", Mirror: "quay.io/multicluster-engine"}}, "")),
			image:         "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedImage: "quay.io/rhacm2/registration@SHA256abc",
			expectedErr:   nil,
		},
		{
			name:        "override acm-d image",
			clusterName: "cluster1",
			cluster: newCluster("cluster1", newAnnotationRegistries([]v1alpha1.Registries{
				{Source: "registry.redhat.io/rhacm2", Mirror: "quay.io/rhacm2"},
				{Source: "registry.redhat.io/multicluster-engine", Mirror: "quay.io/multicluster-engine"}}, "")),
			image:         "registry.redhat.io/acm-d/registration@SHA256abc",
			expectedImage: "registry.redhat.io/acm-d/registration@SHA256abc",
			expectedErr:   nil,
		},
		{
			name:        "override multicluster-engine image",
			clusterName: "cluster1",
			cluster: newCluster("cluster1", newAnnotationRegistries([]v1alpha1.Registries{
				{Source: "registry.redhat.io/rhacm2", Mirror: "quay.io/rhacm2"},
				{Source: "registry.redhat.io/multicluster-engine", Mirror: "quay.io/multicluster-engine"}}, "")),
			image:         "registry.redhat.io/multicluster-engine/registration@SHA256abc",
			expectedImage: "quay.io/multicluster-engine/registration@SHA256abc",
			expectedErr:   nil,
		},
		{
			name:        "override image without source ",
			clusterName: "cluster1",
			cluster: newCluster("cluster1", newAnnotationRegistries([]v1alpha1.Registries{
				{Source: "", Mirror: "quay.io/rhacm2"}}, "")),
			image:         "registry.redhat.io/multicluster-engine/registration@SHA256abc",
			expectedImage: "quay.io/rhacm2/registration@SHA256abc",
			expectedErr:   nil,
		},
		{
			name:        "override image",
			clusterName: "cluster1",
			cluster: newCluster("cluster1", newAnnotationRegistries([]v1alpha1.Registries{
				{Source: "registry.redhat.io/rhacm2", Mirror: "quay.io/rhacm2"},
				{Source: "registry.redhat.io/rhacm2/registration@SHA256abc",
					Mirror: "quay.io/acm-d/registration:latest"}}, "")),
			image:         "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedImage: "quay.io/acm-d/registration:latest",
			expectedErr:   nil,
		},
		{
			name:          "return image without annotation",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster1", ""),
			image:         "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedImage: "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedErr:   nil,
		},
		{
			name:          "return image with wrong annotation",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster1", "abc"),
			image:         "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedImage: "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedErr:   fmt.Errorf("invalid character 'a' looking for beginning of value"),
		},
		{
			name:          "return image without cluster",
			clusterName:   "cluster1",
			cluster:       newCluster("cluster2", ""),
			image:         "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedImage: "registry.redhat.io/rhacm2/registration@SHA256abc",
			expectedErr:   fmt.Errorf(`managedclusters.cluster.open-cluster-management.io "cluster1" not found`),
		},
	}

	pullSecret := newPullSecret("n1", "s1")
	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Secret{})
			_ = AddToScheme(scheme)

			client := NewDefaultClient(fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(c.cluster, pullSecret).Build())
			newImage, err := client.Cluster(c.clusterName).ImageOverride(c.image)
			if err != nil && c.expectedErr != nil {
				if err.Error() != c.expectedErr.Error() {
					t.Errorf("expected err %v, but got %v", c.expectedErr, err)
				}
			}
			if newImage != c.expectedImage {
				t.Errorf("execpted image %v, but got %v", c.expectedImage, newImage)
			}
		})
	}
}
