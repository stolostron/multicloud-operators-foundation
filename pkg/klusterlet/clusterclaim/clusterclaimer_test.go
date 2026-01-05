package clusterclaim

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"

	apiconfigv1 "github.com/openshift/api/config/v1"
	apioauthv1 "github.com/openshift/api/oauth/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	openshiftoauthclientset "github.com/openshift/client-go/oauth/clientset/versioned"
	oauthfake "github.com/openshift/client-go/oauth/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"
)

func newClusterVersion() *apiconfigv1.ClusterVersion {
	now := metav1.Now()
	oneDay := metav1.NewTime(now.AddDate(0, 0, 1))
	oneMonth := metav1.NewTime(now.AddDate(0, 1, 0))
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
					Image:          "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
					State:          "Completed",
					Verified:       false,
					Version:        "4.6.8",
					CompletionTime: &oneMonth,
				},
				{
					Image:          "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
					State:          "Completed",
					Verified:       false,
					Version:        "4.5.11",
					CompletionTime: &oneDay,
				},
				{
					Image:          "quay.io/openshift-release-dev/ocp-release@sha256:4d048ae1274d11c49f9b7e70713a072315431598b2ddbb512aee4027c422fe3e",
					State:          "Completed",
					Verified:       false,
					Version:        "4.4.11",
					CompletionTime: &now,
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
func newNamespace(name, uid string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(uid),
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
func newOtherInfrastructure() *apiconfigv1.Infrastructure {
	return &apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{
			CloudConfig: apiconfigv1.ConfigMapFileReference{},
			PlatformSpec: apiconfigv1.PlatformSpec{
				Type: "Other",
				GCP:  &apiconfigv1.GCPPlatformSpec{},
			},
		},
		Status: apiconfigv1.InfrastructureStatus{
			InfrastructureName: "Other",
			Platform:           "Other",
			PlatformStatus: &apiconfigv1.PlatformStatus{
				Type: "Other",
			},
		},
	}
}

func newROKSInfrastructure() *apiconfigv1.Infrastructure {
	return &apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{
			CloudConfig: apiconfigv1.ConfigMapFileReference{},
			PlatformSpec: apiconfigv1.PlatformSpec{
				Type: "IBMCloud",
			},
		},
		Status: apiconfigv1.InfrastructureStatus{
			InfrastructureName: "kubernetes",
			Platform:           "IBMCloud",
			PlatformStatus: &apiconfigv1.PlatformStatus{
				Type: "IBMCloud",
			},
			EtcdDiscoveryDomain:  "osd-test.wu67.s1.devshift.org",
			APIServerURL:         "https://api.osd-test.wu67.s1.devshift.org:6443",
			APIServerInternalURL: "https://api-int.osd-test.wu67.s1.devshift.org:6443",
			ControlPlaneTopology: apiconfigv1.HighlyAvailableTopologyMode,
		},
	}
}

func newNutanixInfrastructure() *apiconfigv1.Infrastructure {
	return &apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{
			CloudConfig: apiconfigv1.ConfigMapFileReference{},
			PlatformSpec: apiconfigv1.PlatformSpec{
				Type: "Nutanix",
			},
		},
		Status: apiconfigv1.InfrastructureStatus{
			InfrastructureName: "nutanix-cluster",
			Platform:           "Nutanix",
			PlatformStatus: &apiconfigv1.PlatformStatus{
				Type: "Nutanix",
				Nutanix: &apiconfigv1.NutanixPlatformStatus{
					APIServerInternalIPs: []string{"10.0.0.1"},
					IngressIPs:           []string{"10.0.0.2"},
				},
			},
			EtcdDiscoveryDomain:  "osd-test.wu67.s1.devshift.org",
			APIServerURL:         "https://api.osd-test.wu67.s1.devshift.org:6443",
			APIServerInternalURL: "https://api-int.osd-test.wu67.s1.devshift.org:6443",
			ControlPlaneTopology: apiconfigv1.HighlyAvailableTopologyMode,
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

	var infrastructure *apiconfigv1.Infrastructure
	switch platformType {
	case PlatformAWS:
		infrastructure = newAWSInfrastructure()
	case PlatformGCP:
		infrastructure = newGCPInfrastructure()
	case PlatformIBM:
		infrastructure = newROKSInfrastructure()
	case PlatformNutanix:
		infrastructure = newNutanixInfrastructure()
	default:
		infrastructure = newOtherInfrastructure()
	}

	return configfake.NewSimpleClientset(clusterVersion, infrastructure)
}

func newChallengingOauthclient(redirectURIs []string) *apioauthv1.OAuthClient {
	return &apioauthv1.OAuthClient{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-challenging-client",
		},
		RedirectURIs:          redirectURIs,
		RespondWithChallenges: true,
	}
}

