// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rbac

import (
	"time"

	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	clusterlisters "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// ClusterRBACController is the controller to sync cluster role/rolebinding
type ClusterRBACController struct {
	clusterLister clusterlisters.ClusterLister
	clusterSyced  cache.InformerSynced
	kubeclientset kubernetes.Interface
	stopCh        <-chan struct{}
}

// NewClusterRBACController returns a new ClusterRBACController
func NewClusterRBACController(
	kubeclientset kubernetes.Interface,
	clusterInformerFactory clusterinformers.SharedInformerFactory,
	stopCh <-chan struct{}) *ClusterRBACController {
	clusterInformer := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters()
	return &ClusterRBACController{
		kubeclientset: kubeclientset,
		clusterLister: clusterInformer.Lister(),
		clusterSyced:  clusterInformer.Informer().HasSynced,
		stopCh:        stopCh,
	}
}

// Run is the main run loop of kluster server
func (cc *ClusterRBACController) Run() {
	defer runtime.HandleCrash()

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for cluster informer caches to sync")
	if ok := cache.WaitForCacheSync(cc.stopCh, cc.clusterSyced); !ok {
		klog.Errorf("failed to wait for kubernetes caches to sync")
		return
	}

	go wait.Until(cc.syncCluster, 5*time.Second, cc.stopCh)

	<-cc.stopCh
	klog.Info("Shutting controller")
}

func (cc *ClusterRBACController) syncCluster() {
	clusters, err := cc.clusterLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("failed to list clusters: %v", err)
	}

	for _, cluster := range clusters {
		owner := *metav1.NewControllerRef(cluster.DeepCopy(), clusterKind)
		err := CreateOrUpdateRole(cc.kubeclientset, cluster.Name, cluster.Namespace, owner)
		if err != nil {
			klog.Errorf("failed to update cluster role: %v", err)
		}

		err = CreateOrUpdateRoleBinding(cc.kubeclientset, cluster.Name, cluster.Namespace, owner)
		if err != nil {
			klog.Errorf("failed to update cluster rolebinding: %v", err)
		}
	}
}
