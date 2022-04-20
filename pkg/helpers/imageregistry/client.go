package imageregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stolostron/multicloud-operators-foundation/pkg/apis/imageregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

type Interface interface {
	Cluster(cluster *clusterv1.ManagedCluster) Interface
	PullSecret() (*corev1.Secret, error)
	ImageOverride(imageName string) string
}

type Client struct {
	kubeClient      kubernetes.Interface
	imageRegistries v1alpha1.ImageRegistries
}

func NewClient(kubeClient kubernetes.Interface) Interface {
	return &Client{
		kubeClient:      kubeClient,
		imageRegistries: v1alpha1.ImageRegistries{},
	}
}

func (c *Client) Cluster(cluster *clusterv1.ManagedCluster) Interface {
	annotations := cluster.GetAnnotations()
	if len(annotations) == 0 {
		c.imageRegistries = v1alpha1.ImageRegistries{}
		return c
	}

	_ = json.Unmarshal([]byte(annotations[v1alpha1.ClusterImageRegistriesAnnotation]), &c.imageRegistries)
	return c
}

func (c *Client) PullSecret() (*corev1.Secret, error) {
	if c.imageRegistries.PullSecret == "" {
		return nil, nil
	}
	segs := strings.Split(c.imageRegistries.PullSecret, ".")
	if len(segs) != 2 {
		return nil, fmt.Errorf("wrong pullSecret format %v in the annotation %s",
			c.imageRegistries.PullSecret, v1alpha1.ClusterImageRegistriesAnnotation)
	}
	namespace := segs[0]
	pullSecret := segs[1]
	return c.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), pullSecret, metav1.GetOptions{})
}

func (c *Client) ImageOverride(imageName string) string {
	if len(c.imageRegistries.Registries) == 0 {
		return imageName
	}
	overrideImageName := imageName
	for i := 0; i < len(c.imageRegistries.Registries); i++ {
		registry := c.imageRegistries.Registries[i]
		name := imageOverride(registry.Source, registry.Mirror, imageName)
		if name != imageName {
			overrideImageName = name
		}
	}
	return overrideImageName
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

func OverrideImageByAnnotation(annotations map[string]string, imageName string) string {
	if len(annotations) == 0 {
		return imageName
	}

	if annotations[v1alpha1.ClusterImageRegistriesAnnotation] == "" {
		return imageName
	}

	imageRegistries := v1alpha1.ImageRegistries{}
	err := json.Unmarshal([]byte(annotations[v1alpha1.ClusterImageRegistriesAnnotation]), &imageRegistries)
	if err != nil {
		klog.Errorf("failed to unmarshal the annotation %v,err %v", v1alpha1.ClusterImageRegistriesAnnotation, err)
		return imageName
	}

	if len(imageRegistries.Registries) == 0 {
		return imageName
	}
	overrideImageName := imageName
	for i := 0; i < len(imageRegistries.Registries); i++ {
		registry := imageRegistries.Registries[i]
		name := imageOverride(registry.Source, registry.Mirror, imageName)
		if name != imageName {
			overrideImageName = name
		}
	}
	return overrideImageName
}
