// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	"fmt"
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPluginType(t *testing.T) {
	plugin, _ := newIstioPlugin()
	pluginType := plugin.GetType()
	if pluginType != "istio" {
		t.Fatalf("Expect to get istio, but get %s", pluginType)
	}
}

func TestSyncRegisteredEndpoints(t *testing.T) {
	plugin, store := newIstioPlugin()
	store.Add(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "test",
		},
	})
	store.Add(&v1.Service{
		Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Name: "http", Port: 8000}}},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "httpbin",
			Namespace:   "istio-test",
			Labels:      map[string]string{"app": "httpbin"},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	})

	toCreate1, toDelete1, toUpdate1 := plugin.SyncRegisteredEndpoints([]*v1.Endpoints{})
	if len(toCreate1) != 1 {
		t.Fatalf("Expect to 1, but %d", len(toCreate1))
	}
	if len(toDelete1) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toDelete1))
	}
	if len(toUpdate1) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toUpdate1))
	}

	store.Update(&v1.Service{
		Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Name: "http", Port: 8000}}},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "httpbin",
			Namespace:   "istio-test",
			Labels:      map[string]string{"app": "other-httpbin"},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	})
	toCreate2, toDelete2, toUpdate2 := plugin.SyncRegisteredEndpoints(toCreate1)
	if len(toCreate2) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toCreate2))
	}
	if len(toDelete2) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toDelete2))
	}
	if len(toUpdate2) != 1 {
		t.Fatalf("Expect to 1, but %d", len(toUpdate2))
	}

	store.Update(&v1.Service{
		Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Name: "http", Port: 8000}}},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "istio-test",
			Labels:    map[string]string{"app": "other-httpbin"},
		},
	})
	toCreate3, toDelete3, toUpdate3 := plugin.SyncRegisteredEndpoints(toCreate1)
	if len(toCreate3) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toCreate3))
	}
	if len(toDelete3) != 1 {
		t.Fatalf("Expect to 1, but %d", len(toDelete3))
	}
	if len(toUpdate3) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toUpdate3))
	}

	store.Delete(v1.Service{
		Spec: v1.ServiceSpec{Ports: []v1.ServicePort{{Name: "http", Port: 8000}}},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin",
			Namespace: "istio-test",
			Labels:    map[string]string{"app": "other-httpbin"},
		},
	})
	toCreate4, toDelete4, toUpdate4 := plugin.SyncRegisteredEndpoints(toCreate1)
	if len(toCreate4) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toCreate4))
	}
	if len(toDelete4) != 1 {
		t.Fatalf("Expect to 1, but %d", len(toDelete4))
	}
	if len(toUpdate4) != 0 {
		t.Fatalf("Expect to 0, but %d", len(toUpdate4))
	}
}

func TestDiscoveryRequired(t *testing.T) {
	plugin, _ := newIstioPlugin()
	if !plugin.DiscoveryRequired() {
		t.Fatalf("Expect to true, but false")
	}
}

func TestSyncDiscoveredResouces(t *testing.T) {
	plugin, _ := newIstioPlugin()

	discoveredeps := []*v1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster1.istio.test.http",
				Namespace: "cluster1",
				Labels:    map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{
					"mcm.ibm.com/service-discovery":   "{}",
					"mcm.ibm.com/istio-service-ports": "8000",
				},
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{{IP: "1.2.3.4"}},
					Ports:     []v1.EndpointPort{{Port: 32178}},
				},
			},
		},
	}

	plugin.SyncDiscoveredResouces(discoveredeps)

	list1, _ := plugin.serviceEntryClient.Namespace("default").List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", autoCreationLabel),
	})
	if len(list1.Items) != 1 {
		t.Fatalf("Expect to 1, but %d", len(list1.Items))
	}

	discoveredeps = append(discoveredeps, &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster2.istio.test.http",
			Namespace: "cluster2",
			Labels:    map[string]string{"mcm.ibm.com/auto-discovery": "true"},
			Annotations: map[string]string{
				"mcm.ibm.com/service-discovery":   "{}",
				"mcm.ibm.com/istio-service-ports": "8000",
			},
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{{IP: "1.2.3.5"}},
				Ports:     []v1.EndpointPort{{Port: 32178}},
			},
		},
	}, &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster1.istio.other.http",
			Namespace: "cluster1",
			Labels:    map[string]string{"mcm.ibm.com/auto-discovery": "true"},
			Annotations: map[string]string{
				"mcm.ibm.com/service-discovery":   "{}",
				"mcm.ibm.com/istio-service-ports": "8000",
			},
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{{IP: "1.2.3.4"}},
				Ports:     []v1.EndpointPort{{Port: 32178}},
			},
		},
	})

	plugin.SyncDiscoveredResouces(discoveredeps)

	list2, _ := plugin.serviceEntryClient.Namespace("default").List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", autoCreationLabel),
	})
	if len(list2.Items) != 2 {
		t.Fatalf("Expect to 2, but %d", len(list2.Items))
	}

	plugin.SyncDiscoveredResouces(discoveredeps[1:])

	list3, _ := plugin.serviceEntryClient.Namespace("default").List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", autoCreationLabel),
	})
	if len(list3.Items) != 2 {
		t.Fatalf("Expect to 2, but %d", len(list2.Items))
	}

	plugin.SyncDiscoveredResouces(discoveredeps[2:])

	list4, _ := plugin.serviceEntryClient.Namespace("default").List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", autoCreationLabel),
	})
	if len(list4.Items) != 1 {
		t.Fatalf("Expect to 1, but %d", len(list2.Items))
	}
}

func newServiceEntry(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "networking.istio.io/v1alpha3",
			"kind":       "ServiceEntry",
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

func newIstioPlugin() (*IstioPlugin, cache.Store) {
	kubeClient := kubefake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute*10)
	serviceStore := informerFactory.Core().V1().Services().Informer().GetStore()

	options := options.GetSvcRegistryOptions()
	options.ClusterProxyIP = "1.2.3.4"

	//prepare an istio namespace
	kubeClient.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "istio-test",
			Labels: map[string]string{"istio-injection": "enabled"},
		},
	})

	//prepare istio ingressgateway
	serviceStore.Add(&v1.Service{
		Spec:   v1.ServiceSpec{Ports: []v1.ServicePort{{Name: "tls", Port: 15443, NodePort: 32178}}},
		Status: v1.ServiceStatus{LoadBalancer: v1.LoadBalancerStatus{}},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-ingressgateway",
			Namespace: "istio-system",
		},
	})

	dynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme(), newServiceEntry(options.IstioPluginOptions.ServiceEntryRegistryNamespace, "servieentry"))
	return NewIstioPlugin(kubeClient, dynamicClient, informerFactory, options), serviceStore
}
