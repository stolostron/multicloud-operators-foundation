package controllers

import (
	"context"
	tlog "github.com/go-logr/logr/testing"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	routev1Fake "github.com/openshift/client-go/route/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	stdlog "log"
	"os"
	"testing"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/stretchr/testify/assert"
	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
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
			Name:              "foundation-ingress-testcluster-agent",
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

func TestClusterInfoReconcile(t *testing.T) {
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterInfoName,
			Namespace:         clusterInfoNamespace,
			CreationTimestamp: now,
		},
		Status: clusterv1beta1.ClusterInfoStatus{
			Conditions: []metav1.Condition{
				{
					Type:   clusterv1.ManagedClusterConditionAvailable,
					Status: metav1.ConditionTrue,
				},
			},
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

	updatedClusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err = fr.Get(context.Background(), clusterc1Request.NamespacedName, updatedClusterInfo)
	if err != nil {
		t.Errorf("failed get updated clusterinfo ")
	}

	if meta.IsStatusConditionFalse(updatedClusterInfo.Status.Conditions, clusterv1beta1.ManagedClusterInfoSynced) {
		t.Errorf("failed to update synced condtion")
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
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

func TestFailedClusterInfoReconcile(t *testing.T) {
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterInfoName,
			Namespace:         clusterInfoNamespace,
			CreationTimestamp: now,
		},
		Status: clusterv1beta1.ClusterInfoStatus{
			Conditions: []metav1.Condition{
				{
					Type:   clusterv1.ManagedClusterConditionAvailable,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.GroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	clusterv1beta1.AddToScheme(s)

	c := fake.NewFakeClientWithScheme(s, clusterInfo)

	fr := NewFailedClusterInfoReconciler()

	fr.Client = c
	fr.Agent = agent.NewAgent("c1", fr.KubeClient)

	_, err := fr.Reconcile(clusterc1Request)
	if err != nil {
		t.Errorf("Failed to run reconcile cluster. error: %v", err)
	}

	updatedClusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err = fr.Get(context.Background(), clusterc1Request.NamespacedName, updatedClusterInfo)
	if err != nil {
		t.Errorf("failed get updated clusterinfo ")
	}

	if meta.IsStatusConditionTrue(updatedClusterInfo.Status.Conditions, clusterv1beta1.ManagedClusterInfoSynced) {
		t.Errorf("failed to update synced condtion")
	}
}

func NewFailedClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(
		kubeNode, ocpConsole, kubeMonitoringSecret)
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
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
	cir.KubeClient.CoreV1().ConfigMaps("openshift-console").Delete(context.TODO(), "console-config", metav1.DeleteOptions{})
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
	_, _, err = cir.getDistributionInfoAndClusterID()
	if err != nil {
		t.Errorf("Failed to get distributeinfo, err: %v", err)
	}
}

const (
	clusterVersions = `{
  "apiVersion": "config.openshift.io/v1",
  "kind": "ClusterVersion",
  "metadata": {
	"name": "version"
  },
  "spec": {
    "channel": "stable-4.5",
    "clusterID": "ffd989a0-8391-426d-98ac-86ae6d051433",
    "upstream": "https://api.openshift.com/api/upgrades_info/v1/graph"
  },
 "status": {
	"availableUpdates": [
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:95cfe9273aecb9a0070176210477491c347f8e69e41759063642edf8bb8aceb6",
                "version": "4.5.14"
	  },
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:adb5ef06c54ff75ca9033f222ac5e57f2fd82e49bdd84f737d460ae542c8af60",
		"version": "4.5.16"
	  },
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:8d104847fc2371a983f7cb01c7c0a3ab35b7381d6bf7ce355d9b32a08c0031f0",
		"version": "4.5.13"
	  },
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:6dde1b3ad6bec35364b2b89172cfea0459df75c99a4031f6f7b2a94eb9b166cf",
		"version": "4.5.17"
	  },
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:bae5510f19324d8e9c313aaba767e93c3a311902f5358fe2569e380544d9113e",
		"version": "4.5.19"
	  },
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:1df294ebe5b84f0eeceaa85b2162862c390143f5e84cda5acc22cc4529273c4c",
		"version": "4.5.15"
	  },
	  {
 	 	"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:72e3a1029884c70c584a0cadc00c36ee10764182425262fb23f77f32732ef366",
		"version": "4.5.18"
	  },
	  {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:78b878986d2d0af6037d637aa63e7b6f80fc8f17d0f0d5b077ac6aca83f792a0",
		"version": "4.5.20"
	  }
	],
	"conditions": [
	  {
		"lastTransitionTime": "2020-09-30T09:00:07Z",
		"message": "Done applying 4.5.11",
		"status": "True",
		"type": "Available"
	  },
	  {
		"lastTransitionTime": "2020-09-30T08:45:02Z",
		"status": "False",
		"type": "Failing"
	  },
	  {
		"lastTransitionTime": "2020-09-30T09:00:07Z",
		"message": "Cluster version is 4.5.11",
		"status": "False",
		"type": "Progressing"
	  },
	  {
		"lastTransitionTime": "2020-10-06T13:50:43Z",
		"status": "True",
		"type": "RetrievedUpdates"
	  }
	],
	"desired": {
	  "force": false,
	  "image": "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
	  "version": "4.5.11"
	},
 	"history": [
	  {
	 	"completionTime": "2020-09-30T09:00:07Z",
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
		"startedTime": "2020-09-30T08:36:46Z",
		"state": "Completed",
		"verified": false,
		"version": "4.5.11"
	  }
	],
	"observedGeneration": 1,
	"versionHash": "4lK_pl-YbSw="
  }
}`
	clusterVersionsFail = `{
  "apiVersion": "config.openshift.io/v1",
  "kind": "ClusterVersion",
  "metadata": {
	"name": "version"
  },
  "spec": {
    "channel": "stable-4.5",
    "clusterID": "ffd989a0-8391-426d-98ac-86ae6d051433",
    "upstream": "https://api.openshift.com/api/upgrades_info/v1/graph"
  },
 "status": {
	"availableUpdates": [
	 {
		"force": false,
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:78b878986d2d0af6037d637aa63e7b6f80fc8f17d0f0d5b077ac6aca83f792a0",
		"version": "4.5.20"
	  }
	],
	"conditions": [
	  {
		"lastTransitionTime": "2020-09-30T09:00:07Z",
		"message": "Done applying 4.5.11",
		"status": "True",
		"type": "Available"
	  },
	  {
		"lastTransitionTime": "2020-09-30T08:45:02Z",
		"status": "True",
		"type": "Failing"
	  },
	  {
		"lastTransitionTime": "2020-09-30T09:00:07Z",
		"message": "Cluster version is 4.5.11",
		"status": "False",
		"type": "Progressing"
	  },
	  {
		"lastTransitionTime": "2020-10-06T13:50:43Z",
		"status": "True",
		"type": "RetrievedUpdates"
	  }
	],
	"desired": {
	  "force": false,
	  "image": "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
	  "version": "4.5.11"
	},
 	"history": [
	  {
	 	"completionTime": "2020-09-30T09:00:07Z",
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
		"startedTime": "2020-09-30T08:36:46Z",
		"state": "Completed",
		"verified": false,
		"version": "4.5.3"
	  }
	],
	"observedGeneration": 1,
	"versionHash": "4lK_pl-YbSw="
  }
}`
)

func newClusterVersions(version, clusterVersions string) *unstructured.Unstructured {
	if version == "3" {
		return &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "test/v1",
				"kind":       "test",
				"metadata": map[string]interface{}{
					"name": "test",
				},
			},
		}
	}
	if version == "4.x" {
		obj := unstructured.Unstructured{}
		obj.UnmarshalJSON([]byte(clusterVersions))
		return &obj
	}
	return nil
}

func TestClusterInfoReconciler_getOCPDistributionInfo(t *testing.T) {
	cir := NewClusterInfoReconciler()

	tests := []struct {
		name                 string
		dynamicClient        dynamic.Interface
		expectVersion        string
		expectDesiredVersion string
		expectUpgradeFail    bool
		expectError          string
	}{
		{
			name:                 "OCP4.x",
			dynamicClient:        dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x", clusterVersions)),
			expectVersion:        "4.5.11",
			expectDesiredVersion: "4.5.11",
			expectUpgradeFail:    false,
			expectError:          "",
		},
		{
			name:          "OCP3.x",
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("3", "")),
			expectVersion: "3",
			expectError:   "",
		},
		{
			name:                 "UpgradeFail",
			dynamicClient:        dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x", clusterVersionsFail)),
			expectVersion:        "4.5.3",
			expectDesiredVersion: "4.5.11",
			expectUpgradeFail:    true,
			expectError:          "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cir.ManagedClusterDynamicClient = test.dynamicClient
			info, _, err := cir.getOCPDistributionInfo()
			if err != nil {
				assert.Equal(t, err.Error(), test.expectError)
			}
			assert.Equal(t, info.Version, test.expectVersion)
			if test.expectDesiredVersion != "" {
				assert.Equal(t, info.DesiredVersion, test.expectDesiredVersion)
			}
			assert.Equal(t, info.UpgradeFailed, test.expectUpgradeFail)
		})
	}
}

