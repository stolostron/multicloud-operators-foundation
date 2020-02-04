// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package gc

import (
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	clusterinformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	clusterlisters "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

type GarbageCollectorController struct {
	// hcmclientset is a clientset for our own API group
	dynamicClient          dynamic.Interface
	clusterLister          clusterlisters.ClusterLister
	sharedInformers        informers.SharedInformerFactory
	storeMap               map[schema.GroupVersionResource]cache.Store
	garbageCollectorPeriod time.Duration
	stopCh                 <-chan struct{}
	clusterSynced          cache.InformerSynced
}

type resourceAttr struct {
	name      string
	namespace string
	resource  string
}

// syncResources is the resource list to for gc to keeps on syncing
var syncResources = []schema.GroupVersionResource{
	{Group: "mcm.ibm.com", Version: "v1alpha1", Resource: "works"},
}

// watchedResources is the resource list to keep on monitoring
var watchedResources = map[string]schema.GroupVersionResource{
	"works":         {Group: "mcm.ibm.com", Version: "v1alpha1", Resource: "works"},
	"resourceviews": {Group: "mcm.ibm.com", Version: "v1alpha1", Resource: "resourceviews"},
	"worksets":      {Group: "mcm.ibm.com", Version: "v1alpha1", Resource: "worksets"},
}

// expiredResources is the resource list that needs to be expired
var expiredResources = []schema.GroupVersionResource{
	{Group: "mcm.ibm.com", Version: "v1alpha1", Resource: "resourceviews"},
	{Group: "mcm.ibm.com", Version: "v1alpha1", Resource: "worksets"},
}

// NewGarbageCollectorController returns a GarbageCollectorController
func NewGarbageCollectorController(
	dynamicClient dynamic.Interface,
	clusterinformerFactory clusterinformers.SharedInformerFactory,
	informerFactory informers.SharedInformerFactory,
	garbageCollectorPeriod time.Duration,
	stopCh <-chan struct{},
) *GarbageCollectorController {
	storeMap := map[schema.GroupVersionResource]cache.Store{}
	clusterInformer := clusterinformerFactory.Clusterregistry().V1alpha1().Clusters()
	controller := &GarbageCollectorController{
		dynamicClient:          dynamicClient,
		clusterLister:          clusterInformer.Lister(),
		sharedInformers:        informerFactory,
		clusterSynced:          clusterInformer.Informer().HasSynced,
		garbageCollectorPeriod: garbageCollectorPeriod,
		stopCh:                 stopCh,
		storeMap:               storeMap,
	}

	return controller
}

func (g *GarbageCollectorController) controllerFor(resource schema.GroupVersionResource) (cache.Controller, cache.Store, error) {
	shared, err := g.sharedInformers.ForResource(resource)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to use a shared informer for resource %q: %v", resource.String(), err)
	}
	klog.V(4).Infof("using a shared informer for resource %q", resource.String())
	return shared.Informer().GetController(), shared.Informer().GetStore(), nil
}

// Run is the main run loop of garbage collector
func (g *GarbageCollectorController) Run() {
	defer utilruntime.HandleCrash()
	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(g.stopCh, g.clusterSynced); !ok {
		klog.Errorf("failed to wait for hcm caches to sync")
		return
	}

	for _, gvr := range watchedResources {
		controller, store, err := g.controllerFor(gvr)
		if err != nil {
			klog.Errorf("Failed to monitor: %v", err)
			continue
		}

		g.storeMap[gvr] = store
		go controller.Run(g.stopCh)
	}

	go wait.Until(g.cleanExpiredObject, g.garbageCollectorPeriod, g.stopCh)
	go wait.Until(g.syncResource, 5*time.Second, g.stopCh)

	<-g.stopCh
	klog.Info("Shutting gc controller")
}

func (g *GarbageCollectorController) cleanExpiredObject() {
	for _, resource := range expiredResources {
		store, ok := g.storeMap[resource]
		if !ok {
			continue
		}

		objList := store.List()
		for _, objInterface := range objList {
			obj := objInterface.(runtime.Object)
			err := g.deleteExpiredOneResource(resource, obj)
			if err != nil {
				klog.Errorf("Failed to delete expired resource %v: %v", resource, err)
			}
		}
	}
}

