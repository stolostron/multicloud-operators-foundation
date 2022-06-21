package imageregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DefaultInterface interface {
	Cluster(clusterName string) DefaultInterface
	PullSecret() (*corev1.Secret, error)
	ImageOverride(imageName string) (string, error)
}

func AddToScheme(s *runtime.Scheme) error {
	if err := clusterv1.Install(s); err != nil {
		return err
	}

	return nil
}

type DefaultClient struct {
	client  client.Client
	cluster string
}

func NewDefaultClient(client client.Client) DefaultInterface {
	return &DefaultClient{
		client: client,
	}
}

func (c *DefaultClient) Cluster(clusterName string) DefaultInterface {
	return &DefaultClient{
		client:  c.client,
		cluster: clusterName,
	}
}

// PullSecret returns the pullSecret.
// return nil if there is no imageRegistry of the cluster.
func (c *DefaultClient) PullSecret() (*corev1.Secret, error) {
	imageRegistries, err := c.getImageRegistries(c.cluster)
	if err != nil {
		return nil, err
	}
	segs := strings.Split(imageRegistries.PullSecret, ".")
	if len(segs) != 2 {
		return nil, fmt.Errorf("wrong pullSecret format %v in the annotation %s",
			imageRegistries.PullSecret, v1alpha1.ClusterImageRegistriesAnnotation)
	}
	namespace := segs[0]
	pullSecretName := segs[1]

	pullSecret := &corev1.Secret{}
	err = c.client.Get(context.TODO(), types.NamespacedName{Name: pullSecretName, Namespace: namespace}, pullSecret)
	if err != nil {
		return nil, err
	}

	return pullSecret, nil
}

// ImageOverride returns the overridden image.
// return the input image name if there is no custom registry.
func (c *DefaultClient) ImageOverride(imageName string) (newImageName string, err error) {
	imageRegistries, err := c.getImageRegistries(c.cluster)
	if err != nil {
		return imageName, err
	}

	if len(imageRegistries.Registries) == 0 {
		return imageName, nil
	}
	overrideImageName := imageName
	for i := 0; i < len(imageRegistries.Registries); i++ {
		registry := imageRegistries.Registries[i]
		name := imageOverride(registry.Source, registry.Mirror, imageName)
		if name != imageName {
			overrideImageName = name
		}
	}
	return overrideImageName, nil
}

// getImageRegistries retrieves the imageRegistries from annotations of managedCluster
func (c *DefaultClient) getImageRegistries(clusterName string) (v1alpha1.ImageRegistries, error) {
	imageRegistries := v1alpha1.ImageRegistries{}
	managedCluster := &clusterv1.ManagedCluster{}
	err := c.client.Get(context.TODO(), types.NamespacedName{Name: clusterName}, managedCluster)
	if err != nil {
		return imageRegistries, err
	}
	annotations := managedCluster.GetAnnotations()
	if len(annotations) == 0 {
		return imageRegistries, nil
	}

	if annotations[v1alpha1.ClusterImageRegistriesAnnotation] == "" {
		return imageRegistries, nil
	}

	err = json.Unmarshal([]byte(annotations[v1alpha1.ClusterImageRegistriesAnnotation]), &imageRegistries)
	return imageRegistries, err
}
