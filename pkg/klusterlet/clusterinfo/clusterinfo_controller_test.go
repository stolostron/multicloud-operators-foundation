package controllers

import (
	"context"
	stdlog "log"
	"os"
	"reflect"
	"testing"
	"time"

	ocinfrav1 "github.com/openshift/api/config/v1"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	apiconfigv1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"

	tlog "github.com/go-logr/logr/testing"
	clusterfake "github.com/open-cluster-management/api/client/cluster/clientset/versioned/fake"
	clusterinformers "github.com/open-cluster-management/api/client/cluster/informers/externalversions"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	routefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
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
		newAgentIngress(), newMonitoringSecret(), newAgentService())
	fakeRouteV1Client := routefake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset(newClusterClaimList()...)
	informerFactory := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	clusterStore := clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Informer().GetStore()
	for _, item := range newClusterClaimList() {
		clusterStore.Add(item)
	}
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		NodeLister:    informerFactory.Core().V1().Nodes().Lister(),
		KubeClient:    fakeKubeClient,
		ClusterName:   clusterName,
		ClaimLister:   clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
		RouteV1Client: fakeRouteV1Client,
		AgentAddress:  "127.0.0.1:8000",
		AgentIngress:  "kube-system/foundation-ingress-testcluster-agent",
		AgentRoute:    "AgentRoute",
		AgentService:  "kube-system/agent",
	}
}

