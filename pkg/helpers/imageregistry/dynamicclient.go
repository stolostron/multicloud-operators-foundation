package imageregistry

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
)

var imageRegistryGVR = schema.GroupVersionResource{
	Group:    "imageregistry.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "managedclusterimageregistries",
}

type DynamicClient struct {
	clusterClient clusterclient.Interface
	dynamicClient dynamic.Interface
	kubeClient    kubernetes.Interface
	cluster       string
}

func NewDynamicClient(kubeConfig *rest.Config) (Client, error) {
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	clusterClient, err := clusterclient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return &DynamicClient{
		clusterClient: clusterClient,
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}, nil
}

func (c *DynamicClient) Cluster(clusterName string) Client {
	return &DynamicClient{
		clusterClient: c.clusterClient,
		kubeClient:    c.kubeClient,
		dynamicClient: c.dynamicClient,
		cluster:       clusterName,
	}
}

// Registry returns the custom registry address.
// registry is empty if there is no imageRegistry of the cluster.
func (c *DynamicClient) Registry() (string, error) {
	imageRegistry, err := c.getImageRegistry()
	if err != nil {
		return "", err
	}

	if imageRegistry == nil {
		return "", nil
	}

	registry, found, err := unstructured.NestedString(imageRegistry.Object, "spec", "registry")
	if err != nil || !found {
		return "", fmt.Errorf("failed to get registry in imageRegistry. %v", err)
	}

	return registry, nil
}

// PullSecret returns the pullSecret.
// return nil if there is no imageRegistry of the cluster.
func (c *DynamicClient) PullSecret() (*corev1.Secret, error) {
	imageRegistry, err := c.getImageRegistry()
	if err != nil {
		return nil, err
	}

	if imageRegistry == nil {
		return nil, nil
	}

	namespace := imageRegistry.GetNamespace()

	pullSecretName, found, err := unstructured.NestedString(imageRegistry.Object, "spec", "pullSecret", "name")
	if err != nil || !found {
		return nil, fmt.Errorf("failed to get pullSecret in imageRegistry. %v", err)
	}

	pullSecret, err := c.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), pullSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pullSecret, nil
}

// ImageOverride returns the overridden image.
// return the input image name if there is no custom registry.
func (c *DynamicClient) ImageOverride(image string) (string, error) {
	customRegistry, err := c.Registry()
	if err != nil {
		return image, err
	}
	customRegistry = strings.TrimSuffix(customRegistry, "/")
	imageSegments := strings.Split(image, "/")
	if customRegistry == "" {
		return image, nil
	}
	newImage := customRegistry + "/" + imageSegments[len(imageSegments)-1]
	return newImage, nil
}

// getImageRegistry returns the custom imageRegistry.
// imageRegistry is nil if there is no imageRegistry of the cluster
func (c *DynamicClient) getImageRegistry() (*unstructured.Unstructured, error) {
	managedCluster, err := c.clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), c.cluster, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	imageRegistryLabelValue := managedCluster.Labels[v1alpha1.ClusterImageRegistryLabel]
	if imageRegistryLabelValue == "" {
		return nil, nil
	}

	segments := strings.Split(imageRegistryLabelValue, ".")
	if len(segments) != 2 {
		err = fmt.Errorf("invalid format of image registry label value %v from cluster %v", imageRegistryLabelValue, c.cluster)
		return nil, err
	}
	namespace := segments[0]
	imageRegistryName := segments[1]
	imageRegistry, err := c.dynamicClient.Resource(imageRegistryGVR).Namespace(namespace).
		Get(context.TODO(), imageRegistryName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return imageRegistry, nil
}
