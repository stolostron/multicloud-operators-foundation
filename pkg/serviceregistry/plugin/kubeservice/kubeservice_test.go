// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	"testing"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestNoClusterProxyIP(t *testing.T) {
	memberKubeClient := kubefake.NewSimpleClientset()
	memberInformerFactory := informers.NewSharedInformerFactory(memberKubeClient, time.Minute*10)
	options := options.GetSvcRegistryOptions()
	plugin := NewKubeServicePlugin(memberKubeClient, memberInformerFactory, options)
	nodePortSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
			Ports: []v1.ServicePort{{
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
				Port:       80,
				NodePort:   30080,
				Protocol:   "TCP",
				Name:       "http",
			},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc",
			Namespace:   "test",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	ep := plugin.toEndpoints(nodePortSvc)
	if ep != nil {
		t.Fatalf("Expect to get nil, but %v", ep)
	}
}

func TestInvalidService(t *testing.T) {
	plugin, _ := newKubeServicePlugin()

	nilEp := plugin.toEndpoints(nil)
	if nilEp != nil {
		t.Fatalf("Expect to get nil, but %v", nilEp)
	}

	unannotatedSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "test",
		},
	}
	unannotatedEp := plugin.toEndpoints(unannotatedSvc)
	if unannotatedEp != nil {
		t.Fatalf("Expect to get nil, but %v", unannotatedEp)
	}

	clusterIPSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "ClusterIP",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	wrongTypeEp := plugin.toEndpoints(clusterIPSvc)
	if wrongTypeEp != nil {
		t.Fatalf("Expect to get nil, but %v", wrongTypeEp)
	}
}

func TestToEndpointWithNodePortSvc(t *testing.T) {
	nodePortSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
			Ports: []v1.ServicePort{{
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
				Port:       80,
				NodePort:   30080,
				Protocol:   "TCP",
				Name:       "http",
			}, {
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8443},
				Port:       443,
				NodePort:   30443,
				Protocol:   "TCP",
				Name:       "https",
			},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc",
			Namespace:   "test",
			Labels:      map[string]string{"app": "test"},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	plugin, _ := newKubeServicePlugin()
	ep := plugin.toEndpoints(nodePortSvc)
	if ep.Name != "kube-service.test.svc" {
		t.Fatalf("Expect to get name with kube-service.test.svc, but get %s", ep.Name)
	}
	if ep.Namespace != "default" {
		t.Fatalf("Expect to get namespace with default, but get %s", ep.Namespace)
	}
	if ep.Subsets[0].Addresses[0].IP != "1.2.3.4" {
		t.Fatalf("Expect to get ip with 1.2.3.4, but get %s", ep.Subsets[0].Addresses[0].IP)
	}
	if len(ep.Subsets[0].Ports) != 2 {
		t.Fatalf("Expect 2 ports, but get %d", len(ep.Subsets[0].Ports))
	}
}

func TestUnreadyLoadBalancer(t *testing.T) {
	unreadyLoadBalancerSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "LoadBalancer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	plugin, _ := newKubeServicePlugin()
	ep := plugin.toEndpoints(unreadyLoadBalancerSvc)

	if ep.Subsets[0].Addresses[0].IP != "1.2.3.4" {
		t.Fatalf("Expect to get 1.2.3.4, but failed")
	}
}

func TestLoadBalancerSvc(t *testing.T) {
	plugin, _ := newKubeServicePlugin()

	gkeLoadBalancerSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "LoadBalancer",
			Ports: []v1.ServicePort{{
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
				Port:       80,
				NodePort:   32676,
				Protocol:   "TCP",
			}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "gke-svc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					IP: "203.0.113.100",
				}},
			},
		},
	}
	ep := plugin.toEndpoints(gkeLoadBalancerSvc)
	if ep.Subsets[0].Addresses[0].IP != "203.0.113.100" {
		t.Fatalf("Expect to get ip with 203.0.113.100, but get %s", ep.Subsets[0].Addresses[0].IP)
	}

	awsLoadBalancerSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "LoadBalancer",
			Ports: []v1.ServicePort{{
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: 8080},
				Port:       80,
				NodePort:   32503,
				Protocol:   "TCP",
				Name:       "http",
			}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aws-svc",
			Namespace: "test",
			Annotations: map[string]string{
				"mcm.ibm.com/service-discovery":                     "{}",
				"service.beta.kubernetes.io/aws-load-balancer-type": "nlb"},
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					Hostname: "xxxx.us-central-1.elb.amazonaws.com",
				}},
			},
		},
	}
	ep = plugin.toEndpoints(awsLoadBalancerSvc)
	if ep.Subsets[0].Addresses[0].IP != "255.255.255.255" {
		t.Fatalf("Expect to get ip with 255.255.255.255, but get %s", ep.Subsets[0].Addresses[0].IP)
	}
	if ep.Annotations["mcm.ibm.com/load-balancer"] != "xxxx.us-central-1.elb.amazonaws.com" {
		t.Fatalf("Expect to get hostname xxxx.us-central-1.elb.amazonaws.com, but get %s", ep.Subsets[0].Addresses[0].Hostname)
	}
}