func NewFailedClusterInfoReconciler() *ClusterInfoReconciler {
	fakeKubeClient := kubefake.NewSimpleClientset(newMonitoringSecret())
	fakeRouteV1Client := routefake.NewSimpleClientset()
	fakeClusterClient := clusterfake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	return &ClusterInfoReconciler{
		Log:           tlog.NullLogger{},
		NodeLister:    informerFactory.Core().V1().Nodes().Lister(),
		KubeClient:    fakeKubeClient,
		ClusterName:   clusterName,
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

func TestClusterInfoReconciler_getOCPDistributionInfo(t *testing.T) {
	cir := NewClusterInfoReconciler()
	tests := []struct {
		name                      string
		configV1Client            openshiftclientset.Interface
		expectChannel             string
		expectDesiredVersion      string
		expectDesiredChannelLen   int
		expectUpgradeFail         bool
		expectAvailableUpdatesLen int
		expectChannelAndURL       bool
		expectHistoryLen          int
		expectError               string
	}{
		{
			name:                      "OCP4.x",
			configV1Client:            newConfigV1Client("4.x", false),
			expectChannel:             "stable-4.5",
			expectDesiredVersion:      "4.6.8",
			expectDesiredChannelLen:   7,
			expectUpgradeFail:         false,
			expectAvailableUpdatesLen: 2,
			expectChannelAndURL:       true,
			expectHistoryLen:          1,
			expectError:               "",
		},
		{
			name:           "OCP3.x",
			configV1Client: newConfigV1Client("3", true),
			expectError:    "",
		},
		{
			name:                      "UpgradeFail",
			configV1Client:            newConfigV1Client("4.x", true),
			expectChannel:             "stable-4.5",
			expectDesiredChannelLen:   7,
			expectDesiredVersion:      "4.6.8",
			expectUpgradeFail:         true,
			expectAvailableUpdatesLen: 2,
			expectChannelAndURL:       false,
			expectHistoryLen:          1,
			expectError:               "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cir.ConfigV1Client = test.configV1Client
			info, err := cir.getOCPDistributionInfo(context.Background())
			if err != nil {
				assert.Equal(t, err.Error(), test.expectError)
			}

			assert.Equal(t, info.Channel, test.expectChannel)
			assert.Equal(t, info.DesiredVersion, test.expectDesiredVersion)
			assert.Equal(t, len(info.Desired.Channels), test.expectDesiredChannelLen)
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

func newClusterVersion() *apiconfigv1.ClusterVersion {
	return &apiconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: apiconfigv1.ClusterVersionSpec{
			ClusterID: "ffd989a0-8391-426d-98ac-86ae6d051433",
			Upstream:  "https://api.openshift.com/api/upgrades_info/v1/graph",
			Channel:   "stable-4.5",
		},
		Status: apiconfigv1.ClusterVersionStatus{
			ObservedGeneration: 1,
			VersionHash:        "4lK_pl-YbSw=",
			Desired: apiconfigv1.Release{
				Channels: []string{
					"candidate-4.6",
					"candidate-4.7",
					"eus-4.6",
					"fast-4.6",
					"fast-4.7",
					"stable-4.6",
					"stable-4.7",
				},
				Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
				URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
				Version: "4.6.8",
			},
			History: []apiconfigv1.UpdateHistory{
				{
					Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
					State:    "Completed",
					Verified: false,
					Version:  "4.5.11",
				},
			},
			AvailableUpdates: []apiconfigv1.Release{
				{
					Channels: []string{
						"candidate-4.6",
						"candidate-4.7",
						"eus-4.6",
						"fast-4.6",
						"fast-4.7",
						"stable-4.6",
						"stable-4.7",
					},
					Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
					URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
					Version: "4.6.8",
				},
				{
					Channels: []string{
						"candidate-4.6",
						"candidate-4.7",
						"eus-4.6",
						"fast-4.6",
						"fast-4.7",
						"stable-4.6",
						"stable-4.7",
					},
					Image:   "quay.io/openshift-release-dev/ocp-release@sha256:3e855ad88f46ad1b7f56c312f078ca6adaba623c5d4b360143f9f82d2f349741",
					URL:     "https://access.redhat.com/errata/RHBA-2021:0308",
					Version: "4.6.16",
				},
			},
			Conditions: []apiconfigv1.ClusterOperatorStatusCondition{
				{
					Type:   "Failing",
					Status: "False",
				},
			},
		},
	}
}

func newFailingClusterVersion() *apiconfigv1.ClusterVersion {
	return &apiconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
		Spec: apiconfigv1.ClusterVersionSpec{
			ClusterID: "ffd989a0-8391-426d-98ac-86ae6d051433",
			Upstream:  "https://api.openshift.com/api/upgrades_info/v1/graph",
			Channel:   "stable-4.5",
		},
		Status: apiconfigv1.ClusterVersionStatus{
			ObservedGeneration: 1,
			VersionHash:        "4lK_pl-YbSw=",
			Desired: apiconfigv1.Release{
				Channels: []string{
					"candidate-4.6",
					"candidate-4.7",
					"eus-4.6",
					"fast-4.6",
					"fast-4.7",
					"stable-4.6",
					"stable-4.7",
				},
				Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
				URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
				Version: "4.6.8",
			},
			History: []apiconfigv1.UpdateHistory{
				{
					Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
					State:    "Completed",
					Verified: false,
					Version:  "4.5.11",
				},
			},
			AvailableUpdates: []apiconfigv1.Release{
				{
					Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
					Version: "4.6.8",
				},
				{
					Image:   "quay.io/openshift-release-dev/ocp-release@sha256:3e855ad88f46ad1b7f56c312f078ca6adaba623c5d4b360143f9f82d2f349741",
					Version: "4.6.16",
				},
			},
			Conditions: []apiconfigv1.ClusterOperatorStatusCondition{
				{
					Type:   "Failing",
					Status: "True",
				},
			},
		},
	}
}

func newConfigV1Client(version string, failingStatus bool) openshiftclientset.Interface {
	clusterVersion := &apiconfigv1.ClusterVersion{}
	if version == "4.x" {
		if failingStatus {
			clusterVersion = newFailingClusterVersion()
		} else {
			clusterVersion = newClusterVersion()
		}

	}

	return configfake.NewSimpleClientset(clusterVersion)
}

func TestOcpDistributionInfoUpdated(t *testing.T) {
	tests := []struct {
		name           string
		old            *clusterv1beta1.DistributionInfo
		new            *clusterv1beta1.DistributionInfo
		expectedUpdate bool
	}{
		{
			name:           "old and new are equal",
			expectedUpdate: false,
			old: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:          "v4.6",
					AvailableUpdates: []string{"v4.6.8", "v4.6.9"},
					DesiredVersion:   "v4.6",
					UpgradeFailed:    false,
					Channel:          "channel 4.6",
					Desired:          clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: []clusterv1beta1.OCPVersionRelease{
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.8",
						},
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.9",
						},
					},
					VersionHistory: []clusterv1beta1.OCPVersionUpdateHistory{
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.11",
						},
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.12",
						},
					},
				},
			},
			new: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:          "v4.6",
					AvailableUpdates: []string{"v4.6.9", "v4.6.8"},
					DesiredVersion:   "v4.6",
					UpgradeFailed:    false,
					Channel:          "channel 4.6",
					Desired:          clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: []clusterv1beta1.OCPVersionRelease{
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.9",
						},
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.8",
						},
					},
					VersionHistory: []clusterv1beta1.OCPVersionUpdateHistory{
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.12",
						},
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.11",
						},
					},
				},
			},
		},
		{
			name:           "old and new are not equal",
			expectedUpdate: true,
			old: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:          "v4.6",
					AvailableUpdates: []string{"v4.6.8", "v4.6.9"},
					DesiredVersion:   "v4.6",
					UpgradeFailed:    false,
					Channel:          "channel 4.6",
					Desired:          clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: []clusterv1beta1.OCPVersionRelease{
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.8",
						},
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.9",
						},
					},
					VersionHistory: []clusterv1beta1.OCPVersionUpdateHistory{
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.11",
						},
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.12",
						},
					},
				},
			},
			new: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:          "v4.6",
					AvailableUpdates: []string{"v4.6.9", "v4.6.8"},
					DesiredVersion:   "v4.6",
					UpgradeFailed:    false,
					Channel:          "channel 4.6",
					Desired:          clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: []clusterv1beta1.OCPVersionRelease{
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.9",
						},
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.10",
						},
					},
					VersionHistory: []clusterv1beta1.OCPVersionUpdateHistory{
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.12",
						},
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.13",
						},
					},
				},
			},
		},
		{
			name:           "old is nil",
			expectedUpdate: true,
			old: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:                 "v4.6",
					AvailableUpdates:        nil,
					DesiredVersion:          "v4.6",
					UpgradeFailed:           false,
					Channel:                 "channel 4.6",
					Desired:                 clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: nil,
					VersionHistory:          nil,
				},
			},
			new: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:          "v4.6",
					AvailableUpdates: []string{"v4.6.9", "v4.6.8"},
					DesiredVersion:   "v4.6",
					UpgradeFailed:    false,
					Channel:          "channel 4.6",
					Desired:          clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: []clusterv1beta1.OCPVersionRelease{
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.9",
						},
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.10",
						},
					},
					VersionHistory: []clusterv1beta1.OCPVersionUpdateHistory{
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.12",
						},
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.13",
						},
					},
				},
			},
		},
		{
			name:           "new is nil",
			expectedUpdate: true,
			old: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:          "v4.6",
					AvailableUpdates: []string{"v4.6.8", "v4.6.9"},
					DesiredVersion:   "v4.6",
					UpgradeFailed:    false,
					Channel:          "channel 4.6",
					Desired:          clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: []clusterv1beta1.OCPVersionRelease{
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.8",
						},
						{
							Image:   "quay.io/openshift-release-dev/ocp-release@sha256:6ddbf56b7f9776c0498f23a54b65a06b3b846c1012200c5609c4bb716b6bdcdf",
							URL:     "https://access.redhat.com/errata/RHSA-2020:5259",
							Version: "4.6.9",
						},
					},
					VersionHistory: []clusterv1beta1.OCPVersionUpdateHistory{
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.11",
						},
						{
							Image:    "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
							State:    "Completed",
							Verified: false,
							Version:  "4.5.12",
						},
					},
				},
			},
			new: &clusterv1beta1.DistributionInfo{
				Type: clusterv1beta1.DistributionTypeOCP,
				OCP: clusterv1beta1.OCPDistributionInfo{
					Version:                 "v4.6",
					AvailableUpdates:        nil,
					DesiredVersion:          "v4.6",
					UpgradeFailed:           false,
					Channel:                 "channel 4.6",
					Desired:                 clusterv1beta1.OCPVersionRelease{},
					VersionAvailableUpdates: nil,
					VersionHistory:          nil,
				},
			},
		},
	}

	for _, test := range tests {
		update := distributionInfoUpdated(test.old, test.new)
		assert.Equal(t, update, test.expectedUpdate)
	}
}

