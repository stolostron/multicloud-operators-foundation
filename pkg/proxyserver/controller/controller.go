package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
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
	lister            v1.ConfigMapLister
	synced            cache.InformerSynced
	workqueue         workqueue.TypedRateLimitingInterface[string]
	stopCh            <-chan struct{}
}

func NewProxyServiceInfoController(
	client kubernetes.Interface,
	configMapLabels map[string]string,
	configMapInformer informercorev1.ConfigMapInformer,
	getter *getter.ProxyServiceInfoGetter,
	stopCh <-chan struct{}) *ProxyServiceInfoController {
	controller := &ProxyServiceInfoController{
		serviceInfoGetter: getter,
		client:            client,
		labelSelector:     &metav1.LabelSelector{MatchLabels: configMapLabels},
		lister:            configMapInformer.Lister(),
		synced:            configMapInformer.Informer().HasSynced,
		workqueue: workqueue.NewTypedRateLimitingQueueWithConfig(workqueue.DefaultTypedControllerRateLimiter[string](), workqueue.TypedRateLimitingQueueConfig[string]{
			Name: "proxyServiceInfoController",
		}),
		stopCh: stopCh,
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
	if utils.MatchLabelForLabelSelector(configmap.GetLabels(), c.labelSelector) {
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
	err := func(obj string) error {
		defer c.workqueue.Done(obj)
		if err := c.syncHandler(obj); err != nil {
			c.workqueue.AddRateLimited(obj)
			return fmt.Errorf("error syncing '%s': %s, requeuing", obj, err.Error())
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced proxy service info configmap '%s'", obj)
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

var optionsKey = []string{"service", "port", "path", "sub-resource", "secret", "caConfigMap"}

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

	caConfigMapNamespace, caConfigMapName, err := cache.SplitMetaNamespaceKey(cm.Data["caConfigMap"])

	if err != nil {
		return nil, fmt.Errorf("the caConfigMap format is wrong in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}
	if caConfigMapNamespace == "" {
		caConfigMapNamespace = serviceNamespace
	}
	caConfigmap, err := c.client.CoreV1().ConfigMaps(caConfigMapNamespace).Get(context.TODO(), caConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caConfigMap in configmap %s/%s, %v", cm.Namespace, cm.Name, err)
	}

	if _, ok := caConfigmap.Data["service-ca.crt"]; !ok {
		return nil, fmt.Errorf("failed to get service-ca.crt key in caConfigmap %s/%s", caConfigMapNamespace, caConfigMapName)

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
				CAData:   []byte(caConfigmap.Data["service-ca.crt"]),
			},
		},
	}, nil
}