func TestPluginType(t *testing.T) {
	plugin, _ := newKubeServicePlugin()
	pluginType := plugin.GetType()
	if pluginType != "kube-service" {
		t.Fatalf("Expect to get kube-service, but get %s", pluginType)
	}
}

func TestToDeletionEndpoints(t *testing.T) {
	deletionSvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	plugin, _ := newKubeServicePlugin()
	ep := plugin.toDeletionEndpoints(deletionSvc)
	if ep.Name != "kube-service.test.svc" {
		t.Fatalf("Expect to get name with kube-service.test.svc, but get %s", ep.Name)
	}
	if ep.Namespace != "default" {
		t.Fatalf("Expect to get namespace with default, but get %s", ep.Namespace)
	}
}

func TestSyncRegisteredEndpoints(t *testing.T) {
	plugin, svcStore := newKubeServicePlugin()
	changedsvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
			Ports: []v1.ServicePort{{
				Port:     80,
				NodePort: 32503,
				Protocol: "TCP",
				Name:     "http",
			}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "changedsvc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	unchangedsvc := &v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
			Ports: []v1.ServicePort{{
				Port:     80,
				NodePort: 32503,
				Protocol: "TCP",
				Name:     "http",
			}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "unchangedsvc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	}
	hubeps := []*v1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-service.test.deletionsvc"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "kube-service.test.unannotatedsvc"},
		},
		plugin.toEndpoints(changedsvc),
		plugin.toEndpoints(unchangedsvc),
	}
	svcStore.Add(&v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unannotatedsvc",
			Namespace: "test",
		},
	})
	changedsvc.Labels = map[string]string{"test": "test"}
	svcStore.Add(changedsvc)
	svcStore.Add(&v1.Service{
		Spec: v1.ServiceSpec{
			Type: "NodePort",
			Ports: []v1.ServicePort{{
				Port:     80,
				NodePort: 32503,
				Protocol: "TCP",
				Name:     "http",
			}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "newsvc",
			Namespace:   "test",
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
	})
	svcStore.Add(unchangedsvc)
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

func TestDiscoveryRequired(t *testing.T) {
	plugin, _ := newKubeServicePlugin()
	if !plugin.DiscoveryRequired() {
		t.Fatalf("Expect true, but false")
	}
}

func TestSyncDiscoveredResouces(t *testing.T) {
	plugin, svcStore := newKubeServicePlugin()

	discoveredEndpoints := []*v1.Endpoints{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "cluster1.kube-service.test.svc",
				Namespace:   "cluster1ns",
				Labels:      map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
			},
			Subsets: []v1.EndpointSubset{{Addresses: []v1.EndpointAddress{{IP: "1.2.3.4"}}}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster2.kube-service.test.svc2",
				Namespace: "cluster2ns",
				Labels:    map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{
					"mcm.ibm.com/service-discovery": "{}",
					"mcm.ibm.com/load-balancer":     "xxxx.us-central-1.aws.com",
				},
			},
			Subsets: []v1.EndpointSubset{{Addresses: []v1.EndpointAddress{{
				IP: "255.255.255.255",
			}}}},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "cluster3.kube-service.test.svc3",
				Namespace:   "cluster3ns",
				Labels:      map[string]string{"mcm.ibm.com/auto-discovery": "true"},
				Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{\"dns-prefix\": \"http.svc\"}"},
			},
			Subsets: []v1.EndpointSubset{{Addresses: []v1.EndpointAddress{{IP: "1.2.3.5"}}}},
		},
	}
	required, locations := plugin.SyncDiscoveredResouces(discoveredEndpoints)
	if !required {
		t.Fatalf("Expect true, but false")
	}
	if len(locations) != 3 {
		t.Fatalf("Expect 3, but %d", len(locations))
	}

	// has local service
	discoveredEndpoints = append(discoveredEndpoints, &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "cluster4.kube-service.test.svc4",
			Namespace:   "cluster4ns",
			Labels:      map[string]string{"mcm.ibm.com/auto-discovery": "true"},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{\"dns-prefix\": \"http.svc\"}"},
		},
		Subsets: []v1.EndpointSubset{{Addresses: []v1.EndpointAddress{{IP: "1.2.3.5"}}}},
	})
	svcStore.Add(&v1.Service{
		Spec: v1.ServiceSpec{ClusterIP: "10.0.0.1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc4",
			Namespace: "test",
		},
	})
	required, locations = plugin.SyncDiscoveredResouces(discoveredEndpoints)
	if !required {
		t.Fatalf("Expect true, but false")
	}
	if len(locations) != 5 {
		t.Fatalf("Expect 5, but %d", len(locations))
	}
}

func newKubeServicePlugin() (*KubeServicePlugin, cache.Store) {
	memberKubeClient := kubefake.NewSimpleClientset()
	memberInformerFactory := informers.NewSharedInformerFactory(memberKubeClient, time.Minute*10)
	memberServiceStore := memberInformerFactory.Core().V1().Services().Informer().GetStore()
	options := options.GetSvcRegistryOptions()
	options.ClusterProxyIP = "1.2.3.4"
	return NewKubeServicePlugin(memberKubeClient, memberInformerFactory, options), memberServiceStore
}
