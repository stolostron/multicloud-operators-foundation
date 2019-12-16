// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"fmt"
	"strings"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type Controller struct {
	dynamicClient       kubernetes.Interface
	kubeSharedInformers informers.SharedInformerFactory
	aggregatorLister    v1.ConfigMapLister
	aggregatorSynced    cache.InformerSynced
	aggregatorGetter    *InfoGetter
	workqueue           workqueue.RateLimitingInterface
	stopCh              <-chan struct{}
}

// NewController returns a Controller
func NewController(
	dynamicClient kubernetes.Interface,
	informerFactory informers.SharedInformerFactory,
	aggregatorGetter *InfoGetter,
	stopCh <-chan struct{}) *Controller {
	configMapInformer := informerFactory.Core().V1().ConfigMaps()
	controller := &Controller{
		dynamicClient:       dynamicClient,
		kubeSharedInformers: informerFactory,
		aggregatorLister:    configMapInformer.Lister(),
		aggregatorSynced:    configMapInformer.Informer().HasSynced,
		aggregatorGetter:    aggregatorGetter,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "aggregatorController"),
		stopCh:              stopCh,
	}

	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueueAggregatorConfigMap(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueAggregatorConfigMap(newObj)
		},
		DeleteFunc: controller.deleteAggregator,
	})
	return controller
}

// Run is the main run loop of aggregator controller
func (c *Controller) Run() {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	klog.Info("Waiting for aggregator informer caches to sync")
	if !cache.WaitForCacheSync(c.stopCh, c.aggregatorSynced) {
		klog.Errorf("failed to wait for hcm caches to sync")
		return
	}

	go wait.Until(c.runWorker, time.Second, c.stopCh)
	<-c.stopCh
	klog.Info("Shutting aggregator controller")
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}
	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced aggregator configMap '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	aggregatorConfigmap, err := c.aggregatorLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// configmap is deleted, delete aggregator config
			c.aggregatorGetter.Delete(namespace + "/" + name)
			klog.V(5).Infof("delete aggregator %#v", namespace+"/"+name)
			return nil
		}

		return err
	}

	aggregatorOptions, err := getAggregatorOptions(aggregatorConfigmap)
	if err != nil {
		klog.Errorf("fail to get aggregator options %#v", err)
		return err
	}

	c.aggregatorGetter.AddAndUpdate(aggregatorOptions)
	klog.V(5).Infof("add aggregator %#v options %#v ", aggregatorConfigmap.Name, aggregatorConfigmap.Data)
	return nil
}

var aggregatorConfigMapLabels = map[string]string{
	"config": "mcm-aggregator",
}

func (c *Controller) enqueueAggregatorConfigMap(obj interface{}) {
	var key string
	var err error
	labelSelector := &metav1.LabelSelector{
		MatchLabels: aggregatorConfigMapLabels,
	}
	aggregatorConfigmap := obj.(*corev1.ConfigMap)
	if utils.MatchLabelForLabelSelector(aggregatorConfigmap.GetLabels(), labelSelector) {
		if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
			utilruntime.HandleError(err)
			return
		}
		c.workqueue.Add(key)
	}
}

func (c *Controller) deleteAggregator(obj interface{}) {
	var object metav1.Object
	var ok bool
	if _, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(5).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	c.enqueueAggregatorConfigMap(obj)
}

var aggregatorOptionsKey = []string{"service", "port", "path", "sub-resource", "use-id", "secret"}

func getAggregatorOptions(c *corev1.ConfigMap) (*Options, error) {
	for _, key := range aggregatorOptionsKey {
		if _, ok := c.Data[key]; !ok {
			return nil, fmt.Errorf("there is no %v key in configmap %v in namespace %v", key, c.GetName(), c.GetNamespace())
		}
	}

	aggratorOptions := &Options{
		name:        c.Namespace + "/" + c.Name,
		service:     c.Data["service"],
		port:        c.Data["port"],
		path:        strings.Trim(c.Data["path"], "/"),
		subResource: strings.Trim(c.Data["sub-resource"], "/"),
		secret:      c.Data["secret"],
	}
	if c.Data["use-id"] == "true" {
		aggratorOptions.useID = true
	}

	return aggratorOptions, nil
}