func NewClusterInfoReconcilerWithNodes(cloudVendorType clusterv1beta1.CloudVendorType,
	kubeVendorType clusterv1beta1.KubeVendorType) *ClusterInfoReconciler {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
	}

	switch cloudVendorType {
	case clusterv1beta1.CloudVendorAWS:
		node.Spec.ProviderID = "aws:///us-east-1a/i-07bacee3f60562daa"
	case clusterv1beta1.CloudVendorAzure:
		node.Spec.ProviderID = "azure:///subscriptions/03e5f0ef-0741-442a-bc1b-ba34ceb3f63f/resourceGroups/yzwaz-rlpjx-rg/providers/Microsoft.Compute/virtualMachines/yzwaz-rlpjx-master-0"
	case clusterv1beta1.CloudVendorGoogle:
		node.Spec.ProviderID = "gce:///abc"
	case clusterv1beta1.CloudVendorIBM:
		node.Spec.ProviderID = "ibm:///abc"
	case clusterv1beta1.CloudVendorVSphere:
		node.Spec.ProviderID = "vsphere://421a27ac-bb12-f6e6-48cb-f2aa74e56156"
	case clusterv1beta1.CloudVendorOpenStack:
		node.Spec.ProviderID = "openstack:///dda1f31a-3dfb-435a-9e1d-16149a8dd628"
	}

	fakeKubeClient := kubefake.NewSimpleClientset(node)
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()

	switch kubeVendorType {
	case clusterv1beta1.KubeVendorOSD:
		project := metav1.APIResource{
			Name:         "projects",
			SingularName: "project",
			Namespaced:   false,
			Kind:         "Project",
		}
		managed := metav1.APIResource{
			Name:         "subjectpermissions",
			SingularName: "subjectpermission",
			Namespaced:   true,
			Kind:         "SubjectPermission",
		}
		projectResource := &metav1.APIResourceList{
			GroupVersion: "project.openshift.io/v1",
			APIResources: []metav1.APIResource{project},
		}
		managedResource := &metav1.APIResourceList{
			GroupVersion: "managed.openshift.io/v1alpha1",
			APIResources: []metav1.APIResource{managed},
		}

		fakeKubeClient.Resources = append(fakeKubeClient.Resources, projectResource, managedResource)
	}

	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

