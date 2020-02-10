// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

type testContent struct {
	t                   *testing.T
	stopCh              chan struct{}
	configmapLister     []*corev1.ConfigMap
	objects             []runtime.Object
	controller          *Controller
	kubeSharedInformers kubeinformers.SharedInformerFactory
	aggregatorGetter    *InfoGetters
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
	c.aggregatorGetter = NewInfoGetters(kubeClient)
	c.controller = NewController(kubeClient, c.kubeSharedInformers, c.aggregatorGetter, c.stopCh)
}

func (c *testContent) runController() {
	c.newController()
	for _, d := range c.configmapLister {
		c.kubeSharedInformers.Core().V1().ConfigMaps().Informer().GetIndexer().Add(d)
	}
	c.kubeSharedInformers.Start(c.stopCh)
}

func (c *testContent) runWork(configMap *corev1.ConfigMap) {
	c.controller.enqueueAggregatorConfigMap(configMap)
	c.controller.processNextWorkItem()
	c.checkResult(configMap)
}

func (c *testContent) checkResult(configMap *corev1.ConfigMap) {
	name := configMap.Namespace + "/" + configMap.Name
	if subResource, ok := c.aggregatorGetter.nameToSubResource[name]; ok {
		if _, ok := c.aggregatorGetter.Get(subResource); ok {
			return
		}
	}
	c.t.Errorf("failed to get options")
}
func newConfigMap(namespace, name, subResource, secret string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"config": "mcm-aggregator",
			},
		},
		Data: map[string]string{
			"service":      "namespace/service",
			"port":         "8808",
			"path":         "/abc/def",
			"sub-resource": subResource,
			"use-id":       "true",
			"secret":       secret,
		},
		BinaryData: nil,
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
	configMap1 := newConfigMap("kube-system", "test2", "search", "kube-system/search-ca")
	c := newTestContent(t)
	c.configmapLister = append(c.configmapLister, configMap1)
	c.objects = append(c.objects, configMap1)
	c.objects = append(c.objects, secret1)

	c.runController()
	c.runWork(configMap1)

	configMap1 = newConfigMap("kube-system", "test2", "metrics", "kube-system/search-ca")
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
