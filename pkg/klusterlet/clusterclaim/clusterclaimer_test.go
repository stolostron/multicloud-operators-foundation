package clusterclaim

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

const (
	clusterVersions = `{
  "apiVersion": "config.openshift.io/v1",
  "kind": "ClusterVersion",
  "metadata": {
	"name": "version"
  },
  "spec": {
    "channel": "stable-4.5",
    "clusterID": "ffd989a0-8391-426d-98ac-86ae6d051433"
  },
 "status": {
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

func newClusterVersions(version string) *unstructured.Unstructured {
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

func newInfraConfig(platformType string) *unstructured.Unstructured {
	obj := unstructured.Unstructured{}
	switch platformType {
	case PlatformAWS:
		obj.UnmarshalJSON([]byte(awsOCPInfraConfig))
	case PlatformGCP:
		obj.UnmarshalJSON([]byte(gcpInfraConfig))
	}

	return &obj
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

func TestClusterClaimerList(t *testing.T) {
	tests := []struct {
		name          string
		clusterName   string
		kubeClient    kubernetes.Interface
		dynamicClient dynamic.Interface
		expectClaims  map[string]string
		expectErr     error
	}{
		{
			name:        "claims of OCP on AWS",
			clusterName: "clusterAWS",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()},
				[]runtime.Object{newNode(PlatformAWS), newConfigmapConsoleConfig()}),
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x"), newInfraConfig(PlatformAWS)),
			expectClaims: map[string]string{
				ClaimK8sID:                   "clusterAWS",
				ClaimOpenshiftVersion:        "4.5.11",
				ClaimOpenshiftID:             "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure: "{\"infraName\":\"ocp-aws\"}",
				ClaimOCMPlatform:             PlatformAWS,
				ClaimOCMProduct:              ProductOpenShift,
				ClaimOCMConsoleURL:           "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:          "v0.0.0-master+$Format:%h$",
				ClaimOCMRegion:               "region-aws",
			},
			expectErr: nil,
		},
		{
			name:        "claims of OSD on GCP",
			clusterName: "clusterOSDGCP",
			kubeClient: newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource(), managedAPIResource()},
				[]runtime.Object{newNode(PlatformGCP), newConfigmapConsoleConfig()}),
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x"), newInfraConfig(PlatformGCP)),
			expectClaims: map[string]string{
				ClaimK8sID:                   "clusterOSDGCP",
				ClaimOpenshiftVersion:        "4.5.11",
				ClaimOpenshiftID:             "ffd989a0-8391-426d-98ac-86ae6d051433",
				ClaimOpenshiftInfrastructure: "{\"infraName\":\"ocp-gcp\"}",
				ClaimOCMPlatform:             PlatformGCP,
				ClaimOCMProduct:              ProductOSD,
				ClaimOCMConsoleURL:           "https://console-openshift-console.apps.daliu-clu428.dev04.red-chesterfield.com",
				ClaimOCMKubeVersion:          "v0.0.0-master+$Format:%h$",
				ClaimOCMRegion:               "region-gcp",
			},
			expectErr: nil,
		},
		{
			name:        "claims of non-OCP",
			clusterName: "clusterNonOCP",
			kubeClient:  newFakeKubeClient(nil, []runtime.Object{newNode(PlatformGCP), newConfigmapConsoleConfig()}),
			expectClaims: map[string]string{
				ClaimK8sID:          "clusterNonOCP",
				ClaimOCMPlatform:    PlatformGCP,
				ClaimOCMProduct:     ProductOther,
				ClaimOCMKubeVersion: "v0.0.0-master+$Format:%h$",
			},
			expectErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterClaimer := ClusterClaimer{ClusterName: test.clusterName, KubeClient: test.kubeClient, DynamicClient: test.dynamicClient}
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

func TestGetOCPVersion(t *testing.T) {
	tests := []struct {
		name            string
		kubeClient      kubernetes.Interface
		dynamicClient   dynamic.Interface
		expectVersion   string
		expectClusterID string
		expectErr       error
	}{
		{
			name:            "is OCP 3.x",
			kubeClient:      newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			dynamicClient:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("3")),
			expectVersion:   "3",
			expectClusterID: "",
			expectErr:       nil,
		},
		{
			name:            "is OCP 4.x",
			kubeClient:      newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			dynamicClient:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newClusterVersions("4.x")),
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
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, DynamicClient: test.dynamicClient}
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
		dynamicClient     dynamic.Interface
		expectInfraConfig string
		expectErr         error
	}{
		{
			name:              "OCP on AWS",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			dynamicClient:     dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(PlatformAWS)),
			expectInfraConfig: "{\"infraName\":\"ocp-aws\"}",
			expectErr:         nil,
		},
		{
			name:              "OCP on GCP",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			dynamicClient:     dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(PlatformGCP)),
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
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, DynamicClient: test.dynamicClient}
			infraConfig, err := clusterClaimer.getInfraConfig()
			assert.Equal(t, test.expectErr, err)
			assert.Equal(t, test.expectInfraConfig, infraConfig)
		})
	}
}

func TestGetClusterRegion(t *testing.T) {
	tests := []struct {
		name          string
		kubeClient    kubernetes.Interface
		dynamicClient dynamic.Interface
		expectRegion  string
		expectErr     error
	}{
		{
			name:          "OCP on AWS",
			kubeClient:    newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(PlatformAWS)),
			expectRegion:  "region-aws",
			expectErr:     nil,
		},
		{
			name:          "OCP on GCP",
			kubeClient:    newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, nil),
			dynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), newInfraConfig(PlatformGCP)),
			expectRegion:  "region-gcp",
			expectErr:     nil,
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
			clusterClaimer := ClusterClaimer{KubeClient: test.kubeClient, DynamicClient: test.dynamicClient}
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
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformAWS,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on Azure",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformAzure)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformAzure,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on IBM",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformIBM)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformIBM,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on IBMZ",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformIBMZ)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformIBMZ,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on IBMP",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformIBMP)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformIBMP,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on GCP",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformGCP)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformGCP,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on Openstack",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformOpenStack)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformOpenStack,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on VSphere",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformVSphere)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformVSphere,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is OCP on VSphere",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource()}, []runtime.Object{newNode(PlatformVSphere)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformVSphere,
			expectProduct:     ProductOpenShift,
			expectErr:         nil,
		},
		{
			name:              "is AKS",
			kubeClient:        newFakeKubeClient(nil, []runtime.Object{newNode(PlatformAzure)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
			expectPlatform:    PlatformAzure,
			expectProduct:     ProductAKS,
			expectErr:         nil,
		},
		{
			name:              "is OSD on AWS",
			kubeClient:        newFakeKubeClient([]*metav1.APIResourceList{projectAPIResource(), managedAPIResource()}, []runtime.Object{newNode(PlatformAWS)}),
			expectKubeVersion: "v0.0.0-master+$Format:%h$",
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
