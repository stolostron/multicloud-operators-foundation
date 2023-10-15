package controllers

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	clusterv1alpha1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"
)

type distributionInfoSyncer struct {
	configV1Client       openshiftclientset.Interface
	managedClusterClient kubernetes.Interface
	claimLister          clusterv1alpha1lister.ClusterClaimLister
}

func (s *distributionInfoSyncer) sync(ctx context.Context, clusterInfo *clusterv1beta1.ManagedClusterInfo) error {
	// currently we only support OCP in DistributionInfo
	switch clusterInfo.Status.KubeVendor {
	case clusterv1beta1.KubeVendorOpenShift, clusterv1beta1.KubeVendorOSD:
		return s.syncOCPDistributionInfo(ctx, &clusterInfo.Status)
	}

	return nil
}

func (s *distributionInfoSyncer) syncOCPDistributionInfo(ctx context.Context, clusterInfoStatus *clusterv1beta1.ClusterInfoStatus) error {
	lastAppliedAPIServerURL := clusterInfoStatus.DistributionInfo.OCP.LastAppliedAPIServerURL
	clusterInfoStatus.DistributionInfo = clusterv1beta1.DistributionInfo{
		Type: clusterv1beta1.DistributionTypeOCP,
	}
	clusterVersion, err := s.configV1Client.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		clusterInfoStatus.DistributionInfo.OCP.Version = clusterclaim.OCP3Version
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get clusterVersion.error %v", err)
	}

	ocpDistributionInfo := clusterv1beta1.OCPDistributionInfo{
		Channel:        clusterVersion.Spec.Channel,
		DesiredVersion: clusterVersion.Status.Desired.Version,
		Desired: clusterv1beta1.OCPVersionRelease{
			Version:  clusterVersion.Status.Desired.Version,
			Image:    clusterVersion.Status.Desired.Image,
			URL:      string(clusterVersion.Status.Desired.URL),
			Channels: clusterVersion.Status.Desired.Channels,
		},
		// do not override the LastAppliedAPIServerURL to empty
		LastAppliedAPIServerURL: lastAppliedAPIServerURL,
	}

	availableUpdates := clusterVersion.Status.AvailableUpdates
	for _, update := range availableUpdates {
		versionUpdate := clusterv1beta1.OCPVersionRelease{}
		versionUpdate.Version = update.Version
		versionUpdate.Image = update.Image
		versionUpdate.URL = string(update.URL)
		versionUpdate.Channels = update.Channels
		if versionUpdate.Version != "" {
			// ocpDistributionInfo.AvailableUpdates is deprecated in release 2.3 and will be removed in the future.
			// Use VersionAvailableUpdates instead.
			ocpDistributionInfo.AvailableUpdates = append(ocpDistributionInfo.AvailableUpdates, versionUpdate.Version)
			ocpDistributionInfo.VersionAvailableUpdates = append(ocpDistributionInfo.VersionAvailableUpdates, versionUpdate)
		}

	}
	historyItems := clusterVersion.Status.History
	for _, historyItem := range historyItems {
		history := clusterv1beta1.OCPVersionUpdateHistory{}
		history.State = string(historyItem.State)
		history.Image = historyItem.Image
		history.Version = historyItem.Version
		history.Verified = historyItem.Verified
		ocpDistributionInfo.VersionHistory = append(ocpDistributionInfo.VersionHistory, history)
	}

	version, err := s.claimLister.Get(clusterclaim.ClaimOpenshiftVersion)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to get ocp version, error: %v", err)
	}
	if version != nil {
		ocpDistributionInfo.Version = version.Spec.Value
	}

	ocpDistributionInfo.ManagedClusterClientConfig = s.getClientConfig(ctx, clusterInfoStatus.CloudVendor)
	ocpDistributionInfo.UpgradeFailed = false
	conditions := clusterVersion.Status.Conditions

	for _, condition := range conditions {
		if condition.Type == "Failing" {
			if condition.Status == configv1.ConditionTrue && ocpDistributionInfo.DesiredVersion != ocpDistributionInfo.Version {
				ocpDistributionInfo.UpgradeFailed = true
			}
			break
		}
	}

	clusterInfoStatus.DistributionInfo.OCP = ocpDistributionInfo
	return nil
}

func (s *distributionInfoSyncer) getClientConfig(ctx context.Context, cloudVendor clusterv1beta1.CloudVendorType) clusterv1beta1.ClientConfig {
	// get ocp apiserver url
	kubeAPIServer, err := utils.GetKubeAPIServerAddress(ctx, s.configV1Client)
	if err != nil {
		klog.Errorf("Failed to get kube Apiserver. err:%v", err)
		return clusterv1beta1.ClientConfig{}
	}

	// get ocp ca
	clusterca := s.getClusterCA(ctx, kubeAPIServer, cloudVendor)
	if len(clusterca) <= 0 {
		klog.Errorf("Failed to get clusterca.")
		return clusterv1beta1.ClientConfig{}
	}

	return clusterv1beta1.ClientConfig{
		URL:      kubeAPIServer,
		CABundle: clusterca,
	}
}

func (s *distributionInfoSyncer) getClusterCA(ctx context.Context, kubeAPIServer string, cloudVendor clusterv1beta1.CloudVendorType) []byte {
	// Get ca from apiserver
	certData, err := utils.GetCAFromApiserver(ctx, s.configV1Client, s.managedClusterClient, kubeAPIServer)
	if err == nil && len(certData) > 0 {
		return certData
	}

	// Get ca from configmap in kube-public namespace
	certData, err = utils.GetCAFromConfigMap(ctx, s.managedClusterClient)
	if err == nil && len(certData) > 0 {
		return certData
	}

	// Fallback to service account token ca.crt
	certData, err = utils.GetCAFromServiceAccount(ctx, s.managedClusterClient)
	if err == nil && len(certData) > 0 {
		// check if it's roks
		// if it's ocp && it's on ibm cloud, we treat it as roks
		if cloudVendor == clusterv1beta1.CloudVendorIBM {
			// simply don't give any certs as the apiserver is using certs signed by known CAs
			return nil
		}
		return certData
	}

	klog.Warningf("Cannot get ca from service account, error:%v", err)
	return nil
}
