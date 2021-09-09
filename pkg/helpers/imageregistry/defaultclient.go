package imageregistry

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddToScheme(s *runtime.Scheme) error {
	if err := clusterv1.Install(s); err != nil {
		return err
	}
	if err := v1alpha1.AddToScheme(s); err != nil {
		return err
	}
	return nil
}

type DefaultClient struct {
	client  client.Client
	cluster string
}

func NewDefaultClient(client client.Client) Client {
	return &DefaultClient{
		client: client,
	}
}

func (c *DefaultClient) Cluster(clusterName string) Client {
	return &DefaultClient{
		client:  c.client,
		cluster: clusterName,
	}
}

// PullSecret returns the pullSecret.
// return nil if there is no imageRegistry of the cluster.
func (c *DefaultClient) PullSecret() (*corev1.Secret, error) {
	imageRegistry, err := c.getImageRegistry(c.cluster)
	if err != nil {
		return nil, err
	}
	if imageRegistry == nil {
		return nil, nil
	}

	pullSecretNamespace := imageRegistry.Namespace
	pullSecretName := imageRegistry.Spec.PullSecret.Name
	pullSecret := &corev1.Secret{}
	err = c.client.Get(context.TODO(), types.NamespacedName{Name: pullSecretName, Namespace: pullSecretNamespace}, pullSecret)
	if err != nil {
		return nil, err
	}

	return pullSecret, nil
}

// Registry returns the custom registry address.
// registry is empty if there is no imageRegistry of the cluster.
func (c *DefaultClient) Registry() (registry string, err error) {
	imageRegistry, err := c.getImageRegistry(c.cluster)
	if err != nil {
		return "", err
	}
	if imageRegistry == nil {
		return "", nil
	}
	registry = imageRegistry.Spec.Registry
	return registry, nil
}

// ImageOverride returns the overridden image.
// return the input image name if there is no custom registry.
func (c *DefaultClient) ImageOverride(imageName string) (newImageName string, err error) {
	customRegistry, err := c.Registry()
	if err != nil {
		return imageName, err
	}
	customRegistry = strings.TrimSuffix(customRegistry, "/")
	imageSegments := strings.Split(imageName, "/")
	if customRegistry == "" {
		return imageName, nil
	}
	newImageName = customRegistry + "/" + imageSegments[len(imageSegments)-1]
	return newImageName, nil
}

// getImageRegistry returns the custom imageRegistry.
// imageRegistry is nil if there is no imageRegistry of the cluster
func (c *DefaultClient) getImageRegistry(clusterName string) (*v1alpha1.ManagedClusterImageRegistry, error) {
	managedCluster := &clusterv1.ManagedCluster{}
	err := c.client.Get(context.TODO(), types.NamespacedName{Name: clusterName}, managedCluster)
	if err != nil {
		return nil, err
	}
	imageRegistryLabelValue := managedCluster.Labels[v1alpha1.ClusterImageRegistryLabel]
	if imageRegistryLabelValue == "" {
		return nil, nil
	}

	segments := strings.Split(imageRegistryLabelValue, ".")
	if len(segments) != 2 {
		err = fmt.Errorf("invalid format of image registry label value %v from cluster %v", imageRegistryLabelValue, clusterName)
		return nil, err
	}
	namespace := segments[0]
	imageRegistryName := segments[1]
	imageRegistry := &v1alpha1.ManagedClusterImageRegistry{}
	err = c.client.Get(context.TODO(), types.NamespacedName{Name: imageRegistryName, Namespace: namespace}, imageRegistry)
	if err != nil {
		return nil, err
	}

	return imageRegistry, nil
}
