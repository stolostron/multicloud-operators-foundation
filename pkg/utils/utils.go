package utils

import (
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"
)

func GetComponentNamespace() (string, error) {
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "open-cluster-management-agent-addon", err
	}
	return string(nsBytes), nil
}

func ClusterIsOffLine(conditions []metav1.Condition) bool {
	return meta.IsStatusConditionPresentAndEqual(conditions, clusterapiv1.ManagedClusterConditionAvailable, metav1.ConditionUnknown)
}
