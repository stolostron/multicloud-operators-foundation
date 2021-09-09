package imageregistry

import (
	corev1 "k8s.io/api/core/v1"
)

type Client interface {
	Cluster(clusterName string) Client
	Registry() (string, error)
	PullSecret() (*corev1.Secret, error)
	ImageOverride(imageName string) (string, error)
}