func newOauthV1Client(redirectURIs []string) openshiftoauthclientset.Interface {
	if redirectURIs != nil {
		return oauthfake.NewSimpleClientset(newChallengingOauthclient(redirectURIs))
	}
	return oauthfake.NewSimpleClientset()
}

var projectAPIGroupResources = &restmapper.APIGroupResources{
	Group: metav1.APIGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "project.openshift.io/v1",
		},
		Name: "project.openshift.io",
		Versions: []metav1.GroupVersionForDiscovery{
			{
				GroupVersion: "v1",
				Version:      "v1",
			},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{
			GroupVersion: "v1",
			Version:      "v1",
		},
		ServerAddressByClientCIDRs: nil,
	},
	VersionedResources: map[string][]metav1.APIResource{
		"v1": {
			{
				Name:         "projects",
				SingularName: "project",
				Group:        "project.openshift.io",
				Kind:         "Project",
				Version:      "v1",
			},
		},
	},
}

var aroAPIGroupResources = &restmapper.APIGroupResources{
	Group: metav1.APIGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "aro.openshift.io/v1alpha1",
		},
		Name: "aro.openshift.io",
		Versions: []metav1.GroupVersionForDiscovery{
			{
				GroupVersion: "v1alpha1",
				Version:      "v1alpha1",
			},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{
			GroupVersion: "v1alpha1",
			Version:      "v1alpha1",
		},
		ServerAddressByClientCIDRs: nil,
	},
	VersionedResources: map[string][]metav1.APIResource{
		"v1alpha1": {
			{
				Name:         "clusters",
				SingularName: "cluster",
				Group:        "aro.openshift.io",
				Kind:         "Cluster",
				Version:      "v1alpha1",
			},
		},
	},
}

var osdAPIGroupResources = &restmapper.APIGroupResources{
	Group: metav1.APIGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SubjectPermission",
			APIVersion: "managed.openshift.io/v1alpha1",
		},
		Name: "managed.openshift.io",
		Versions: []metav1.GroupVersionForDiscovery{
			{
				GroupVersion: "v1alpha1",
				Version:      "v1alpha1",
			},
		},
		PreferredVersion: metav1.GroupVersionForDiscovery{
			GroupVersion: "v1alpha1",
			Version:      "v1alpha1",
		},
		ServerAddressByClientCIDRs: nil,
	},
	VersionedResources: map[string][]metav1.APIResource{
		"v1alpha1": {
			{
				Name:         "subjectpermissions",
				SingularName: "subjectpermission",
				Group:        "managed.openshift.io",
				Kind:         "SubjectPermission",
				Version:      "v1alpha1",
			},
		},
	},
}

func newFakeRestMapper(resources []*restmapper.APIGroupResources) meta.RESTMapper {
	return restmapper.NewDiscoveryRESTMapper(resources)
}

