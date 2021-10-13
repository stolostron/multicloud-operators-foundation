package clusterclaim

import (
	"testing"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	apiconfigv1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

func newAWSInfrastructure() *apiconfigv1.Infrastructure {
	return &apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{
			CloudConfig: apiconfigv1.ConfigMapFileReference{},
			PlatformSpec: apiconfigv1.PlatformSpec{
				Type: "AWS",
				AWS:  &apiconfigv1.AWSPlatformSpec{},
			},
		},
		Status: apiconfigv1.InfrastructureStatus{
			InfrastructureName: "ocp-aws",
			Platform:           "AWS",
			PlatformStatus: &apiconfigv1.PlatformStatus{
				Type: "AWS",
				AWS: &apiconfigv1.AWSPlatformStatus{
					Region: "region-aws",
				},
			},
			EtcdDiscoveryDomain:  "osd-test.wu67.s1.devshift.org",
			APIServerURL:         "https://api.osd-test.wu67.s1.devshift.org:6443",
			APIServerInternalURL: "https://api-int.osd-test.wu67.s1.devshift.org:6443",
			ControlPlaneTopology: apiconfigv1.HighlyAvailableTopologyMode,
		},
	}
}

func newGCPInfrastructure() *apiconfigv1.Infrastructure {
	return &apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{
			CloudConfig: apiconfigv1.ConfigMapFileReference{},
			PlatformSpec: apiconfigv1.PlatformSpec{
				Type: "GCP",
				GCP:  &apiconfigv1.GCPPlatformSpec{},
			},
		},
		Status: apiconfigv1.InfrastructureStatus{
			InfrastructureName: "ocp-gcp",
			Platform:           "GCP",
			PlatformStatus: &apiconfigv1.PlatformStatus{
				Type: "GCP",
				GCP: &apiconfigv1.GCPPlatformStatus{
					Region: "region-gcp",
				},
			},
			EtcdDiscoveryDomain:  "osd-test.wu67.s1.devshift.org",
			APIServerURL:         "https://api.osd-test.wu67.s1.devshift.org:6443",
			APIServerInternalURL: "https://api-int.osd-test.wu67.s1.devshift.org:6443",
			ControlPlaneTopology: apiconfigv1.SingleReplicaTopologyMode,
		},
	}
}

func newConfigV1Client(version string, platformType string) openshiftclientset.Interface {
	clusterVersion := &apiconfigv1.ClusterVersion{}
	if version == "4.x" {
		clusterVersion = newClusterVersion()
	} else {
		return configfake.NewSimpleClientset(clusterVersion)
	}

	infrastructure := &apiconfigv1.Infrastructure{}
	switch platformType {
	case PlatformAWS:
		infrastructure = newAWSInfrastructure()
	case PlatformGCP:
		infrastructure = newGCPInfrastructure()
	}

	return configfake.NewSimpleClientset(clusterVersion, infrastructure)
}

func projectAPIResource() *metav1.APIResourceList {
	project := metav1.APIResource{
		Name:         "projects",
		SingularName: "project",
		Namespaced:   false,
		Kind:         "Project",
	}
	return &metav1.APIResourceList{
		GroupVersion: "project.openshift.io/v1",
		APIResources: []metav1.APIResource{project},
	}
}

func managedAPIResource() *metav1.APIResourceList {
	managed := metav1.APIResource{
		Name:         "subjectpermissions",
		SingularName: "subjectpermission",
		Namespaced:   true,
		Kind:         "SubjectPermission",
	}
	return &metav1.APIResourceList{
		GroupVersion: "managed.openshift.io/v1alpha1",
		APIResources: []metav1.APIResource{managed},
	}
}

func newFakeKubeClient(resources []*metav1.APIResourceList, objects []runtime.Object) kubernetes.Interface {
	fakeKubeClient := kubefake.NewSimpleClientset(objects...)
	fakeKubeClient.Resources = append(fakeKubeClient.Resources, resources...)
	fakeKubeClient.Discovery().ServerVersion()
	return fakeKubeClient
}

func newConfigmapConsoleConfig() *corev1.ConfigMap {
	return &corev1.ConfigMap{
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
}

func newEndpointKubernetes() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "10.0.139.149",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     "https",
						Port:     6443,
						Protocol: "TCP",
					},
				},
			},
		},
	}
}

