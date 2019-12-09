// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package controller

import (
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// PluginController object
type PluginController struct {
	hubKubeClient      kubernetes.Interface
	hubEndpointsLister listers.EndpointsLister
	clusterNamespace   string
	clusterName        string
	plugin             plugin.Plugin
	syncPeriod         int
	queue              workqueue.RateLimitingInterface
	stopCh             <-chan struct{}
}

type eventType int

const (
	add eventType = iota
	update
	delete
)

type registryEvent struct {
	endpoints *v1.Endpoints
	eventType eventType
}

// NewPluginController creates plugin controller
func NewPluginController(hubKubeClient kubernetes.Interface, hubInformerFactory informers.SharedInformerFactory,
	clusterNamespace, clusterName string,
	plugin plugin.Plugin,
	syncPeriod int,
	stopCh <-chan struct{}) *PluginController {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	plugin.RegisterAnnotatedResouceHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoints, ok := obj.(*v1.Endpoints)
			if ok {
				queue.Add(registryEvent{
					endpoints: endpoints,
					eventType: add,
				})
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldEndpoints, oldOk := old.(*v1.Endpoints)
			newEndpoints, newOk := new.(*v1.Endpoints)
			if oldOk && newOk && !utils.NeedToUpdateEndpoints(oldEndpoints, newEndpoints) {
				queue.Add(registryEvent{
					endpoints: newEndpoints,
					eventType: update,
				})
			}
		},
		DeleteFunc: func(obj interface{}) {
			endpoints, ok := obj.(*v1.Endpoints)
			if ok {
				queue.Add(registryEvent{
					endpoints: endpoints,
					eventType: delete,
				})
			}
		},
	})

	return &PluginController{
		hubKubeClient:      hubKubeClient,
		hubEndpointsLister: hubInformerFactory.Core().V1().Endpoints().Lister(),
		clusterNamespace:   clusterNamespace,
		clusterName:        clusterName,
		plugin:             plugin,
		syncPeriod:         syncPeriod,
		queue:              queue,
		stopCh:             stopCh,
	}
}

// Run starts this controller
func (c *PluginController) Run() {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("%s plugin controller is ready\n", c.plugin.GetType())

	go wait.Until(c.runWorker, time.Second, c.stopCh)

	period := time.Duration(c.syncPeriod) * time.Second
	go wait.Until(c.syncRegisteredEndpoints, period, c.stopCh)
	<-c.stopCh
	klog.Infof("%s plugin controller is stop\n", c.plugin.GetType())
}

func (c *PluginController) runWorker() {
	for c.processRegistryEvent() {
	}
}

func (c *PluginController) processRegistryEvent() bool {
	event, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(event)
	newEvent := event.(registryEvent)
	switch newEvent.eventType {
	case add:
		c.createEndpoints(newEvent.endpoints)
	case update:
		c.updateEndpoints(newEvent.endpoints)
	case delete:
		c.deleteEndpoints(newEvent.endpoints)
	}
	c.queue.Forget(event)
	return true
}

func (c *PluginController) syncRegisteredEndpoints() {
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{
			utils.ServiceTypeLabel: c.plugin.GetType(),
			utils.ClusterLabel:     c.clusterName,
		},
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: utils.AutoDiscoveryLabel, Operator: metav1.LabelSelectorOpDoesNotExist},
		},
	})
	endpoints, err := c.hubEndpointsLister.Endpoints(c.clusterNamespace).List(selector)
	if err != nil {
		return
	}
	toCreate, toDelete, toUpdate := c.plugin.SyncRegisteredEndpoints(endpoints)
	for _, ep := range toCreate {
		c.createEndpoints(ep)
	}
	for _, ep := range toDelete {
		c.deleteEndpoints(ep)
	}
	for _, ep := range toUpdate {
		c.updateEndpoints(ep)
	}
}

func (c *PluginController) createEndpoints(endpoints *v1.Endpoints) {
	_, err := c.hubKubeClient.CoreV1().Endpoints(endpoints.Namespace).Create(endpoints)
	if err != nil {
		klog.Errorf("failed to register endpoint (%s/%s) in hub cluster, %v", endpoints.Namespace, endpoints.Name, err)
		return
	}
	klog.V(5).Infof("register endpoints (%s/%s) in hub cluster", endpoints.Namespace, endpoints.Name)
}

func (c *PluginController) updateEndpoints(endpoints *v1.Endpoints) {
	_, err := c.hubKubeClient.CoreV1().Endpoints(endpoints.Namespace).Update(endpoints)
	if err != nil {
		klog.Errorf("failed to update endpoint (%s/%s) in hub cluster, %v", endpoints.Namespace, endpoints.Name, err)
		return
	}
	klog.V(5).Infof("update endpoints (%s/%s) in hub cluster", endpoints.Namespace, endpoints.Name)
}

func (c *PluginController) deleteEndpoints(endpoints *v1.Endpoints) {
	err := c.hubKubeClient.CoreV1().Endpoints(endpoints.Namespace).Delete(endpoints.Name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("failed to delete endpoint (%s/%s) in hub cluster, %v", endpoints.Namespace, endpoints.Name, err)
		return
	}
	klog.V(5).Infof("delete endpoints (%s/%s) in hub cluster", endpoints.Namespace, endpoints.Name)
}