func TestGetVendor(t *testing.T) {
	tests := []struct {
		name                  string
		clusterInfoReconciler *ClusterInfoReconciler
		gitVersion            string
		isOpenShift           bool
		expectCloudVendor     clusterv1beta1.CloudVendorType
		expectKubeVendor      clusterv1beta1.KubeVendorType
	}{
		{
			name:                  "aws-openshift",
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorAWS, ""),
			gitVersion:            "v1.18.3+2fbd7c7",
			isOpenShift:           true,
			expectCloudVendor:     clusterv1beta1.CloudVendorAWS,
			expectKubeVendor:      clusterv1beta1.KubeVendorOpenShift,
		},
		{
			name:                  "azure",
			gitVersion:            "v1.18.3+aks",
			isOpenShift:           false,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorAzure, ""),
			expectCloudVendor:     clusterv1beta1.CloudVendorAzure,
			expectKubeVendor:      clusterv1beta1.KubeVendorAKS,
		},
		{
			name:                  "google",
			gitVersion:            "v1.18.3+gke",
			isOpenShift:           false,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorGoogle, ""),
			expectCloudVendor:     clusterv1beta1.CloudVendorGoogle,
			expectKubeVendor:      clusterv1beta1.KubeVendorGKE,
		},
		{
			name:                  "IBM",
			gitVersion:            "v1.18.3+icp",
			isOpenShift:           false,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorIBM, ""),
			expectCloudVendor:     clusterv1beta1.CloudVendorIBM,
			expectKubeVendor:      clusterv1beta1.KubeVendorICP,
		},
		{
			name:                  "vsphere",
			gitVersion:            "v1.18.3+2fbd7c7",
			isOpenShift:           true,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorVSphere, ""),
			expectCloudVendor:     clusterv1beta1.CloudVendorVSphere,
			expectKubeVendor:      clusterv1beta1.KubeVendorOpenShift,
		},
		{
			name:                  "openstack",
			gitVersion:            "v1.18.3+2fbd7c7",
			isOpenShift:           true,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorOpenStack, ""),
			expectCloudVendor:     clusterv1beta1.CloudVendorOpenStack,
			expectKubeVendor:      clusterv1beta1.KubeVendorOpenShift,
		},
		{
			name:                  "aws-osd",
			gitVersion:            "v1.18.3+2fbd7c7",
			isOpenShift:           true,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorAWS, clusterv1beta1.KubeVendorOSD),
			expectCloudVendor:     clusterv1beta1.CloudVendorAWS,
			expectKubeVendor:      clusterv1beta1.KubeVendorOSD,
		},
		{
			name:                  "others",
			gitVersion:            "v1.18.3+2fbd7c7",
			isOpenShift:           false,
			clusterInfoReconciler: NewClusterInfoReconcilerWithNodes(clusterv1beta1.CloudVendorOther, ""),
			expectCloudVendor:     clusterv1beta1.CloudVendorOther,
			expectKubeVendor:      clusterv1beta1.KubeVendorOther,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeVendor, cloudVendor := test.clusterInfoReconciler.getVendor(test.gitVersion, test.isOpenShift)
			assert.Equal(t, test.expectCloudVendor, cloudVendor)
			assert.Equal(t, test.expectKubeVendor, kubeVendor)
		})
	}
}

func NewOCPClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(
		kubeNode, ocpConsole, kubeMonitoringSecret)
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()

	project := metav1.APIResource{
		Name:         "projects",
		SingularName: "project",
		Namespaced:   false,
		Kind:         "Project",
	}
	ocpResource := &metav1.APIResourceList{
		GroupVersion: "project.openshift.io/v1",
		APIResources: []metav1.APIResource{project},
	}
	fakeKubeClient.Resources = append(fakeKubeClient.Resources, ocpResource)
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

const (
	awsOCPInfraConfig = `{
    "apiVersion": "config.openshift.io/v1",
    "kind": "Infrastructure",
    "metadata": {
        "name": "cluster"
    },
    "spec": {
        "cloudConfig": {
        "name": ""
        },
        "platformSpec": {
        "aws": {},
        "type": "AWS"
        }
    },
    "status": {
        "apiServerInternalURI": "https://api-int.osd-test.wu67.s1.devshift.org:6443",
        "apiServerURL": "https://api.osd-test.wu67.s1.devshift.org:6443",
        "etcdDiscoveryDomain": "osd-test.wu67.s1.devshift.org",
        "infrastructureName": "ocp-aws",
        "platform": "AWS",
        "platformStatus": {
        "aws": {
            "region": "region-aws"
        },
        "type": "AWS"
        } 
    }
}`

	gcpInfraConfig = `{
    "apiVersion": "config.openshift.io/v1",
    "kind": "Infrastructure",
    "metadata": {
        "name": "cluster"
    },
    "spec": {
        "cloudConfig": {
            "name": ""
        },
        "platformSpec": {
            "gcp": {},
            "type": "GCP"
        }
    },
    "status": {
        "apiServerInternalURI": "https://api-int.osd-test.wu67.s1.devshift.org:6443",
        "apiServerURL": "https://api.osd-test.wu67.s1.devshift.org:6443",
        "etcdDiscoveryDomain": "osd-test.wu67.s1.devshift.org",
        "infrastructureName": "ocp-gcp",
        "platform": "GCP",
        "platformStatus": {
            "gcp": {
                "region": "region-gcp"
            },
            "type": "GCP"
        }
    }
}`
)

