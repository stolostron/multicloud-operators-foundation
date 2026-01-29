package utils

import (
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
)

// GetComponentNamespace returns the namespace where the component is running.
// It reads from the service account namespace file, falling back to default if unavailable.
func GetComponentNamespace() (string, error) {
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "open-cluster-management-agent-addon", err
	}
	return string(nsBytes), nil
}

// ClusterIsOffLine checks if the cluster is offline based on its conditions.
// Returns true if the ManagedClusterConditionAvailable condition is Unknown.
func ClusterIsOffLine(conditions []metav1.Condition) bool {
	return meta.IsStatusConditionPresentAndEqual(conditions, clusterapiv1.ManagedClusterConditionAvailable, metav1.ConditionUnknown)
}
