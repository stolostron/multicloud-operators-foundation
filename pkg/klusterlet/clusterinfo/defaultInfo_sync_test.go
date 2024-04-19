package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

func newClaim(name, value string) *clusterv1alpha1.ClusterClaim {
	return &clusterv1alpha1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: clusterv1alpha1.ClusterClaimSpec{
			Value: value,
		},
	}
}

func Test_defaultInfo_syncer(t *testing.T) {
	tests := []struct {
		name               string
		claims             []runtime.Object
		managedClusterInfo *v1beta1.ManagedClusterInfo
		validate           func(managedClusterInfo *v1beta1.ManagedClusterInfo)
	}{
		{
			name: "OCP cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductOpenShift),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformAWS),
				newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ConsoleURL != "https://abc.com" ||
					managedClusterInfo.Status.ClusterID != "aaa-bbb" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorOpenShift ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorAWS {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
		{
			name: "AKS cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductAKS),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformAzure),
				newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ConsoleURL != "https://abc.com" ||
					managedClusterInfo.Status.ClusterID != "aaa-bbb" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorAKS ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorAzure {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
		{
			name: "GKE cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductGKE),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformGCP),
				newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ConsoleURL != "https://abc.com" ||
					managedClusterInfo.Status.ClusterID != "aaa-bbb" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorGKE ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorGoogle {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
		{
			name: "OSD cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductOSD),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformIBM),
				newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ConsoleURL != "https://abc.com" ||
					managedClusterInfo.Status.ClusterID != "aaa-bbb" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorOSD ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorIBM {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
		{
			name: "IKS cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductIKS),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformOpenStack),
				newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ConsoleURL != "https://abc.com" ||
					managedClusterInfo.Status.ClusterID != "aaa-bbb" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorIKS ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorOpenStack {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
		{
			name: "Other cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimOCMConsoleURL, "https://abc.com"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, "test"),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformIBMP),
				newClaim(clusterclaim.ClaimOpenshiftID, "aaa-bbb"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ConsoleURL != "https://abc.com" ||
					managedClusterInfo.Status.ClusterID != "aaa-bbb" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorOther ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorIBMP {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
		{
			name: "Microshift cluster",
			claims: []runtime.Object{
				newClaim(clusterclaim.ClaimK8sID, "aaa-bbb"),
				newClaim(clusterclaim.ClaimOCMKubeVersion, "v1.20.0"),
				newClaim(clusterclaim.ClaimOCMProduct, clusterclaim.ProductMicroShift),
				newClaim(clusterclaim.ClaimOCMPlatform, clusterclaim.PlatformOther),
				newClaim(clusterclaim.ClaimMicroShiftID, "aaa-bbb-ccc"),
				newClaim(clusterclaim.ClaimMicroShiftVersion, "4.16.0"),
			},
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
			},
			validate: func(managedClusterInfo *v1beta1.ManagedClusterInfo) {
				if managedClusterInfo.Status.ClusterID != "aaa-bbb-ccc" ||
					managedClusterInfo.Status.Version != "v1.20.0" ||
					managedClusterInfo.Status.KubeVendor != v1beta1.KubeVendorMicroShift ||
					managedClusterInfo.Status.CloudVendor != v1beta1.CloudVendorOther {
					t.Errorf("failed to validate defaut info")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClusterClient := clusterfake.NewSimpleClientset(test.claims...)
			clusterInformerFactory := clusterinformers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
			syncer := defaultInfoSyncer{claimLister: clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister()}
			clusterStore := clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Informer().GetStore()
			for _, item := range test.claims {
				clusterStore.Add(item)
			}

			err := syncer.sync(context.TODO(), test.managedClusterInfo)
			if err != nil {
				t.Errorf("failed to sync default info")
			}
			test.validate(test.managedClusterInfo)
		})
	}

}
