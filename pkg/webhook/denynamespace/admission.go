package denynamespace

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
)

var clusterDeploymentsGVR = schema.GroupVersionResource{
	Group:    "hive.openshift.io",
	Version:  "v1",
	Resource: "clusterdeployments",
}

var managedClustersGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
}

var manifestWorksGVR = schema.GroupVersionResource{
	Group:    "work.open-cluster-management.io",
	Version:  "v1",
	Resource: "manifestworks",
}

func ShouldDenyDeleteNamespace(ns string, dynamicClient dynamic.Interface) (bool, string) {
	var hasWorks = false
	var hasClusterDeployments = false
	var msg = ""

	klog.V(2).Infof("list clusterDeployments %+v ", ns)

	if ns == "" {
		klog.Errorf("the namespace is invalid")
		return false, msg
	}

	// check if the managedcluster exists
	_, err := dynamicClient.Resource(managedClustersGVR).Get(ns, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, msg
		}

		klog.Errorf("failed to get managed cluster %v. err: %v", ns, err)
		return false, msg
	}

	// check if the manifestworks exist
	works, err := dynamicClient.Resource(manifestWorksGVR).Namespace(ns).List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to list manifestworks in ns %v. err: %v", ns, err)
	} else if len(works.Items) != 0 {
		msg = fmt.Sprintf("deny deleting namespace %v since the manifestworks exist", ns)
		hasWorks = true
	}

	// check if the clusterdeployments exist
	cd, err := dynamicClient.Resource(clusterDeploymentsGVR).Namespace(ns).List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to list clusterdeployments in ns %v. err: %v", ns, err)
	} else if len(cd.Items) != 0 {
		msg = fmt.Sprintf("deny deleting namespace %v since the clusterdeployments exist", ns)
		hasClusterDeployments = true
	}

	if hasWorks && hasClusterDeployments {
		msg = fmt.Sprintf("deny deleting namespace %v since the manifestworks and clusterdeployments exist", ns)
	}

	return hasWorks || hasClusterDeployments, msg
}
