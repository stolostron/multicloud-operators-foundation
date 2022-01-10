package controller

import (
	"testing"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func newConfigMap(namespace, name, subResource, secret string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"config": "acm-proxyserver",
			},
		},
		Data: map[string]string{
			"service":      "default/service",
			"port":         "8808",
			"path":         "/api/",
			"sub-resource": subResource,
			"use-id":       "true",
			"secret":       secret,
		},
	}
}

func newSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ca.crt":  []byte("abc"),
			"tls.crt": []byte("abc"),
			"tls.key": []byte("abc"),
		},
		StringData: nil,
		Type:       "",
	}
}

type testContent struct {
	t                   *testing.T
	stopCh              chan struct{}
	configmapLister     []*corev1.ConfigMap
	objects             []runtime.Object
	controller          *ProxyServiceInfoController
	kubeSharedInformers kubeinformers.SharedInformerFactory
	serviceInfoGetter   *getter.ProxyServiceInfoGetter
	configMapLabels     map[string]string
}

func newTestContent(t *testing.T) *testContent {
	return &testContent{
		t:      t,
		stopCh: make(chan struct{}),
	}
}

func (c *testContent) newController() {
	kubeClient := k8sfake.NewSimpleClientset(c.objects...)
	c.kubeSharedInformers = kubeinformers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
	c.serviceInfoGetter = getter.NewProxyServiceInfoGetter()
	c.configMapLabels = map[string]string{"config": "acm-proxyserver"}
	c.controller = NewProxyServiceInfoController(kubeClient, c.configMapLabels,
		c.kubeSharedInformers, c.serviceInfoGetter, c.stopCh)
}

func (c *testContent) runController() {
	c.newController()
	for _, d := range c.configmapLister {
		c.kubeSharedInformers.Core().V1().ConfigMaps().Informer().GetIndexer().Add(d)
	}
	c.kubeSharedInformers.Start(c.stopCh)
}

func (c *testContent) runWork(configMap *corev1.ConfigMap) {
	c.controller.enqueue(configMap)
	c.controller.processNextWorkItem()
	c.checkResult(configMap)
}

func (c *testContent) checkResult(configMap *corev1.ConfigMap) {
	subResource := configMap.Data["sub-resource"]
	if info := c.serviceInfoGetter.Get(subResource); info != nil {
		if info.Name == configMap.Namespace+"/"+configMap.Name {
			return
		}
	}
	c.t.Errorf("failed to get options")
}

func TestAddOptions(t *testing.T) {
	secret1 := newSecret("default", "finding-ca")
	configMap1 := newConfigMap("default", "test1", "finding", "default/finding-ca")
	c := newTestContent(t)
	c.configmapLister = append(c.configmapLister, configMap1)
	c.objects = append(c.objects, configMap1)
	c.objects = append(c.objects, secret1)

	c.runController()
	c.runWork(configMap1)

	close(c.stopCh)
}

func TestUpdateOptions(t *testing.T) {
	secret1 := newSecret("kube-system", "search-ca")
	secret2 := newSecret("default", "search-ca")
	configMap1 := newConfigMap("kube-system", "test2", "search", "kube-system/search-ca")
	c := newTestContent(t)
	c.configmapLister = append(c.configmapLister, configMap1)
	c.objects = append(c.objects, configMap1)
	c.objects = append(c.objects, secret1, secret2)

	c.runController()
	c.runWork(configMap1)

	configMap1 = newConfigMap("kube-system", "test2", "search", "default/search-ca")
	c.runWork(configMap1)

	close(c.stopCh)
}

func TestDeleteOptions(t *testing.T) {
	secret1 := newSecret("default", "finding-ca")
	configMap1 := newConfigMap("default", "test1", "finding", "default/finding-ca")
	c := newTestContent(t)
	c.configmapLister = append(c.configmapLister, configMap1)
	c.objects = append(c.objects, configMap1)
	c.objects = append(c.objects, secret1)

	c.runController()
	c.runWork(configMap1)

	c.configmapLister = []*corev1.ConfigMap{}
	c.runWork(configMap1)

	close(c.stopCh)
}