func newFakeKubeClient(objects []runtime.Object) kubernetes.Interface {
	fakeKubeClient := kubefake.NewSimpleClientset(objects...)
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

func newEndpoint() *corev1.Endpoints {
	return &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
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

func newMicroShiftVersionConfigmap(version string) *corev1.ConfigMap {
	configmap := newConfigmap("kube-public", "microshift-version")
	configmap.Data = map[string]string{
		"version": version,
	}
	return configmap
}

func TestClusterClaimerList(t *testing.T) {
	tests := []struct {
		name           string
		clusterName    string
		kubeClient     kubernetes.Interface
		configV1Client openshiftclientset.Interface
		oauthV1Client  openshiftoauthclientset.Interface
		mapper         meta.RESTMapper
		expectClaims   map[string]string
		expectErr      error
	}{
		{
			name:           "claims of OCP on AWS",
			clusterName:    "clusterAWS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAWS), newConfigmapConsoleConfig()}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			oauthV1Client: newOauthV1Client([]string{
				"https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			}),
			expectClaims: map[string]string{
				ClaimK8sID:                      "clusterAWS",
				ClaimOpenshiftAPIServerURL:      "https://api.osd-test.wu67.s1.devshift.org:6443",
				ClaimOpenshiftVersion:           "4.6.8",
				ClaimOpenshiftID:                "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure:    "{\"infraName\":\"ocp-aws\"}",
				ClaimOCMPlatform:                PlatformAWS,
				ClaimOCMProduct:                 ProductOpenShift,
				ClaimOCMConsoleURL:              "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:             "v0.0.0-master+$Format:%H$",
				ClaimOCMRegion:                  "region-aws",
				ClaimControlPlaneTopology:       "HighlyAvailable",
				ClaimOpenshiftOauthRedirectURIs: "https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			},
			expectErr: nil,
		},
		{
			name:           "claims of OCP on AWS disable enableSyncLabelsToClusterClaims",
			clusterName:    "clusterAWS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAWS), newConfigmapConsoleConfig()}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			oauthV1Client: newOauthV1Client([]string{
				"https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			}),
			expectClaims: map[string]string{
				ClaimK8sID:                      "clusterAWS",
				ClaimOpenshiftVersion:           "4.6.8",
				ClaimOpenshiftID:                "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftAPIServerURL:      "https://api.osd-test.wu67.s1.devshift.org:6443",
				ClaimOpenshiftInfrastructure:    "{\"infraName\":\"ocp-aws\"}",
				ClaimOCMPlatform:                PlatformAWS,
				ClaimOCMProduct:                 ProductOpenShift,
				ClaimOCMConsoleURL:              "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:             "v0.0.0-master+$Format:%H$",
				ClaimOCMRegion:                  "region-aws",
				ClaimControlPlaneTopology:       "HighlyAvailable",
				ClaimOpenshiftOauthRedirectURIs: "https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			},
			expectErr: nil,
		},
		{
			name:           "claims of OSD on GCP",
			clusterName:    "clusterOSDGCP",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformGCP), newConfigmapConsoleConfig()}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources, osdAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformGCP),
			oauthV1Client:  newOauthV1Client(nil),

			expectClaims: map[string]string{
				ClaimK8sID:                   "clusterOSDGCP",
				ClaimOpenshiftVersion:        "4.6.8",
				ClaimOpenshiftID:             "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure: "{\"infraName\":\"ocp-gcp\"}",
				ClaimOpenshiftAPIServerURL:   "https://api.osd-test.wu67.s1.devshift.org:6443",
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
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformGCP), newEndpoint()}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{}),
			configV1Client: newConfigV1Client("3", ""),
			oauthV1Client:  newOauthV1Client(nil),
			expectClaims: map[string]string{
				ClaimK8sID:          "clusterNonOCP",
				ClaimOCMPlatform:    PlatformGCP,
				ClaimOCMProduct:     ProductOther,
				ClaimOCMKubeVersion: "v0.0.0-master+$Format:%H$",
			},
			expectErr: nil,
		},
		{
			name:           "claims of IBM Cloud (ROKS)",
			clusterName:    "clusterROKS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBM), newConfigmapConsoleConfig()}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformIBM),
			oauthV1Client: newOauthV1Client([]string{
				"https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			}),
			expectClaims: map[string]string{
				ClaimK8sID:                      "clusterROKS",
				ClaimOpenshiftVersion:           "4.6.8",
				ClaimOpenshiftID:                "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure:    "{\"infraName\":\"kubernetes\"}",
				ClaimOpenshiftAPIServerURL:      "https://api.osd-test.wu67.s1.devshift.org:6443",
				ClaimOCMPlatform:                PlatformIBM,
				ClaimOCMProduct:                 ProductROKS,
				ClaimOCMConsoleURL:              "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:             "v0.0.0-master+$Format:%H$",
				ClaimControlPlaneTopology:       "HighlyAvailable",
				ClaimOpenshiftOauthRedirectURIs: "https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			},
			expectErr: nil,
		},
		{
			name:        "claims of microshift",
			clusterName: "microshift",
			kubeClient:  newFakeKubeClient([]runtime.Object{newEndpointKubernetes(), newNode(""), newMicroShiftVersionConfigmap("4.13.6")}),
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectClaims: map[string]string{
				ClaimK8sID:             "microshift",
				ClaimOCMPlatform:       PlatformOther,
				ClaimOCMProduct:        ProductMicroShift,
				ClaimMicroShiftVersion: "4.13.6",
				ClaimOCMKubeVersion:    "v0.0.0-master+$Format:%H$",
			},
			expectErr: nil,
		},
		{
			name:           "claims of OCP on Nutanix",
			clusterName:    "clusterNutanix",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformNutanix), newConfigmapConsoleConfig()}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformNutanix),
			oauthV1Client: newOauthV1Client([]string{
				"https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			}),
			expectClaims: map[string]string{
				ClaimK8sID:                      "clusterNutanix",
				ClaimOpenshiftAPIServerURL:      "https://api.osd-test.wu67.s1.devshift.org:6443",
				ClaimOpenshiftVersion:           "4.6.8",
				ClaimOpenshiftID:                "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure:    "{\"infraName\":\"nutanix-cluster\"}",
				ClaimOCMPlatform:                PlatformNutanix,
				ClaimOCMProduct:                 ProductOpenShift,
				ClaimOCMConsoleURL:              "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:             "v0.0.0-master+$Format:%H$",
				ClaimControlPlaneTopology:       "HighlyAvailable",
				ClaimOpenshiftOauthRedirectURIs: "https://oauth-openshift.apps.a.b.c.com/oauth/token/implicit",
			},
			expectErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{
				KubeClient:       test.kubeClient,
				Mapper:           test.mapper,
				ConfigV1Client:   test.configV1Client,
				OauthV1Client:    test.oauthV1Client,
				managedclusterID: test.clusterName,
			}
			claims, err := clusterClaimer.GenerateExpectClusterClaims()
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
		name        string
		mapper      meta.RESTMapper
		expectedRet bool
		expectedErr error
	}{
		{
			name:        "is openshift",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: true,
			expectedErr: nil,
		},
		{
			name:        "is not openshift",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectedRet: false,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{Mapper: test.mapper}
			rst, err := clusterClaimer.isOpenShift()
			assert.Equal(t, test.expectedRet, rst)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestOpenshiftDedicated(t *testing.T) {
	tests := []struct {
		name        string
		mapper      meta.RESTMapper
		expectedRet bool
		expectedErr error
	}{
		{
			name:        "is osd",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources, osdAPIGroupResources}),
			expectedRet: true,
			expectedErr: nil,
		},
		{
			name:        "is openshift not osd",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: false,
			expectedErr: nil,
		},
		{
			name:        "is not openshift",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectedRet: false,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{Mapper: test.mapper}
			rst, err := clusterClaimer.isOpenshiftDedicated()
			assert.Equal(t, test.expectedRet, rst)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestROSA(t *testing.T) {
	tests := []struct {
		name        string
		kubeClient  kubernetes.Interface
		mapper      meta.RESTMapper
		expectedRet bool
		expectedErr error
	}{
		{
			name:        "is ROSA",
			kubeClient:  newFakeKubeClient([]runtime.Object{newConfigmap("openshift-config", "rosa-brand-logo")}),
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: true,
			expectedErr: nil,
		},
		{
			name:        "is openshift not ROSA",
			kubeClient:  newFakeKubeClient(nil),
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: false,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper}
			rst, err := clusterClaimer.isROSA()
			assert.Equal(t, test.expectedRet, rst)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestROSACluster(t *testing.T) {
	tests := []struct {
		name        string
		kubeClient  kubernetes.Interface
		mapper      meta.RESTMapper
		expectedRet bool
		expectedErr error
	}{
		{
			name:        "is ROSA",
			kubeClient:  newFakeKubeClient([]runtime.Object{newConfigmap("openshift-config", "rosa-brand-logo")}),
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: true,
			expectedErr: nil,
		},
		{
			name:        "is openshift not ROSA",
			kubeClient:  newFakeKubeClient(nil),
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: false,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper}
			rst, err := clusterClaimer.IsROSACluster()
			assert.Equal(t, test.expectedRet, rst)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}
func TestARO(t *testing.T) {
	tests := []struct {
		name        string
		mapper      meta.RESTMapper
		expectedRet bool
		expectedErr error
	}{
		{
			name:        "is ARO",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources, aroAPIGroupResources}),
			expectedRet: true,
			expectedErr: nil,
		},
		{
			name:        "is openshift not ARO",
			mapper:      newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectedRet: false,
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{Mapper: test.mapper}
			rst, err := clusterClaimer.isARO()
			assert.Equal(t, test.expectedRet, rst)
			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestGetOCPVersion(t *testing.T) {
	tests := []struct {
		name            string
		kubeClient      kubernetes.Interface
		configV1Client  openshiftclientset.Interface
		mapper          meta.RESTMapper
		expectVersion   string
		expectClusterID string
		expectErr       error
	}{
		{
			name:            "is OCP 3.x",
			kubeClient:      newFakeKubeClient(nil),
			mapper:          newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client:  newConfigV1Client("3", ""),
			expectVersion:   "3",
			expectClusterID: "",
			expectErr:       nil,
		},
		{
			name:            "is OCP 4.x",
			kubeClient:      newFakeKubeClient(nil),
			mapper:          newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client:  newConfigV1Client("4.x", ""),
			expectVersion:   "4.6.8",
			expectClusterID: "ffd989a0-8391-426d-98ac-86ae6d051433",
			expectErr:       nil,
		},
		{
			name:            "is not OCP",
			kubeClient:      newFakeKubeClient(nil),
			mapper:          newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectVersion:   "",
			expectClusterID: "",
			expectErr:       nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper, ConfigV1Client: test.configV1Client}
			version, clusterID, err := clusterClaimer.getOCPVersion()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectVersion, version)
			assert.Equal(t, test.expectClusterID, clusterID)
		})
	}
}

func TestManagedClusterID(t *testing.T) {
	tests := []struct {
		name            string
		kubeClient      kubernetes.Interface
		configV1Client  openshiftclientset.Interface
		mapper          meta.RESTMapper
		expectClusterID string
		expectErr       error
	}{
		{
			name:            "is OCP 3.x",
			kubeClient:      newFakeKubeClient([]runtime.Object{newNamespace("kube-system", "abc123")}),
			mapper:          newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client:  newConfigV1Client("3", ""),
			expectClusterID: "abc123",
			expectErr:       nil,
		},
		{
			name:            "is OCP 4.x",
			kubeClient:      newFakeKubeClient(nil),
			mapper:          newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client:  newConfigV1Client("4.x", ""),
			expectClusterID: "ffd989a0-8391-426d-98ac-86ae6d051433",
			expectErr:       nil,
		},
		{
			name:            "is not OCP",
			kubeClient:      newFakeKubeClient([]runtime.Object{newNamespace("kube-system", "abc123")}),
			mapper:          newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectClusterID: "abc123",
			expectErr:       nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper, ConfigV1Client: test.configV1Client}
			clusterID, err := clusterClaimer.getManagedClusterID()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectClusterID, clusterID)
		})
	}
}

func TestGetInfraConfig(t *testing.T) {
	tests := []struct {
		name              string
		kubeClient        kubernetes.Interface
		configV1Client    openshiftclientset.Interface
		mapper            meta.RESTMapper
		expectInfraConfig string
		expectServerURL   string
		expectErr         error
	}{
		{
			name:              "OCP on AWS",
			kubeClient:        newFakeKubeClient(nil),
			mapper:            newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client:    newConfigV1Client("4.x", PlatformAWS),
			expectInfraConfig: "{\"infraName\":\"ocp-aws\"}",
			expectServerURL:   "https://api.osd-test.wu67.s1.devshift.org:6443",
			expectErr:         nil,
		},
		{
			name:              "OCP on GCP",
			kubeClient:        newFakeKubeClient(nil),
			mapper:            newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client:    newConfigV1Client("4.x", PlatformGCP),
			expectInfraConfig: "{\"infraName\":\"ocp-gcp\"}",
			expectServerURL:   "https://api.osd-test.wu67.s1.devshift.org:6443",
			expectErr:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper, ConfigV1Client: test.configV1Client}
			infraConfig, apiServerURL, err := clusterClaimer.getInfraConfig()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectInfraConfig, infraConfig)
			assert.Equal(t, test.expectServerURL, apiServerURL)
		})
	}
}

func TestGetClusterRegion(t *testing.T) {
	tests := []struct {
		name           string
		kubeClient     kubernetes.Interface
		configV1Client openshiftclientset.Interface
		mapper         meta.RESTMapper
		expectRegion   string
		expectErr      error
	}{
		{
			name:           "OCP on AWS",
			kubeClient:     newFakeKubeClient(nil),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			expectRegion:   "region-aws",
			expectErr:      nil,
		},
		{
			name:           "OCP on GCP",
			kubeClient:     newFakeKubeClient(nil),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformGCP),
			expectRegion:   "region-gcp",
			expectErr:      nil,
		},
		{
			name:         "is not OCP",
			kubeClient:   newFakeKubeClient(nil),
			mapper:       newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectRegion: "",
			expectErr:    nil,
		},
		{
			name: "get region from nodes",
			kubeClient: newFakeKubeClient([]runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-1",
						Labels: map[string]string{
							corev1.LabelFailureDomainBetaRegion: "eastus",
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-2",
						Labels: map[string]string{
							corev1.LabelTopologyRegion: "eastus",
						},
					},
				},
			}),
			mapper:       newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectRegion: "eastus",
			expectErr:    nil,
		},
		{
			name: "get different regions from nodes",
			kubeClient: newFakeKubeClient([]runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-1",
						Labels: map[string]string{
							corev1.LabelFailureDomainBetaRegion: "eastus",
						},
					},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-2",
						Labels: map[string]string{
							corev1.LabelTopologyRegion: "westus",
						},
					},
				},
			}),
			mapper:       newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectRegion: "eastus,westus",
			expectErr:    nil,
		},
		{
			name: "get none regions from nodes",
			kubeClient: newFakeKubeClient([]runtime.Object{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-1",
					},
				},
			}),
			mapper:       newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectRegion: "",
			expectErr:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper, ConfigV1Client: test.configV1Client}
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
		mapper                meta.RESTMapper
		expectMasterAddresses []corev1.EndpointAddress
		expectMasterPorts     []corev1.EndpointPort
		expectClusterURL      string
		expectErr             error
	}{
		{
			name:                  "is OCP",
			kubeClient:            newFakeKubeClient([]runtime.Object{newConfigmapConsoleConfig()}),
			mapper:                newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			expectMasterAddresses: []corev1.EndpointAddress{{IP: "api.daliu-clu428.dev04.red-chesterfield.com"}},
			expectMasterPorts:     []corev1.EndpointPort{{Port: 6443}},
			expectClusterURL:      "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
			expectErr:             nil,
		},
		{
			name:                  "is not OCP",
			kubeClient:            newFakeKubeClient([]runtime.Object{newEndpointKubernetes()}),
			mapper:                newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectMasterAddresses: []corev1.EndpointAddress{{IP: "10.0.139.149"}},
			expectMasterPorts:     []corev1.EndpointPort{{Name: "https", Port: 6443, Protocol: "TCP"}},
			expectClusterURL:      "",
			expectErr:             nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper}
			masterAddresses, masterPorts, clusterURL, err := clusterClaimer.getMasterAddresses()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectMasterAddresses, masterAddresses)
			assert.Equal(t, test.expectMasterPorts, masterPorts)
			assert.Equal(t, test.expectClusterURL, clusterURL)
		})
	}
}

