package controllers

import (
	"context"
	"reflect"
	"testing"
	"time"

	apiconfigv1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
)

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

func Test_distributionInfo_syncer(t *testing.T) {
	tests := []struct {
		name                          string
		managedClusterInfo            *v1beta1.ManagedClusterInfo
		configV1Client                openshiftclientset.Interface
		expectChannel                 string
		expectDesiredVersion          string
		expectDesiredChannelLen       int
		expectUpgradeFail             bool
		expectAvailableUpdatesLen     int
		expectChannelAndURL           bool
		expectLastAppliedAPIServerURL string
		expectHistoryLen              int
		expectError                   string
		expectVersion                 string
		claims                        []runtime.Object
	}{
		{
			name: "OSD",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorOSD,
					DistributionInfo: v1beta1.DistributionInfo{
						Type: v1beta1.DistributionTypeOCP,
						OCP: v1beta1.OCPDistributionInfo{
							LastAppliedAPIServerURL: "http://test-last-applied-url",
						},
					},
				},
			},
			configV1Client:                newConfigV1Client("4.x", false),
			expectChannel:                 "stable-4.5",
			expectDesiredVersion:          "4.6.8",
			expectDesiredChannelLen:       7,
			expectUpgradeFail:             false,
			expectAvailableUpdatesLen:     2,
			expectChannelAndURL:           true,
			expectHistoryLen:              3,
			expectLastAppliedAPIServerURL: "http://test-last-applied-url",
			expectError:                   "",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOpenshiftVersion, "4.6.8"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
			},
			expectVersion: "4.6.8",
		},
		{
			name: "OCP4.x",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorOpenShift,
					DistributionInfo: v1beta1.DistributionInfo{
						Type: v1beta1.DistributionTypeOCP,
						OCP: v1beta1.OCPDistributionInfo{
							LastAppliedAPIServerURL: "http://test-last-applied-url",
						},
					},
				},
			},
			configV1Client:                newConfigV1Client("4.x", false),
			expectChannel:                 "stable-4.5",
			expectDesiredVersion:          "4.6.8",
			expectDesiredChannelLen:       7,
			expectUpgradeFail:             false,
			expectAvailableUpdatesLen:     2,
			expectChannelAndURL:           true,
			expectHistoryLen:              3,
			expectLastAppliedAPIServerURL: "http://test-last-applied-url",
			expectError:                   "",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOpenshiftVersion, "4.6.8"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
			},
			expectVersion: "4.6.8",
		},
		{
			name: "OCP3.x",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorOpenShift,
				},
			},
			configV1Client: newConfigV1Client("3", true),
			expectError:    "",
			expectVersion:  "3",
		},
		{
			name: "UpgradeFail",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorOpenShift,
				},
			},
			configV1Client:            newConfigV1Client("4.x", true),
			expectChannel:             "stable-4.5",
			expectDesiredChannelLen:   7,
			expectDesiredVersion:      "4.6.8",
			expectUpgradeFail:         true,
			expectAvailableUpdatesLen: 2,
			expectChannelAndURL:       false,
			expectHistoryLen:          1,
			expectError:               "",
			claims:                    []runtime.Object{},
			expectVersion:             "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClusterClient := clusterfake.NewSimpleClientset(test.claims...)
			clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
			clusterStore := clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Informer().GetStore()
			for _, item := range test.claims {
				clusterStore.Add(item)
			}
			syncer := distributionInfoSyncer{
				configV1Client:       test.configV1Client,
				managedClusterClient: nil,
				claimLister:          clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
			}

			err := syncer.sync(context.TODO(), test.managedClusterInfo)
			if err != nil {
				assert.Equal(t, err.Error(), test.expectError)
			}
			info := test.managedClusterInfo.Status.DistributionInfo.OCP
			assert.Equal(t, info.Channel, test.expectChannel)
			assert.Equal(t, info.DesiredVersion, test.expectDesiredVersion)
			assert.Equal(t, len(info.Desired.Channels), test.expectDesiredChannelLen)
			assert.Equal(t, info.UpgradeFailed, test.expectUpgradeFail)
			assert.Equal(t, len(info.VersionAvailableUpdates), test.expectAvailableUpdatesLen)
			assert.Equal(t, info.LastAppliedAPIServerURL, test.expectLastAppliedAPIServerURL)

			//get the latest succeed version
			assert.Equal(t, test.expectVersion, info.Version)

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

func newInfra() *apiconfigv1.Infrastructure {
	return &apiconfigv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: apiconfigv1.InfrastructureSpec{},
		Status: apiconfigv1.InfrastructureStatus{
			APIServerURL: "http://127.0.0.1:6443",
		},
	}
}

func Test_getClientConfig(t *testing.T) {
	tests := []struct {
		name               string
		managedClusterInfo *v1beta1.ManagedClusterInfo
		configV1Client     openshiftclientset.Interface
		existingKubeOjb    []runtime.Object
		existingOcpOjb     []runtime.Object
		expectClientConfig v1beta1.ClientConfig
	}{
		{
			name:               "Apiserver url not found",
			existingKubeOjb:    []runtime.Object{},
			existingOcpOjb:     []runtime.Object{},
			expectClientConfig: v1beta1.ClientConfig{},
		},
		{
			name:            "clusterca not found",
			existingKubeOjb: []runtime.Object{},
			existingOcpOjb: []runtime.Object{
				&apiconfigv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: apiconfigv1.InfrastructureSpec{},
					Status: apiconfigv1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
			},
			expectClientConfig: v1beta1.ClientConfig{},
		},
		{
			name: "get apiserver ca",
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
				&apiconfigv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: apiconfigv1.InfrastructureSpec{},
					Status: apiconfigv1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
				&apiconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: apiconfigv1.APIServerSpec{
						ServingCerts: apiconfigv1.APIServerServingCerts{
							NamedCertificates: []apiconfigv1.APIServerNamedServingCert{
								{
									Names:              []string{"127.0.0.1"},
									ServingCertificate: apiconfigv1.SecretNameReference{Name: "test-secret"},
								},
							},
						},
					},
				},
			},
			expectClientConfig: v1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("fake-cert-data"),
			},
		},
		{
			name: "get configmap ca",
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
				&apiconfigv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: apiconfigv1.InfrastructureSpec{},
					Status: apiconfigv1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
				&apiconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: apiconfigv1.APIServerSpec{},
				},
			},
			expectClientConfig: v1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("configmap-fake-cert-data"),
			},
		},

		{
			name: "get service account ca",
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
				&apiconfigv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: apiconfigv1.InfrastructureSpec{},
					Status: apiconfigv1.InfrastructureStatus{
						APIServerURL: "http://127.0.0.1:6443",
					},
				},
			},
			expectClientConfig: v1beta1.ClientConfig{
				URL:      "http://127.0.0.1:6443",
				CABundle: []byte("sa-fake-cert-data"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			syncer := distributionInfoSyncer{
				configV1Client:       configfake.NewSimpleClientset(test.existingOcpOjb...),
				managedClusterClient: kubefake.NewSimpleClientset(test.existingKubeOjb...),
			}
			retConfig := syncer.getClientConfig(ctx, v1beta1.CloudVendorAWS)
			if !reflect.DeepEqual(retConfig, test.expectClientConfig) {
				t.Errorf("case:%v, expect client config is not same as return client config. "+
					"expecrClientConfigs=%v, return client config=%v", test.name, test.expectClientConfig, retConfig)
			}
			return
		})
	}
}