func newInfraConfig(platformType string) *unstructured.Unstructured {
	obj := unstructured.Unstructured{}
	switch platformType {
	case AWSPlatformType:
		obj.UnmarshalJSON([]byte(awsOCPInfraConfig))
	case GCPPlatformType:
		obj.UnmarshalJSON([]byte(gcpInfraConfig))
	}

	return &obj
}

func TestGetRegion(t *testing.T) {
	tests := []struct {
		name                  string
		clusterInfoReconciler *ClusterInfoReconciler
		dynamicClient         dynamic.Interface
		expectRegion          string
	}{
		{
			name:                  "aws OCP",
			clusterInfoReconciler: NewOCPClusterInfoReconciler(),
			dynamicClient:         dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(AWSPlatformType)),
			expectRegion:          "region-aws",
		},
		{
			name:                  "GCP",
			clusterInfoReconciler: NewOCPClusterInfoReconciler(),
			dynamicClient:         dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(GCPPlatformType)),
			expectRegion:          "region-gcp",
		},
		{
			name:                  "Non-OCP",
			clusterInfoReconciler: NewClusterInfoReconciler(),
			dynamicClient:         dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
			expectRegion:          "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.clusterInfoReconciler.ManagedClusterDynamicClient = test.dynamicClient
			region := test.clusterInfoReconciler.getClusterRegion()
			assert.Equal(t, test.expectRegion, region)
		})
	}
}

func TestGetInfraConfig(t *testing.T) {
	tests := []struct {
		name                  string
		clusterInfoReconciler *ClusterInfoReconciler
		dynamicClient         dynamic.Interface
		expectInfra           string
	}{
		{
			name:                  "aws OCP",
			clusterInfoReconciler: NewOCPClusterInfoReconciler(),
			dynamicClient:         dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(AWSPlatformType)),
			expectInfra:           "{\"infraName\":\"ocp-aws\"}",
		},
		{
			name:                  "GCP",
			clusterInfoReconciler: NewOCPClusterInfoReconciler(),
			dynamicClient:         dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(GCPPlatformType)),
			expectInfra:           "{\"infraName\":\"ocp-gcp\"}",
		},
		{
			name:                  "Non-OCP",
			clusterInfoReconciler: NewClusterInfoReconciler(),
			dynamicClient:         dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
			expectInfra:           "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.clusterInfoReconciler.ManagedClusterDynamicClient = test.dynamicClient
			infraName := test.clusterInfoReconciler.getInfraConfig()
			assert.Equal(t, test.expectInfra, infraName)
		})
	}
}