func TestUpdatePlatformProduct(t *testing.T) {
	tests := []struct {
		name           string
		kubeClient     kubernetes.Interface
		dynamicClient  dynamic.Interface
		mapper         meta.RESTMapper
		configV1Client openshiftclientset.Interface
		expectPlatform string
		expectProduct  string
		expectErr      error
	}{
		{
			name:           "is OCP on AWS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAWS)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			expectPlatform: PlatformAWS,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on Azure",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAzure)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformAzure,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on IBM",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBM)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformIBM,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on IBMZ",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBMZ)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformIBMZ,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on IBMP",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBMP)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformIBMP,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on GCP",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformGCP)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformGCP),
			expectPlatform: PlatformGCP,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on Openstack",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformOpenStack)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformOpenStack,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on VSphere",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformVSphere)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformVSphere,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is OCP on VSphere",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformVSphere)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", ""),
			expectPlatform: PlatformVSphere,
			expectProduct:  ProductOpenShift,
			expectErr:      nil,
		},
		{
			name:           "is AKS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAzure)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{}),
			expectPlatform: PlatformAzure,
			expectProduct:  ProductAKS,
			expectErr:      nil,
		},
		{
			name:           "is OSD on AWS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAWS)}),
			mapper:         newFakeRestMapper([]*restmapper.APIGroupResources{projectAPIGroupResources, osdAPIGroupResources}),
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			expectPlatform: PlatformAWS,
			expectProduct:  ProductOSD,
			expectErr:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, Mapper: test.mapper, ConfigV1Client: test.configV1Client}
			platform, product, err := clusterClaimer.getPlatformProduct()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectPlatform, platform)
			assert.Equal(t, test.expectProduct, product)
		})
	}
}

