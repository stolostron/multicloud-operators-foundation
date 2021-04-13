package controllers

import (
	"context"
	tlog "github.com/go-logr/logr/testing"
	clusterfake "github.com/open-cluster-management/api/client/cluster/clientset/versioned/fake"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
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

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
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

func newNode() *corev1.Node {
	return &corev1.Node{
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
}

func newMonitoringSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("aaa"),
			"tls.key": []byte("aaa"),
		},
	}
}

func newAgentIngress() *extensionv1beta1.Ingress {
	return &extensionv1beta1.Ingress{
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
}

func newAgentService() *corev1.Service {
	return &corev1.Service{
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
}

func newClusterClaimList() *clusterv1alpha1.ClusterClaimList {
	return &clusterv1alpha1.ClusterClaimList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items: []clusterv1alpha1.ClusterClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterclaim.ClaimOCMConsoleURL,
				},
				Spec: clusterv1alpha1.ClusterClaimSpec{
					Value: "https://abc.com",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterclaim.ClaimOCMKubeVersion,
				},
				Spec: clusterv1alpha1.ClusterClaimSpec{
					Value: "v1.20.0",
				},
			},
		},
	}
}

func NewClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(
		newNode(), newAgentIngress(), newMonitoringSecret(), newAgentService())
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset(newClusterClaimList())
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		ClusterClient: fakeClusterClient,
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

func NewFailedClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(
		newNode(), newMonitoringSecret())
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset()
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		ClusterClient: fakeClusterClient,
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
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

	assert.Equal(t, updatedClusterInfo.Status.Version, "v1.20.0")
	assert.Equal(t, updatedClusterInfo.Status.ConsoleURL, "https://abc.com")

	if meta.IsStatusConditionFalse(updatedClusterInfo.Status.Conditions, clusterv1beta1.ManagedClusterInfoSynced) {
		t.Errorf("failed to update synced condtion")
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

func TestGetNodeList(t *testing.T) {
	tests := []struct {
		name             string
		expectNoteStatus []clusterv1beta1.NodeStatus
		expectError      error
	}{
		{
			name: "get node status",
			expectNoteStatus: []clusterv1beta1.NodeStatus{
				{
					Name: "node1",
					Labels: map[string]string{
						"node-role.kubernetes.io/worker": "",
					},
					Capacity: map[clusterv1beta1.ResourceName]resource.Quantity{
						clusterv1beta1.ResourceCPU:    {},
						clusterv1beta1.ResourceMemory: {},
					},
					Conditions: []clusterv1beta1.NodeCondition{
						{
							Type: corev1.NodeReady,
						},
					},
				}},
			expectError: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cir := NewClusterInfoReconciler()
			nodeList, err := cir.getNodeList()
			assert.Equal(t, err, test.expectError)
			assert.Equal(t, len(nodeList), len(test.expectNoteStatus))
			assert.Equal(t, nodeList[0].Name, test.expectNoteStatus[0].Name)
			assert.Equal(t, len(nodeList[0].Labels), len(test.expectNoteStatus[0].Labels))
		})
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
		expectDesiredVersion string
		expectUpgradeFail    bool
		expectError          string
	}{
		{
			name:                 "OCP4.x",
			dynamicClient:        dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x", clusterVersions)),
			expectDesiredVersion: "4.5.11",
			expectUpgradeFail:    false,
			expectError:          "",
		},
		{
			name:          "OCP3.x",
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("3", "")),
			expectError:   "",
		},
		{
			name:                 "UpgradeFail",
			dynamicClient:        dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x", clusterVersionsFail)),
			expectDesiredVersion: "4.5.11",
			expectUpgradeFail:    true,
			expectError:          "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cir.ManagedClusterDynamicClient = test.dynamicClient
			info, err := cir.getOCPDistributionInfo()
			if err != nil {
				assert.Equal(t, err.Error(), test.expectError)
			}
			if test.expectDesiredVersion != "" {
				assert.Equal(t, info.DesiredVersion, test.expectDesiredVersion)
			}
			assert.Equal(t, info.UpgradeFailed, test.expectUpgradeFail)
		})
	}
}
