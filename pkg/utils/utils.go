package utils

import (
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapiv1 "open-cluster-management.io/api/cluster/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

func GetComponentNamespace() (string, error) {
	nsBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "open-cluster-management-agent-addon", err
	}
	return string(nsBytes), nil
}

func BuildKubeClient(kubeConfigPath string) (*kubernetes.Clientset, error) {
	hubRestConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		klog.Errorf("failed to build kubeconfig. Error:%v", err)
		return nil, err
	}
	return kubernetes.NewForConfig(hubRestConfig)
}

func ResourceNamespacedName(resourceType, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", resourceType, namespace, name)
}

func ClusterIsOffLine(conditions []metav1.Condition) bool {
	return meta.IsStatusConditionPresentAndEqual(conditions, clusterapiv1.ManagedClusterConditionAvailable, metav1.ConditionUnknown)
}