func TestGetProductFromGitVersion(t *testing.T) {
	tests := []struct {
		name           string
		gitVersion     string
		expectPlatform string
		expectProduct  string
		expectFound    bool
	}{
		{
			name:           "is IKS",
			gitVersion:     "v1.2.3-IKS",
			expectPlatform: PlatformIBM,
			expectProduct:  ProductIKS,
			expectFound:    true,
		},
		{
			name:           "is ICP",
			gitVersion:     "v1.2.3-ICP",
			expectPlatform: PlatformIBM,
			expectProduct:  ProductICP,
			expectFound:    true,
		},
		{
			name:           "is EKS",
			gitVersion:     "v1.2.3-EKS",
			expectPlatform: PlatformAWS,
			expectProduct:  ProductEKS,
			expectFound:    true,
		},
		{
			name:           "is GKE",
			gitVersion:     "v1.2.3-GKE",
			expectPlatform: PlatformGCP,
			expectProduct:  ProductGKE,
			expectFound:    true,
		},
		{
			name:           "is not known",
			gitVersion:     "v1.2.3",
			expectPlatform: "",
			expectProduct:  "",
			expectFound:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{}
			platform, product, found := clusterClaimer.getProductFromGitVersion(test.gitVersion)
			assert.Equal(t, test.expectPlatform, platform)
			assert.Equal(t, test.expectProduct, product)
			assert.Equal(t, test.expectFound, found)
		})
	}
}

