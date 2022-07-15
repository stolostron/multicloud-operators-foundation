package utils

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetComponentNamespace(t *testing.T) {
	GetComponentNamespace()
}

func TestBuildKubeClient(t *testing.T) {
	BuildKubeClient("")
}

func TestResourceNamespacedName(t *testing.T) {
	ResourceNamespacedName("resourceType", "namespace", "name")
}

func TestClusterIsOffLine(t *testing.T) {
	ClusterIsOffLine([]metav1.Condition{})
}