func newNode(platform string) *corev1.Node {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
	}

	switch platform {
	case PlatformAWS:
		node.Spec.ProviderID = "aws:///us-east-1a/i-07bacee3f60562daa"
	case PlatformAzure:
		node.Spec.ProviderID = "azure:///subscriptions/03e5f0ef-0741-442a-bc1b-ba34ceb3f63f/resourceGroups/yzwaz-rlpjx-rg/providers/Microsoft.Compute/virtualMachines/yzwaz-rlpjx-master-0"
	case PlatformGCP:
		node.Spec.ProviderID = "gce:///abc"
	case PlatformIBM:
		node.Spec.ProviderID = "ibm:///abc"
	case PlatformVSphere:
		node.Spec.ProviderID = "vsphere://421a27ac-bb12-f6e6-48cb-f2aa74e56156"
	case PlatformOpenStack:
		node.Spec.ProviderID = "openstack:///dda1f31a-3dfb-435a-9e1d-16149a8dd628"
	case PlatformIBMZ:
		node.Status.NodeInfo.Architecture = "s390x"
	case PlatformIBMP:
		node.Status.NodeInfo.Architecture = "ppc64le"
	}
	return node
}

func newConfigmap(namespace, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func fakeHubClient(clusterName string, labels map[string]string) client.Client {
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: clusterName,
			Labels: map[string]string{
				clusterv1beta1.LabelClusterID:   "1234",
				clusterv1beta1.LabelCloudVendor: "AWS",
				clusterv1beta1.LabelKubeVendor:  "OCP",
			},
		},
	}
	clusterInfo.SetLabels(labels)
	s := scheme.Scheme
	s.AddKnownTypes(clusterv1beta1.GroupVersion, &clusterv1beta1.ManagedClusterInfo{})
	clusterv1beta1.AddToScheme(s)

	return fake.NewFakeClientWithScheme(s, clusterInfo)
}

func TestClusterClaimerList(t *testing.T) {
	tests := []struct {
		name           string
		clusterName    string
		hubClient      client.Client
		kubeClient     kubernetes.Interface
		configV1Client openshiftclientset.Interface
		expectClaims   map[string]string
		expectErr      error
	}{
		{
			name:        "claims of OCP on AWS",
			clusterName: "clusterAWS",
			hubClient:   fakeHubClient("clusterAWS", map[string]string{"testLabel/abc": "testLabel"}),
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()},
				[]runtime.Object{newNode(PlatformAWS), newConfigmapConsoleConfig()}),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			expectClaims: map[string]string{
				ClaimK8sID:                   "clusterAWS",
				ClaimOpenshiftVersion:        "4.5.11",
				ClaimOpenshiftID:             "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure: "{\"infraName\":\"ocp-aws\"}",
				ClaimOCMPlatform:             PlatformAWS,
				ClaimOCMProduct:              ProductOpenShift,
				ClaimOCMConsoleURL:           "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:          "v0.0.0-master+$Format:%H$",
				ClaimOCMRegion:               "region-aws",
				ClaimControlPlaneTopology:    "HighlyAvailable",
				"abc.testLabel":              "testLabel",
			},
			expectErr: nil,
		},
		{
			name:        "claims of OSD on GCP",
			clusterName: "clusterOSDGCP",
			hubClient:   fakeHubClient("clusterOSDGCP", map[string]string{"open-cluster-management.io/agent": "abc"}),
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource(), managedAPIResource()},
				[]runtime.Object{newNode(PlatformGCP), newConfigmapConsoleConfig()}),
			configV1Client: newConfigV1Client("4.x", PlatformGCP),
			expectClaims: map[string]string{
				ClaimK8sID:                   "clusterOSDGCP",
				ClaimOpenshiftVersion:        "4.5.11",
				ClaimOpenshiftID:             "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure: "{\"infraName\":\"ocp-gcp\"}",
				ClaimOCMPlatform:             PlatformGCP,
				ClaimOCMProduct:              ProductOSD,
				ClaimOCMConsoleURL:           "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:          "v0.0.0-master+$Format:%H$",
				ClaimOCMRegion:               "region-gcp",
				ClaimControlPlaneTopology:    "SingleReplica",
			},
			expectErr: nil,
		},
		{
			name:           "claims of non-OCP",
			clusterName:    "clusterNonOCP",
			hubClient:      fakeHubClient("clusterNonOCP", map[string]string{}),
			kubeClient:     newFakeKubeClient(nil, []runtime.Object{newNode(PlatformGCP), newConfigmapConsoleConfig()}),
			configV1Client: newConfigV1Client("3", ""),
			expectClaims: map[string]string{
				ClaimK8sID:          "clusterNonOCP",
				ClaimOCMPlatform:    PlatformGCP,
				ClaimOCMProduct:     ProductOther,
				ClaimOCMKubeVersion: "v0.0.0-master+$Format:%H$",
			},
			expectErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{ClusterName: test.clusterName,
				HubClient:      test.hubClient,
				KubeClient:     test.kubeClient,
				ConfigV1Client: test.configV1Client}
			claims, err := clusterClaimer.List()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, len(claims), len(test.expectClaims))
			for _, claim := range claims {
				assert.Equal(t, test.expectClaims[claim.Name], claim.Spec.Value)
			}
		})
	}
}