func newTestClusterInfoReconciler(existingCluster, existingKubeOjb, existingOcpOjb []runtime.Object) *ClusterInfoReconciler {
	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.GroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	clusterv1beta1.AddToScheme(s)

	return &ClusterInfoReconciler{
		Client:         fake.NewFakeClientWithScheme(s, existingCluster...),
		KubeClient:     kubefake.NewSimpleClientset(existingKubeOjb...),
		ConfigV1Client: configfake.NewSimpleClientset(existingOcpOjb...),
	}
}

func Test_getClientConfig(t *testing.T) {
	tests := []struct {
		name               string
		existingCluster    []runtime.Object
		existingKubeOjb    []runtime.Object
		existingOcpOjb     []runtime.Object
		expectClientConfig clusterv1beta1.ClientConfig
	}{
		{
			name: "Apiserver url not found",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
						},
					},
				},
			},
			existingKubeOjb:    []runtime.Object{},
			existingOcpOjb:     []runtime.Object{},
			expectClientConfig: clusterv1beta1.ClientConfig{},
		},
		{
			name: "clusterca not found",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
						},
					},
				},
			},
			existingKubeOjb: []runtime.Object{},
			existingOcpOjb: []runtime.Object{
				&ocinfrav1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.InfrastructureSpec{},
					Status: ocinfrav1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
			},
			expectClientConfig: clusterv1beta1.ClientConfig{},
		},
		{
			name: "add apiserver ca to ManagedClusterInfo",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
							OCP:  clusterv1beta1.OCPDistributionInfo{},
						},
					},
				},
			},
			existingKubeOjb: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "openshift-config",
					},
					Data: map[string][]byte{
						"tls.crt": []byte("fake-cert-data"),
						"tls.key": []byte("fake-key-data"),
					},
					Type: corev1.SecretTypeTLS,
				},
			},
			existingOcpOjb: []runtime.Object{
				&ocinfrav1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.InfrastructureSpec{},
					Status: ocinfrav1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
				&ocinfrav1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.APIServerSpec{
						ServingCerts: ocinfrav1.APIServerServingCerts{
							NamedCertificates: []ocinfrav1.APIServerNamedServingCert{
								{
									Names:              []string{"127.0.0.1"},
									ServingCertificate: ocinfrav1.SecretNameReference{Name: "test-secret"},
								},
							},
						},
					},
				},
			},
			expectClientConfig: clusterv1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("fake-cert-data"),
			},
		},
		{
			name: "add configmap ca to ManagedClusterInfo",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
						},
					},
				},
			},
			existingKubeOjb: []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kube-root-ca.crt",
						Namespace: "kube-public",
					},
					Data: map[string]string{
						"ca.crt": "configmap-fake-cert-data",
					},
				},
			},
			existingOcpOjb: []runtime.Object{
				&ocinfrav1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.InfrastructureSpec{},
					Status: ocinfrav1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
				&ocinfrav1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.APIServerSpec{},
				},
			},
			expectClientConfig: clusterv1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("configmap-fake-cert-data"),
			},
		},
		{
			name: "add service account ca to ManagedClusterInfo",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
							OCP:  clusterv1beta1.OCPDistributionInfo{},
						},
					},
				},
			},
			existingKubeOjb: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default-token-xxx",
						Namespace: "kube-system",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("sa-fake-cert-data"),
					},
					Type: corev1.SecretTypeServiceAccountToken,
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "kube-system",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "default-token-xxx",
						},
					},
				},
			},
			existingOcpOjb: []runtime.Object{
				&ocinfrav1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.InfrastructureSpec{},
					Status: ocinfrav1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
			},
			expectClientConfig: clusterv1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("sa-fake-cert-data"),
			},
		},
		{
			name: "update service account ca to ManagedClusterInfo",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
							OCP: clusterv1beta1.OCPDistributionInfo{
								ManagedClusterClientConfig: clusterv1beta1.ClientConfig{
									URL:      "http://127.0.0.1:6443",
									CABundle: []byte("ori data"),
								},
							},
						},
					},
				},
			},
			existingKubeOjb: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default-token-xxx",
						Namespace: "kube-system",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("sa-fake-cert-data"),
					},
					Type: corev1.SecretTypeServiceAccountToken,
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "kube-system",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "default-token-xxx",
						},
					},
				},
			},
			existingOcpOjb: []runtime.Object{
				&ocinfrav1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.InfrastructureSpec{},
					Status: ocinfrav1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
			},
			expectClientConfig: clusterv1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("sa-fake-cert-data"),
			},
		},
		{
			name: "do not need updateca to ManagedClusterInfo",
			existingCluster: []runtime.Object{
				&clusterv1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ManagedClusterInfo1",
						Namespace: "ManagedClusterInfo1",
					},
					Spec: clusterv1beta1.ClusterInfoSpec{},
					Status: clusterv1beta1.ClusterInfoStatus{
						DistributionInfo: clusterv1beta1.DistributionInfo{
							Type: clusterv1beta1.DistributionTypeOCP,
							OCP: clusterv1beta1.OCPDistributionInfo{
								ManagedClusterClientConfig: clusterv1beta1.ClientConfig{
									URL:      "http://127.0.0.1:6443",
									CABundle: []byte("sa-fake-cert-data"),
								},
							},
						},
					},
				},
			},
			existingKubeOjb: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default-token-xxx",
						Namespace: "kube-system",
					},
					Data: map[string][]byte{
						"ca.crt": []byte("sa-fake-cert-data"),
					},
					Type: corev1.SecretTypeServiceAccountToken,
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "default",
						Namespace: "kube-system",
					},
					Secrets: []corev1.ObjectReference{
						{
							Name: "default-token-xxx",
						},
					},
				},
			},
			existingOcpOjb: []runtime.Object{
				&ocinfrav1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: ocinfrav1.InfrastructureSpec{},
					Status: ocinfrav1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
			},
			expectClientConfig: clusterv1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("sa-fake-cert-data"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			svrc := newTestClusterInfoReconciler(test.existingCluster, test.existingKubeOjb, test.existingOcpOjb)
			retConfig := svrc.getClientConfig(ctx)
			if !reflect.DeepEqual(retConfig, test.expectClientConfig) {
				t.Errorf("case:%v, expect client config is not same as return client config. expecrClientConfigs=%v, return client config=%v", test.name, test.expectClientConfig, retConfig)
			}
			return
		})
	}
}
