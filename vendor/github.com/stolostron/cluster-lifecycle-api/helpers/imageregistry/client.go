package imageregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

type Interface interface {
	Cluster(cluster *clusterv1.ManagedCluster) Interface
	PullSecret() (*corev1.Secret, error)
	ImageOverride(imageName string) (string, error)
}

type Client struct {
	kubeClient kubernetes.Interface
	cluster    *clusterv1.ManagedCluster
}

func NewClient(kubeClient kubernetes.Interface) Interface {
	return &Client{
		kubeClient: kubeClient,
	}
}

func (c *Client) Cluster(cluster *clusterv1.ManagedCluster) Interface {
	return &Client{kubeClient: c.kubeClient, cluster: cluster}
}

func (c *Client) PullSecret() (*corev1.Secret, error) {
	imageRegistries, err := c.getImageRegistries()
	if err != nil {
		return nil, err
	}

	if imageRegistries.PullSecret == "" {
		return nil, nil
	}

	segs := strings.Split(imageRegistries.PullSecret, ".")
	if len(segs) != 2 {
		return nil, fmt.Errorf("wrong pullSecret format %v in the annotation %s",
			imageRegistries.PullSecret, v1alpha1.ClusterImageRegistriesAnnotation)
	}
	namespace := segs[0]
	pullSecret := segs[1]
	return c.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), pullSecret, metav1.GetOptions{})
}

// ImageOverride is to override the image by image-registries annotation of managedCluster.
// The source registry will be replaced by the Mirror.
// The larger index will work if the Sources are the same.
func (c *Client) ImageOverride(imageName string) (string, error) {
	imageRegistries, err := c.getImageRegistries()
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

func imageOverride(source, mirror, imageName string) string {
	source = strings.TrimSuffix(source, "/")
	mirror = strings.TrimSuffix(mirror, "/")
	imageSegments := strings.Split(imageName, "/")
	imageNameTag := imageSegments[len(imageSegments)-1]
	if source == "" {
		if mirror == "" {
			return imageNameTag
		}
		return fmt.Sprintf("%s/%s", mirror, imageNameTag)
	}

	if !strings.HasPrefix(imageName, source) {
		return imageName
	}

	trimSegment := strings.TrimPrefix(imageName, source)
	return fmt.Sprintf("%s%s", mirror, trimSegment)
}

func (c *Client) getImageRegistries() (v1alpha1.ImageRegistries, error) {
	imageRegistries := v1alpha1.ImageRegistries{}
	if c.cluster == nil {
		return imageRegistries, fmt.Errorf("the managedCluster cannot be nil")
	}
	annotations := c.cluster.GetAnnotations()
	if len(annotations) == 0 {
		return imageRegistries, nil
	}

	if _, ok := annotations[v1alpha1.ClusterImageRegistriesAnnotation]; !ok {
		return imageRegistries, nil
	}

	err := json.Unmarshal([]byte(annotations[v1alpha1.ClusterImageRegistriesAnnotation]), &imageRegistries)
	return imageRegistries, err
}

// OverrideImageByAnnotation is to override the image by image-registries annotation of managedCluster.
// The source registry will be replaced by the Mirror.
// The larger index will work if the Sources are the same.
func OverrideImageByAnnotation(annotations map[string]string, imageName string) (string, error) {
	if len(annotations) == 0 {
		return imageName, nil
	}

	if _, ok := annotations[v1alpha1.ClusterImageRegistriesAnnotation]; !ok {
		return imageName, nil
	}

	imageRegistries := v1alpha1.ImageRegistries{}
	err := json.Unmarshal([]byte(annotations[v1alpha1.ClusterImageRegistriesAnnotation]), &imageRegistries)
	if err != nil {
		klog.Errorf("failed to unmarshal the annotation %v,err %v", annotations[v1alpha1.ClusterImageRegistriesAnnotation], err)
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