func TestIsOpenShift(t *testing.T) {
	tests := []struct {
		name       string
		kubeClient kubernetes.Interface
		expectRet  bool
	}{
		{
			name:       "is openshift",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			expectRet:  true,
		},
		{
			name:       "is not openshift",
			kubeClient: newFakeKubeClient(nil, nil),
			expectRet:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient}
			rst := clusterClaimer.isOpenShift()
			assert.Equal(t, test.expectRet, rst)
		})
	}
}

func TestOpenshiftDedicated(t *testing.T) {
	tests := []struct {
		name       string
		kubeClient kubernetes.Interface
		expectRet  bool
	}{
		{
			name:       "is osd",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource(), managedAPIResource()}, nil),
			expectRet:  true,
		},
		{
			name:       "is openshift not osd",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			expectRet:  false,
		},
		{
			name:       "is not openshift",
			kubeClient: newFakeKubeClient(nil, nil),
			expectRet:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient}
			rst := clusterClaimer.isOpenshiftDedicated()
			assert.Equal(t, test.expectRet, rst)
		})
	}
}

func TestROSA(t *testing.T) {
	tests := []struct {
		name       string
		kubeClient kubernetes.Interface
		expectRet  bool
	}{
		{
			name: "is ROSA",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource(), managedAPIResource()},
				[]runtime.Object{newConfigmap("openshift-config", "rosa-brand-logo")}),
			expectRet: true,
		},
		{
			name:       "is openshift not ROSA",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			expectRet:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient}
			rst := clusterClaimer.isROSA()
			assert.Equal(t, test.expectRet, rst)
		})
	}
}

func TestGetOCPVersion(t *testing.T) {
	tests := []struct {
		name            string
		kubeClient      kubernetes.Interface
		configV1Client  openshiftclientset.Interface
		expectVersion   string
		expectClusterID string
		expectErr       error
	}{
		{
			name:            "is OCP 3.x",
			kubeClient:      newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			configV1Client:  newConfigV1Client("3", ""),
			expectVersion:   "3",
			expectClusterID: "",
			expectErr:       nil,
		},
		{
			name:            "is OCP 4.x",
			kubeClient:      newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			configV1Client:  newConfigV1Client("4.x", ""),
			expectVersion:   "4.5.11",
			expectClusterID: "ffd989a0-8391-426d-98ac-86ae6d051433",
			expectErr:       nil,
		},
		{
			name:            "is not OCP",
			kubeClient:      newFakeKubeClient(nil, nil),
			expectVersion:   "",
			expectClusterID: "",
			expectErr:       nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, ConfigV1Client: test.configV1Client}
			version, clusterID, err := clusterClaimer.getOCPVersion()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectVersion, version)
			assert.Equal(t, test.expectClusterID, clusterID)
		})
	}
}

func TestGetInfraConfig(t *testing.T) {
	tests := []struct {
		name              string
		kubeClient        kubernetes.Interface
		configV1Client    openshiftclientset.Interface
		expectInfraConfig string
		expectErr         error
	}{
		{
			name:              "OCP on AWS",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			configV1Client:    newConfigV1Client("4.x", PlatformAWS),
			expectInfraConfig: "{\"infraName\":\"ocp-aws\"}",
			expectErr:         nil,
		},
		{
			name:              "OCP on GCP",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			configV1Client:    newConfigV1Client("4.x", PlatformGCP),
			expectInfraConfig: "{\"infraName\":\"ocp-gcp\"}",
			expectErr:         nil,
		},
		{
			name:              "is not OCP",
			kubeClient:        newFakeKubeClient(nil, nil),
			expectInfraConfig: "",
			expectErr:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, ConfigV1Client: test.configV1Client}
			infraConfig, err := clusterClaimer.getInfraConfig()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectInfraConfig, infraConfig)
		})
	}
}