func (g *GarbageCollectorController) deleteExpiredOneResource(gvr schema.GroupVersionResource, obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	if accessor.GetCreationTimestamp().Add(g.garbageCollectorPeriod).After(time.Now()) {
		return nil
	}

	// ugly code to handle resourceview case...
	if gvr.Resource == "resourceviews" {
		view, ok := obj.(*v1alpha1.ResourceView)
		if !ok {
			return fmt.Errorf("expected to get resourceview, but failed to get it")
		}

		if view.Spec.Mode != "" {
			return nil
		}
	}

	return g.handleDelete(gvr, accessor.GetNamespace(), accessor.GetName())
}

func (g *GarbageCollectorController) syncResource() {
	for _, re := range syncResources {
		store, ok := g.storeMap[re]
		if !ok {
			continue
		}
		objList := store.List()
		for _, obj := range objList {
			err := g.syncOneResource(re, obj.(runtime.Object))
			if err != nil {
				klog.Errorf("Failed to sync resources: %v", err)
				continue
			}
		}
	}
}

func (g *GarbageCollectorController) syncOneResource(gvr schema.GroupVersionResource, obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	annotations := accessor.GetAnnotations()
	//get owner of the resources
	if _, ok := annotations[mcm.OwnersLabel]; !ok {
		return nil
	}

	resourceOwnerArray, err := g.extractOwnersInLabel(annotations[mcm.OwnersLabel])
	if err != nil {
		return fmt.Errorf("failed to get owners: %v", err)
	}

	shouldDelete := false
	for _, resourceOwner := range resourceOwnerArray {
		shouldDelete = g.shouldDeleteResource(resourceOwner)
		if shouldDelete {
			break
		}
	}

	if shouldDelete {
		namespace, name, err := g.getObjectNamespaceName(obj)
		if err != nil {
			return fmt.Errorf("failed to transfer resources: %v", err)
		}

		err = g.handleDelete(gvr, namespace, name)
		if err != nil {
			return fmt.Errorf("failed to delete resources: %v", err)
		}
	}

	return nil
}

func (g *GarbageCollectorController) shouldDeleteResource(resourceOwner resourceAttr) bool {
	//use cluster list to get cluster
	if resourceOwner.resource == "clusters" {
		_, err := g.clusterLister.Clusters(resourceOwner.namespace).Get(resourceOwner.name)
		if err != nil {
			if errors.IsNotFound(err) {
				return true
			}
			utilruntime.HandleError(fmt.Errorf("can not get cluster: %v, error: %s", resourceOwner, err))
		}
		return false
	}
	//use restclient to get mcm resource
	gvr, ok := watchedResources[resourceOwner.resource]
	if !ok {
		return false
	}

	store, ok := g.storeMap[gvr]
	if !ok {
		return false
	}
	_, exist, err := store.GetByKey(fmt.Sprintf("%s/%s", resourceOwner.namespace, resourceOwner.name))
	if !exist {
		return true
	}

	if err != nil {
		klog.Errorf("Failed to get object: %v", err)
		return true
	}

	return false
}

func (g *GarbageCollectorController) extractOwnersInLabel(owners string) ([]resourceAttr, error) {
	resourceArr := []resourceAttr{}
	if len(owners) == 0 {
		return resourceArr, nil
	}
	ownerArray := strings.Split(owners, ",")
	for _, owner := range ownerArray {
		res := strings.Split(owner, ".")
		if len(res) < 2 {
			return resourceArr, fmt.Errorf("can not split owner : %s", owner)
		}
		tempResource := resourceAttr{
			resource:  res[0],
			namespace: res[1],
			name:      res[2],
		}
		resourceArr = append(resourceArr, tempResource)
	}
	return resourceArr, nil
}

//transfer the obj type to resourceAttr struct
func (g *GarbageCollectorController) getObjectNamespaceName(obj interface{}) (string, string, error) {
	remeta, err := meta.Accessor(obj)
	if err != nil {
		return "", "", fmt.Errorf("object has no meta: %v", err)
	}

	return remeta.GetNamespace(), remeta.GetName(), nil
}

func (g *GarbageCollectorController) handleDelete(gvr schema.GroupVersionResource, namespace, name string) error {
	err := g.dynamicClient.Resource(gvr).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
