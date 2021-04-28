package controllers

import (
	"context"
	stdlog "log"
	"os"
	"testing"
	"time"

	tlog "github.com/go-logr/logr/testing"
	clusterfake "github.com/open-cluster-management/api/client/cluster/clientset/versioned/fake"
	clusterinformers "github.com/open-cluster-management/api/client/cluster/informers/externalversions"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	routev1Fake "github.com/openshift/client-go/route/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/scheme"

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

var clusterName = "c1"
var clusterc1Request = reconcile.Request{
	NamespacedName: types.NamespacedName{
		Name: clusterName, Namespace: clusterName}}

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

func newClusterClaimList() []runtime.Object {
	return []runtime.Object{
		&clusterv1alpha1.ClusterClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterclaim.ClaimOCMConsoleURL,
			},
			Spec: clusterv1alpha1.ClusterClaimSpec{
				Value: "https://abc.com",
			},
		},
		&clusterv1alpha1.ClusterClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterclaim.ClaimOCMKubeVersion,
			},
			Spec: clusterv1alpha1.ClusterClaimSpec{
				Value: "v1.20.0",
			},
		},
	}
}

func NewClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(
		newNode(), newAgentIngress(), newMonitoringSecret(), newAgentService())
	fakeRouteV1Client := routev1Fake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset(newClusterClaimList()...)
	informerFactory := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	store := informerFactory.Core().V1().Nodes().Informer().GetStore()
	store.Add(newNode())
	clusterStore := clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Informer().GetStore()
	for _, item := range newClusterClaimList() {
		clusterStore.Add(item)
	}
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		ClusterName:   clusterName,
		NodeLister:    informerFactory.Core().V1().Nodes().Lister(),
		ClaimLister:   clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
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
	informerFactory := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	store := informerFactory.Core().V1().Nodes().Informer().GetStore()
	store.Add(newNode())
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		KubeClient:    fakeKubeClient,
		ClusterName:   clusterName,
		NodeLister:    informerFactory.Core().V1().Nodes().Lister(),
		ClaimLister:   clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

func TestClusterInfoReconcile(t *testing.T) {
	ctx := context.Background()
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterName,
			Namespace:         clusterName,
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

	_, err := fr.Reconcile(ctx, clusterc1Request)
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
	ctx := context.Background()
	// Create new cluster
	now := metav1.Now()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              clusterName,
			Namespace:         clusterName,
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

	_, err := fr.Reconcile(ctx, clusterc1Request)
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
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:3e855ad88f46ad1b7f56c312f078ca6adaba623c5d4b360143f9f82d2f349741",
		"url": "https://access.redhat.com/errata/RHBA-2021:0308",
		"version": "4.6.16"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:5c3618ab914eb66267b7c552a9b51c3018c3a8f8acf08ce1ff7ae4bfdd3a82bd",
		"url": "https://access.redhat.com/errata/RHSA-2021:0037",
		"version": "4.6.12"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:08ef16270e643a001454410b22864db6246d782298be267688a4433d83f404f4",
		"url": "https://access.redhat.com/errata/RHBA-2021:0510",
		"version": "4.6.18"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:a7b23f38d1e5be975a6b516739689673011bdfa59a7158dc6ca36cefae169c18",
		"url": "https://access.redhat.com/errata/RHSA-2021:0424",
		"version": "4.6.17"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:ac5bbe391f9f5db07b8a710cfda1aee80f6eb3bf37a3c44a5b89763957d8d5ad",
		"url": "https://access.redhat.com/errata/RHBA-2021:0674",
		"version": "4.6.20"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:47df4bfe1cfd6d63dd2e880f00075ed1d37f997fd54884ed823ded9f5d96abfc",
		"url": "https://access.redhat.com/errata/RHBA-2021:0634",
		"version": "4.6.19"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:b70f550e3fa94af2f7d60a3437ec0275194db36f2dc49991da2336fe21e2824c",
		"url": "https://access.redhat.com/errata/RHBA-2021:0235",
		"version": "4.6.15"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:8a9e40df2a19db4cc51dc8624d54163bef6e88b7d88cc0f577652ba25466e338",
		"url": "https://access.redhat.com/errata/RHBA-2021:0171",
		"version": "4.6.13"
	},
	{
		"channels": [
			"candidate-4.6",
			"candidate-4.7",
			"eus-4.6",
			"fast-4.6",
			"fast-4.7",
			"stable-4.6",
			"stable-4.7"
		],
		"image": "quay.io/openshift-release-dev/ocp-release@sha256:6ae80e777c206b7314732aff542be105db892bf0e114a6757cb9e34662b8f891",
		"url": "https://access.redhat.com/errata/RHBA-2021:0753",
		"version": "4.6.21"
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
	  "channels": [
		"candidate-4.6",
		"candidate-4.7",
		"eus-4.6",
		"fast-4.6",
		"fast-4.7",
		"stable-4.6",
		"stable-4.7"
	  ],
	  "image": "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
	  "url": "https://access.redhat.com/errata/RHSA-2020:5259",
	  "version": "4.6.8"
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
		name                      string
		dynamicClient             dynamic.Interface
		expectDesiredVersion      string
		expectUpgradeFail         bool
		expectAvailableUpdatesLen int
		expectChannelAndURL       bool
		expectHistoryLen          int
		expectError               string
	}{
		{
			name:                      "OCP4.x",
			dynamicClient:             dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x", clusterVersions)),
			expectDesiredVersion:      "4.6.8",
			expectUpgradeFail:         false,
			expectAvailableUpdatesLen: 9,
			expectChannelAndURL:       true,
			expectHistoryLen:          1,
			expectError:               "",
		},
		{
			name:          "OCP3.x",
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("3", "")),
			expectError:   "",
		},
		{
			name:                      "UpgradeFail",
			dynamicClient:             dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x", clusterVersionsFail)),
			expectDesiredVersion:      "4.5.11",
			expectUpgradeFail:         true,
			expectAvailableUpdatesLen: 1,
			expectChannelAndURL:       false,
			expectHistoryLen:          1,
			expectError:               "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cir.ManagedClusterDynamicClient = test.dynamicClient
			info, err := cir.getOCPDistributionInfo()
			if err != nil {
				assert.Equal(t, err.Error(), test.expectError)
			}

			assert.Equal(t, info.DesiredVersion, test.expectDesiredVersion)
			assert.Equal(t, info.UpgradeFailed, test.expectUpgradeFail)
			assert.Equal(t, len(info.VersionAvailableUpdates), test.expectAvailableUpdatesLen)
			if len(info.VersionAvailableUpdates) != 0 {
				for _, update := range info.VersionAvailableUpdates {
					assert.NotEqual(t, update.Version, "")
					assert.NotEqual(t, update.Image, "")
					if test.expectChannelAndURL {
						assert.NotEqual(t, update.URL, "")
						assert.NotEqual(t, len(update.Channels), 0)
					} else {
						assert.Equal(t, update.URL, "")
						assert.Equal(t, len(update.Channels), 0)
					}

				}
			}
			assert.Equal(t, len(info.VersionHistory), test.expectHistoryLen)
			if len(info.VersionHistory) != 0 {
				for _, history := range info.VersionHistory {
					assert.NotEqual(t, history.Version, "")
					assert.NotEqual(t, history.Image, "")
					assert.Equal(t, history.State, "Completed")
					assert.Equal(t, history.Verified, false)
				}
			}
		})
	}
}
