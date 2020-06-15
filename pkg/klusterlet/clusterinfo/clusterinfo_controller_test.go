package controllers

import (
	stdlog "log"
	"os"
	"testing"

	tlog "github.com/go-logr/logr/testing"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/scheme"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	routev1Fake "github.com/openshift/client-go/route/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"

	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	kubeNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
			Labels: map[string]string{
				"kubernetes.io/arch":             "amd64",
				"kubernetes.io/os":               "linux",
				"node-role.kubernetes.io/worker": "",
				"node.openshift.io/os_id":        "rhcos",
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.Quantity{},
				corev1.ResourceMemory: resource.Quantity{},
			},
			Conditions: []corev1.NodeCondition{
				{
					Type: corev1.NodeReady,
				},
			},
		},
	}

	ocpConsole = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "console-config",
			Namespace: "openshift-console",
		},
		Data: map[string]string{
			"console-config.yaml": "apiVersion: console.openshift.io/v1\nauth:\n" +
				"clientID: console\n  clientSecretFile: /var/oauth-config/clientSecret\n" +
				"oauthEndpointCAFile: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt\n" +
				"clusterInfo:\n  consoleBaseAddress: https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com\n" +
				"masterPublicURL: https://api.daliu-clu428.dev04.red-chesterfield.com:6443\ncustomization:\n" +
				"branding: ocp\n  documentationBaseURL: https://docs.openshift.com/container-platform/4.3/\n" +
				"kind: ConsoleConfig\nproviders: {}\nservingInfo:\n  bindAddress: https://[::]:8443\n" +
				"certFile: /var/serving-cert/tls.crt\n  keyFile: /var/serving-cert/tls.key\n",
		},
	}
	kubeEndpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "127.0.0.1",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: 443,
					},
				},
			},
		},
	}

	kubeMonitoringSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("aaa"),
			"tls.key": []byte("aaa"),
		},
	}
	agentIngress = &extensionv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extension/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "acm-ingress-testcluster-agent",
			Namespace:         "kube-system",
			CreationTimestamp: metav1.Now(),
		},
		Spec: extensionv1beta1.IngressSpec{
			Rules: []extensionv1beta1.IngressRule{
				{
					Host: "test.com",
				},
			},
		},
		Status: extensionv1beta1.IngressStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "127.0.0.1",
					},
				},
			},
		},
	}

	agentService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "10.0.0.1",
					},
				},
			},
		},
	}
)
var clusterInfoNamespace = "cn1"
var clusterInfoName = "c1"
var clusterc1Request = reconcile.Request{
	NamespacedName: types.NamespacedName{
		Name: clusterInfoName, Namespace: clusterInfoNamespace}}

var cfg *rest.Config

func TestMain(m *testing.M) {
	t := &envtest.Environment{}
	var err error
	if cfg, err = t.Start(); err != nil {
		stdlog.Fatal(err)
	}
	code := m.Run()
	t.Stop()
	os.Exit(code)
}

func TestClusterRbacReconcile(t *testing.T) {
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterInfoName,
			Namespace:         clusterInfoNamespace,
			CreationTimestamp: now,
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.GroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	clusterv1beta1.AddToScheme(s)

	c := fake.NewFakeClientWithScheme(s, clusterInfo)

	fr := NewClusterInfoReconciler()

	fr.Client = c
	fr.Agent = agent.NewAgent("c1", fr.KubeClient)

	_, err := fr.Reconcile(clusterc1Request)
	if err != nil {
		t.Errorf("Failed to run reconcile cluster. error: %v", err)
	}
}
func NewClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(
		kubeNode, kubeEndpoints, ocpConsole, agentIngress, kubeMonitoringSecret, agentService)
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/acm-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

func TestClusterInfoReconciler_getMasterAddresses(t *testing.T) {
	cir := NewClusterInfoReconciler()
	endpointaddr, endpointport, clusterurl := cir.getMasterAddresses()
	if len(endpointaddr) < 1 || len(endpointport) < 1 {
		t.Errorf("Failed to get clusterinfo. endpointaddr:%v, endpointport:%v, clusterurl:%v", endpointaddr, endpointport, clusterurl)
	}
	cir.KubeClient.CoreV1().ConfigMaps("openshift-console").Delete("console-config", &metav1.DeleteOptions{})
	endpointaddr, endpointport, clusterurl = cir.getMasterAddresses()
	if len(endpointaddr) < 1 || len(endpointport) < 1 {
		t.Errorf("Failed to get clusterinfo. endpointaddr:%v, endpointport:%v, clusterurl:%v", endpointaddr, endpointport, clusterurl)
	}

	coreEndpointAddr, coreEndpointPort, err := cir.readAgentConfig()
	if err != nil {
		t.Errorf("Failed to read agent config. coreEndpoindAddr:%v, coreEndpointPort:%v, err:%v", coreEndpointAddr, coreEndpointPort, err)
	}

	err = cir.setEndpointAddressFromService(coreEndpointAddr, coreEndpointPort)
	if err != nil {
		t.Errorf("Failed to read agent config. coreEndpoindAddr:%v, coreEndpointPort:%v, err:%v", coreEndpointAddr, coreEndpointPort, err)
	}
	err = cir.setEndpointAddressFromRoute(coreEndpointAddr)
	if err == nil {
		t.Errorf("set endpoint should have error")
	}
	version := cir.getVersion()
	if version == "" {
		t.Errorf("Failed to get version")
	}
	_, err = cir.getNodeList()
	if err != nil {
		t.Errorf("Failed to get nodelist, err: %v", err)
	}
	_, err = cir.getDistributionInfo()
	if err != nil {
		t.Errorf("Failed to get distributeinfo, err: %v", err)
	}
}