func TestGetClusterRegion(t *testing.T) {
	tests := []struct {
		name           string
		kubeClient     kubernetes.Interface
		configV1Client openshiftclientset.Interface
		expectRegion   string
		expectErr      error
	}{
		{
			name:           "OCP on AWS",
			kubeClient:     newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			expectRegion:   "region-aws",
			expectErr:      nil,
		},
		{
			name:           "OCP on GCP",
			kubeClient:     newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			configV1Client: newConfigV1Client("4.x", PlatformGCP),
			expectRegion:   "region-gcp",
			expectErr:      nil,
		},
		{
			name:         "is not OCP",
			kubeClient:   newFakeKubeClient(nil, nil),
			expectRegion: "",
			expectErr:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, ConfigV1Client: test.configV1Client}
			region, err := clusterClaimer.getClusterRegion()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectRegion, region)
		})
	}
}

func TestGetMasterAddresses(t *testing.T) {
	tests := []struct {
		name                  string
		kubeClient            kubernetes.Interface
		dynamicClient         dynamic.Interface
		expectMasterAddresses []corev1.EndpointAddress
		expectMasterPorts     []corev1.EndpointPort
		expectClusterURL      string
		expectErr             error
	}{
		{
			name:                  "is OCP",
			kubeClient:            newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newConfigmapConsoleConfig()}),
			expectMasterAddresses: []corev1.EndpointAddress{{IP: "api.daliu-clu428.dev04.red-chesterfield.com"}},
			expectMasterPorts:     []corev1.EndpointPort{{Port: 6443}},
			expectClusterURL:      "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
			expectErr:             nil,
		},
		{
			name:                  "is not OCP",
			kubeClient:            newFakeKubeClient(nil, []runtime.Object{newEndpointKubernetes()}),
			expectMasterAddresses: []corev1.EndpointAddress{{IP: "10.0.139.149"}},
			expectMasterPorts:     []corev1.EndpointPort{{Name: "https", Port: 6443, Protocol: "TCP"}},
			expectClusterURL:      "",
			expectErr:             nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient}
			masterAddresses, masterPorts, clusterURL, err := clusterClaimer.getMasterAddresses()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectMasterAddresses, masterAddresses)
			assert.Equal(t, test.expectMasterPorts, masterPorts)
			assert.Equal(t, test.expectClusterURL, clusterURL)
		})
	}
}

func TestGetKubeVersionPlatformProduct(t *testing.T) {
	tests := []struct {
		name              string
		kubeClient        kubernetes.Interface
		dynamicClient     dynamic.Interface
		expectKubeVersion string
		expectPlatform    string
		expectProduct     string
		expectErr         error
	}{
		{
			name:              "is OCP on AWS",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformAWS)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformAWS,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on Azure",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformAzure)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformAzure,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on IBM",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformIBM)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformIBM,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on IBMZ",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformIBMZ)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformIBMZ,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on IBMP",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformIBMP)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformIBMP,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on GCP",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformGCP)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformGCP,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on Openstack",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformOpenStack)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformOpenStack,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on VSphere",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformVSphere)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformVSphere,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on VSphere",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformVSphere)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformVSphere,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is AKS",
			kubeClient:        newFakeKubeClient(nil, []runtime.Object{newNode(PlatformAzure)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformAzure,
			expectProduct:     ProductAKS,
			expectErr:         nil,
		},
		{
			name:              "is OSD on AWS",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource(), managedAPIResource()}, []runtime.Object{newNode(PlatformAWS)}),
			expectKubeVersion: "v0.0.0-master+$Format:%H$",
			expectPlatform:    PlatformAWS,
			expectProduct:     ProductOSD,
			expectErr:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient}
			kubeVersion, platform, product, err := clusterClaimer.getKubeVersionPlatformProduct()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectKubeVersion, kubeVersion)
			assert.Equal(t, test.expectPlatform, platform)
			assert.Equal(t, test.expectProduct, product)
		})
	}
}
