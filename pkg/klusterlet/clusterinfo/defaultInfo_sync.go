package controllers

import (
	"fmt"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/labels"
	clusterv1alpha1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"
)

type defaultInfoSyncer struct {
	claimLister clusterv1alpha1lister.ClusterClaimLister
}

func (s *defaultInfoSyncer) sync(ctx context.Context, clusterInfo *clusterv1beta1.ManagedClusterInfo) error {
	claims, err := s.claimLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list clusterClaims error:%v ", err)
	}
	for _, claim := range claims {
		value := claim.Spec.Value
		switch claim.Name {
		case clusterclaim.ClaimOCMConsoleURL:
			clusterInfo.Status.ConsoleURL = value
		case clusterclaim.ClaimOCMKubeVersion:
			clusterInfo.Status.Version = value
		case clusterclaim.ClaimOpenshiftID:
			clusterInfo.Status.ClusterID = value
		case clusterclaim.ClaimOCMProduct:
			clusterInfo.Status.KubeVendor = getKubeVendor(value)
		case clusterclaim.ClaimOCMPlatform:
			clusterInfo.Status.CloudVendor = getCloudVendor(value)
		}
	}

	return nil
}

func getCloudVendor(platform string) (cloudVendor clusterv1beta1.CloudVendorType) {
	switch platform {
	case clusterclaim.PlatformAzure:
		cloudVendor = clusterv1beta1.CloudVendorAzure
	case clusterclaim.PlatformAWS:
		cloudVendor = clusterv1beta1.CloudVendorAWS
	case clusterclaim.PlatformIBM:
		cloudVendor = clusterv1beta1.CloudVendorIBM
	case clusterclaim.PlatformIBMZ:
		cloudVendor = clusterv1beta1.CloudVendorIBMZ
	case clusterclaim.PlatformIBMP:
		cloudVendor = clusterv1beta1.CloudVendorIBMP
	case clusterclaim.PlatformGCP:
		cloudVendor = clusterv1beta1.CloudVendorGoogle
	case clusterclaim.PlatformVSphere:
		cloudVendor = clusterv1beta1.CloudVendorVSphere
	case clusterclaim.PlatformOpenStack:
		cloudVendor = clusterv1beta1.CloudVendorOpenStack
	case clusterclaim.PlatformRHV:
		cloudVendor = clusterv1beta1.CloudVendorRHV
	case clusterclaim.PlatformAlibabaCloud:
		cloudVendor = clusterv1beta1.CloudVendorAlibabaCloud
	case clusterclaim.PlatformBareMetal:
		cloudVendor = clusterv1beta1.CloudVendorBareMetal
	default:
		cloudVendor = clusterv1beta1.CloudVendorOther
	}
	return
}
func getKubeVendor(product string) (kubeVendor clusterv1beta1.KubeVendorType) {
	switch product {
	case clusterclaim.ProductAKS:
		return clusterv1beta1.KubeVendorAKS
	case clusterclaim.ProductGKE:
		return clusterv1beta1.KubeVendorGKE
	case clusterclaim.ProductEKS:
		return clusterv1beta1.KubeVendorEKS
	case clusterclaim.ProductIKS:
		return clusterv1beta1.KubeVendorIKS
	case clusterclaim.ProductICP:
		return clusterv1beta1.KubeVendorICP
	case clusterclaim.ProductOSD:
		return clusterv1beta1.KubeVendorOSD
	case clusterclaim.ProductMicroShift:
		return clusterv1beta1.KubeVendorMicroShift
	}

	if isProductOCP(product) {
		return clusterv1beta1.KubeVendorOpenShift
	}

	return clusterv1beta1.KubeVendorOther
}

func isProductOCP(product string) bool {
	for _, productOCP := range clusterclaim.ProductOCPList {
		if productOCP == product {
			return true
		}
	}
	return false
}