func TestGetPlatformByInfrastructure(t *testing.T) {
	tests := []struct {
		name           string
		configV1Client openshiftclientset.Interface
		product        string
		expectPlatform string
		expectProduct  string
		expectFound    bool
		expectErr      error
	}{
		{
			name:           "is AWS",
			configV1Client: newConfigV1Client("4.x", PlatformAWS),
			product:        ProductOpenShift,
			expectPlatform: PlatformAWS,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is GCP",
			configV1Client: newConfigV1Client("4.x", PlatformGCP),
			product:        ProductOpenShift,
			expectPlatform: PlatformGCP,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is ROKS",
			configV1Client: newConfigV1Client("4.x", PlatformIBM),
			product:        ProductOpenShift,
			expectPlatform: PlatformIBM,
			expectProduct:  ProductROKS,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is Nutanix",
			configV1Client: newConfigV1Client("4.x", PlatformNutanix),
			product:        ProductOpenShift,
			expectPlatform: PlatformNutanix,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is not found",
			configV1Client: newConfigV1Client("4.x", "other"),
			product:        ProductOpenShift,
			expectPlatform: "",
			expectProduct:  "",
			expectFound:    false,
			expectErr:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{ConfigV1Client: test.configV1Client}
			platform, product, found, err := clusterClaimer.getPlatformByInfrastructure(test.product)
			assert.Equal(t, test.expectPlatform, platform)
			assert.Equal(t, test.expectProduct, product)
			assert.Equal(t, test.expectFound, found)
			assert.Equal(t, test.expectErr, err)
		})
	}
}

