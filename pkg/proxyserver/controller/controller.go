package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type ProxyServiceInfoController struct {
	serviceInfoGetter *getter.ProxyServiceInfoGetter
	client            kubernetes.Interface
	labelSelector     *metav1.LabelSelector
	informerFactory   informers.SharedInformerFactory
	lister            v1.ConfigMapLister
	synced            cache.InformerSynced
	workqueue         workqueue.RateLimitingInterface
	stopCh            <-chan struct{}
}

func NewProxyServiceInfoController(
	client kubernetes.Interface,
	configMapLabels map[string]string,
	informerFactory informers.SharedInformerFactory,
	getter *getter.ProxyServiceInfoGetter,
	stopCh <-chan struct{}) *ProxyServiceInfoController {
	configMapInformer := informerFactory.Core().V1().ConfigMaps()
	controller := &ProxyServiceInfoController{
		serviceInfoGetter: getter,
		client:            client,
		labelSelector:     &metav1.LabelSelector{MatchLabels: configMapLabels},
		informerFactory:   informerFactory,
		lister:            configMapInformer.Lister(),
		synced:            configMapInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "proxyServiceInfoController"),
		stopCh:            stopCh,
	}

	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueue(newObj)
		},
		DeleteFunc: controller.deleteObj,
	})

	return controller
}

func (c *ProxyServiceInfoController) enqueue(obj interface{}) {
	var key string
	var err error

	configmap := obj.(*corev1.ConfigMap)
	if matchLabelForLabelSelector(configmap.GetLabels(), c.labelSelector) {
		if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
			utilruntime.HandleError(err)
			return
		}
		c.workqueue.Add(key)
	}
}

func (c *ProxyServiceInfoController) deleteObj(obj interface{}) {
	var ok bool
	if _, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		_, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
	}
	c.enqueue(obj)
}

func (c *ProxyServiceInfoController) Run() {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Waiting for proxy service info configmap informer caches to sync")
	if !cache.WaitForCacheSync(c.stopCh, c.synced) {
		klog.Errorf("failed to wait for proxy service info configmap informer caches to sync")
		return
	}

	go wait.Until(c.runWorker, time.Second, c.stopCh)
	<-c.stopCh
	klog.Info("Shutting proxy service info controller")
}

func (c *ProxyServiceInfoController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ProxyServiceInfoController) processNextWorkItem() bool {
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
		klog.Infof("Successfully synced proxy service info configmap '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *ProxyServiceInfoController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: '%s'", key))
		return nil
	}

	configMap, err := c.lister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// delete proxy service info when configmap is deleted.
			c.serviceInfoGetter.Delete(namespace + "/" + name)
			return nil
		}
		return err
	}

	serviceInfo, err := c.generateServiceInfo(configMap)
	if err != nil {
		return err
	}

	c.serviceInfoGetter.Add(serviceInfo)
	return nil
}

var optionsKey = []string{"service", "port", "path", "sub-resource", "secret"}

func (c *ProxyServiceInfoController) generateServiceInfo(cm *corev1.ConfigMap) (*getter.ProxyServiceInfo, error) {
	for _, key := range optionsKey {
		if _, ok := cm.Data[key]; !ok {
			return nil, fmt.Errorf("the '%s' key is required in configmap %s/%s", key, cm.Namespace, cm.Name)
		}
	}

	useID := false
	if v, ok := cm.Data["use-id"]; ok && v == "true" {
		useID = true
	}

	serviceNamespace, serviceName, err := cache.SplitMetaNamespaceKey(cm.Data["service"])
	if err != nil {
		return nil, fmt.Errorf("the service format is wrong in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	if serviceNamespace == "" || serviceName == "" {
		return nil, fmt.Errorf("the service format is wrong in configmap %s/%s %s/%s", cm.Namespace, cm.Name, serviceNamespace, serviceName)
	}

	secretNamespace, secretName, err := cache.SplitMetaNamespaceKey(cm.Data["secret"])
	if err != nil {
		return nil, fmt.Errorf("the secret format is wrong in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	if secretNamespace == "" {
		secretNamespace = serviceNamespace
	}

	secret, err := c.client.CoreV1().Secrets(secretNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	return &getter.ProxyServiceInfo{
		Name:             cm.Namespace + "/" + cm.Name,
		SubResource:      strings.Trim(cm.Data["sub-resource"], "/"),
		ServiceName:      serviceName,
		ServiceNamespace: serviceNamespace,
		ServicePort:      cm.Data["port"],
		RootPath:         strings.Trim(cm.Data["path"], "/"),
		UseID:            useID,
		RestConfig: &rest.Config{
			TLSClientConfig: rest.TLSClientConfig{
				CertData: secret.Data["tls.crt"],
				KeyData:  secret.Data["tls.key"],
				CAData:   secret.Data["ca.crt"],
			},
		},
	}, nil
}

// matchLabelForLabelSelector match labels for labelselector, if labelSelecor is nil, select everything
func matchLabelForLabelSelector(targetLabels map[string]string, labelSelector *metav1.LabelSelector) bool {
	var err error
	var selector = labels.Everything()

	if labelSelector != nil {
		selector, err = metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return false
		}
	}

	if selector.Matches(labels.Set(targetLabels)) {
		return true
	}
	return false
}
