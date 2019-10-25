// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	"testing"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestPluginType(t *testing.T) {
	plugin, _ := newKubeIngressPlugin()
	pluginType := plugin.GetType()
	if pluginType != "kube-ingress" {
		t.Fatalf("Expect to get kube-ingress, but get %s", pluginType)
	}
}

func TestDiscoveryRequired(t *testing.T) {
	plugin, _ := newKubeIngressPlugin()
	if !plugin.DiscoveryRequired() {
		t.Fatalf("Expect true, but false")
	}
}

func TestInvalidIngress(t *testing.T) {
	plugin, _ := newKubeIngressPlugin()

	nilEp := plugin.toEndpoints(nil)
	if nilEp != nil {
		t.Fatalf("Expect to get nil, but %v", nilEp)
	}

	withoutAnnotation := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unannotated",
			Namespace: "test",
		},
	}
	nilEp = plugin.toEndpoints(withoutAnnotation)
	if nilEp != nil {
		t.Fatalf("Expect to get nil, but %v", nilEp)
	}

	withoutRules := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "unannotated",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: "test",
				ServicePort: intstr.IntOrString{
					IntVal: 8000,
				},
			},
		},
	}
	nilEp = plugin.toEndpoints(withoutRules)
	if nilEp != nil {
		t.Fatalf("Expect to get nil, but %v", nilEp)
	}

	withoutStatus := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "unannotated",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "foo.bar.com",
			}},
		},
	}
	nilEp = plugin.toEndpoints(withoutStatus)
	if nilEp != nil {
		t.Fatalf("Expect to get nil, but %v", nilEp)
	}
}

func TestToEndpoitns(t *testing.T) {
	plugin, _ := newKubeIngressPlugin()

	withIP := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "withIP",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "foo.bar.com",
			}},
		},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					IP: "9.111.254.180",
				}},
			},
		},
	}
	ep := plugin.toEndpoints(withIP)
	if ep.Labels["mcm.ibm.com/service-type"] != "kube-ingress" {
		t.Fatalf("Expect to get kube-ingress, but %v", ep.Annotations["mcm.ibm.com/ingress-hosts"])
	}
	if ep.Annotations["mcm.ibm.com/ingress-hosts"] != "foo.bar.com" {
		t.Fatalf("Expect to get foo.bar.com, but %v", ep.Annotations["mcm.ibm.com/ingress-hosts"])
	}
	if ep.Subsets[0].Addresses[0].IP != "9.111.254.180" {
		t.Fatalf("Expect to get 9.111.254.180, but %v", ep.Subsets[0].Addresses[0].IP)
	}

	withHostname := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "ing",
			Namespace:   "test",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "foo.bar.com",
			}, {
				Host: "bar.foo.com",
			}},
		},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					Hostname: "example.us-central-1.aws.com",
				}},
			},
		},
	}
	ep = plugin.toEndpoints(withHostname)
	if ep.Annotations["mcm.ibm.com/ingress-hosts"] != "foo.bar.com,bar.foo.com" {
		t.Fatalf("Expect to get foo.bar.com, but %v", ep.Annotations["mcm.ibm.com/ingress-hosts"])
	}
	if ep.Annotations["mcm.ibm.com/load-balancer"] != "example.us-central-1.aws.com" {
		t.Fatalf("Expect to get example.us-central-1.aws.com, but %v", ep.Subsets[0].Addresses[0].Hostname)
	}
	if ep.Subsets[0].Addresses[0].IP != "255.255.255.255" {
		t.Fatalf("Expect to get 255.255.255.255, but %v", ep.Subsets[0].Addresses[0].IP)
	}
}

func TestSyncRegisteredEndpoints(t *testing.T) {
	plugin, store := newKubeIngressPlugin()

	unchanged := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "unchanged",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "foo.bar.com",
			}},
		},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					IP: "9.111.254.180",
				}},
			},
		},
	}
	store.Add(unchanged)

	changed := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "changed",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "foo.bar.com",
			}},
		},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					IP: "9.111.254.181",
				}},
			},
		},
	}

	store.Add(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unannotated",
			Namespace: "test",
		},
	})

	hubeps := []*v1.Endpoints{
		plugin.toEndpoints(unchanged),
		plugin.toEndpoints(changed),
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-ingress.test.unannotated"},
		},
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-ingress.test.deleted"},
		},
	}

	changed.Spec.Rules = append(changed.Spec.Rules, v1beta1.IngressRule{
		Host: "foo.bar.com",
	})
	store.Add(changed)

	store.Add(&v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "new",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "foo.bar.com",
			}},
		},
		Status: v1beta1.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					IP: "9.111.254.182",
				}},
			},
		},
	})

	toCreate, toDelete, toUpdate := plugin.SyncRegisteredEndpoints(hubeps)
	if len(toCreate) != 1 {
		t.Fatalf("Expect to get 1 svc to create, but get %d", len(toCreate))
	}
	if len(toDelete) != 2 {
		t.Fatalf("Expect to get 2 svc to delete, but get %d", len(toDelete))
	}
	if len(toUpdate) != 1 {
		t.Fatalf("Expect to get 1 svc to update, but get %d", len(toUpdate))
	}
}

func TestSyncDiscoveredResouces(t *testing.T) {
	plugin, _ := newKubeIngressPlugin()
	discovered := []*v1.Endpoints{
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster1.kube-ingress.test.ing1",
				Namespace: "cluster0ns",
				Labels:    map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{
					"mcm.ibm.com/service-discovery": "{}",
					"mcm.ibm.com/ingress-hosts":     "foo.bar.com",
				},
			},
			Subsets: []v1.EndpointSubset{v1.EndpointSubset{Addresses: []v1.EndpointAddress{v1.EndpointAddress{IP: "1.2.3.4"}}}},
		},
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster2.kube-ingress.test.ing2",
				Namespace: "cluster0ns",
				Labels:    map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{
					"mcm.ibm.com/service-discovery": "{\"dns-prefix\": \"http.ing\"}",
					"mcm.ibm.com/ingress-hosts":     "foo.bar.com,bar.foo.com",
					"mcm.ibm.com/load-balancer":     "xxxx.us-central-1.aws.com",
				},
			},
			Subsets: []v1.EndpointSubset{v1.EndpointSubset{Addresses: []v1.EndpointAddress{v1.EndpointAddress{
				IP: "255.255.255.255",
			}}}},
		},
		&v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "cluster3.kube-ingress.test.ing3",
				Namespace:   "cluster0ns",
				Labels:      map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{\"dns-prefix\": \"http.ing\"}"},
			},
			Subsets: []v1.EndpointSubset{v1.EndpointSubset{Addresses: []v1.EndpointAddress{v1.EndpointAddress{IP: "1.2.3.5"}}}},
		},
	}
	required, locations := plugin.SyncDiscoveredResouces(discovered)
	if !required {
		t.Fatalf("Expect true, but false")
	}
	if len(locations) != 3 {
		t.Fatalf("Expect 3, but %d", len(locations))
	}
}

func newKubeIngressPlugin() (*KubeIngressPlugin, cache.Store) {
	kubeClient := kubefake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute*10)
	store := informerFactory.Extensions().V1beta1().Ingresses().Informer().GetStore()
	return NewKubeIngressPlugin(informerFactory, options.GetSvcRegistryOptions()), store
}