func TestGetPlatformByNode(t *testing.T) {
	tests := []struct {
		name           string
		kubeClient     kubernetes.Interface
		product        string
		expectPlatform string
		expectProduct  string
		expectFound    bool
		expectErr      error
	}{
		{
			name:           "is IBMZ",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBMZ)}),
			product:        ProductOpenShift,
			expectPlatform: PlatformIBMZ,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is IBMP",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBMP)}),
			product:        ProductOpenShift,
			expectPlatform: PlatformIBMP,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is IBM",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformIBM)}),
			product:        ProductOpenShift,
			expectPlatform: PlatformIBM,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is Azure AKS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAzure)}),
			product:        ProductOther,
			expectPlatform: PlatformAzure,
			expectProduct:  ProductAKS,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is Azure OCP",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAzure)}),
			product:        ProductOpenShift,
			expectPlatform: PlatformAzure,
			expectProduct:  ProductOpenShift,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is AWS",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformAWS)}),
			product:        ProductOther,
			expectPlatform: PlatformAWS,
			expectProduct:  ProductOther,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is GCP",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformGCP)}),
			product:        ProductOther,
			expectPlatform: PlatformGCP,
			expectProduct:  ProductOther,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is VSphere",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformVSphere)}),
			product:        ProductOther,
			expectPlatform: PlatformVSphere,
			expectProduct:  ProductOther,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is OpenStack",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode(PlatformOpenStack)}),
			product:        ProductOther,
			expectPlatform: PlatformOpenStack,
			expectProduct:  ProductOther,
			expectFound:    true,
			expectErr:      nil,
		},
		{
			name:           "is not found",
			kubeClient:     newFakeKubeClient([]runtime.Object{newNode("other")}),
			product:        ProductOther,
			expectPlatform: "",
			expectProduct:  "",
			expectFound:    false,
			expectErr:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient}
			platform, product, found, err := clusterClaimer.getPlatformByNode(test.product)
			assert.Equal(t, test.expectPlatform, platform)
			assert.Equal(t, test.expectProduct, product)
			assert.Equal(t, test.expectFound, found)
			assert.Equal(t, test.expectErr, err)
		})
	}
}
