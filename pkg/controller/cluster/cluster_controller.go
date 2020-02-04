// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package cluster

import (
	"time"

	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	hcmClientset "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	clientset "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	listers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	hcminformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	hcmlisters "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/listers_generated/mcm/v1alpha1"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
)

const offlineReason = "Klusterlet failed to update cluster status on time"

// Controller manages the lifecycle of cluster
type Controller struct {
	clusterclientset clientset.Interface

	clusterLister listers.ClusterLister

	clusterSynced     cache.InformerSynced
	hcmClientset      hcmClientset.Interface
	hcmWorkLister     hcmlisters.WorkLister
	hcmWorkSynced     cache.InformerSynced
	healthCheckPeriod time.Duration

	stopCh <-chan struct{}
}

// NewController returns a cluster Controller
func NewController(
	hcmClientset hcmClientset.Interface,
	hcmInformerFactory hcminformers.SharedInformerFactory,
	clusterclientset clientset.Interface,
	informerFactory informers.SharedInformerFactory,
	healthCheckPeriod time.Duration,
	stopCh <-chan struct{}) *Controller {
	clusterInformer := informerFactory.Clusterregistry().V1alpha1().Clusters()
	hcmWorkInformer := hcmInformerFactory.Mcm().V1alpha1().Works()
	controller := &Controller{
		hcmClientset:      hcmClientset,
		clusterclientset:  clusterclientset,
		hcmWorkLister:     hcmWorkInformer.Lister(),
		clusterLister:     clusterInformer.Lister(),
		clusterSynced:     clusterInformer.Informer().HasSynced,
		hcmWorkSynced:     hcmWorkInformer.Informer().HasSynced,
		stopCh:            stopCh,
		healthCheckPeriod: healthCheckPeriod,
	}
	return controller
}

// Run is the main run loop of kluster server
func (c *Controller) Run() {
	defer runtime.HandleCrash()

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for hcm informer caches to sync")
	if ok := cache.WaitForCacheSync(c.stopCh, c.clusterSynced, c.hcmWorkSynced); !ok {
		klog.Error("failed to wait for hcm caches to sync")
		return
	}

	// Start syncing cluster status immediately, this may set up things the runtime needs to run.
	go wait.Until(c.clusterHealthCheck, c.healthCheckPeriod, wait.NeverStop)

	<-c.stopCh
	klog.Info("Shutting controller")
}

func (c *Controller) clusterHealthCheck() {
	clusters, err := c.clusterLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list clusters: %v", err)
		return
	}

	for _, cluster := range clusters {
		// Skip of cluster status is already offline
		status, lastProbeTime := utils.GetStatusFromCluster(*cluster)
		if status != clusterv1alpha1.ClusterOK {
			continue
		}

		current := metav1.Now()
		if current.After(lastProbeTime.Add(c.healthCheckPeriod)) {
			// klusterlet does not update status on time, change status to offline
			cluster.Status.Conditions[0].Type = ""
			cluster.Status.Conditions[0].LastTransitionTime = current
			cluster.Status.Conditions[0].Reason = offlineReason

			_, err = c.clusterclientset.ClusterregistryV1alpha1().Clusters(cluster.Namespace).UpdateStatus(cluster)
			if err != nil {
				klog.Errorf("Failed to update cluster status %v", err)
			}
		}
	}
}
