package agentdeploy

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/basecontroller/factory"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workapiv1 "open-cluster-management.io/api/work/v1"
)

type hostedSyncer struct {
	buildWorks func(installMode, workNamespace string, cluster *clusterv1.ManagedCluster, existingWorks []*workapiv1.ManifestWork,
		addon *addonapiv1alpha1.ManagedClusterAddOn) (appliedWorks, deleteWorks []*workapiv1.ManifestWork, err error)

	applyWork func(ctx context.Context, appliedType string,
		work *workapiv1.ManifestWork, addon *addonapiv1alpha1.ManagedClusterAddOn) (*workapiv1.ManifestWork, error)

	deleteWork func(ctx context.Context, workNamespace, workName string) error

	getWorkByAddon func(addonName, addonNamespace string) ([]*workapiv1.ManifestWork, error)

	getCluster func(clusterName string) (*clusterv1.ManagedCluster, error)

	agentAddon agent.AgentAddon
}

func (s *hostedSyncer) sync(ctx context.Context,
	syncCtx factory.SyncContext,
	cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (*addonapiv1alpha1.ManagedClusterAddOn, error) {
	// Hosted mode is not enabled, will not deploy any resource on the hosting cluster
	if !s.agentAddon.GetAgentAddonOptions().HostedModeEnabled {
		return addon, nil
	}

	installMode, hostingClusterName := constants.GetHostedModeInfo(addon.GetAnnotations())
	if installMode != constants.InstallModeHosted {
		// the installMode is changed from hosted to default, cleanup the hosting resources
		if err := s.cleanupDeployWork(ctx, addon); err != nil {
			return addon, err
		}
		addonRemoveFinalizer(addon, constants.HostingManifestFinalizer)
		return addon, nil
	}

	// Get Hosting Cluster, check whether the hosting cluster is a managed cluster of the hub
	// TODO: check whether the hosting cluster of the addon is the same hosting cluster of the klusterlet
	hostingCluster, err := s.getCluster(hostingClusterName)
	if errors.IsNotFound(err) {
		if err = s.cleanupDeployWork(ctx, addon); err != nil {
			return addon, err
		}

		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    constants.HostingClusterValidity,
			Status:  metav1.ConditionFalse,
			Reason:  constants.HostingClusterValidityReasonInvalid,
			Message: fmt.Sprintf("hosting cluster %s is not a managed cluster of the hub", hostingClusterName),
		})

		addonRemoveFinalizer(addon, constants.HostingManifestFinalizer)
		return addon, nil
	}
	if err != nil {
		return addon, err
	}
	meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
		Type:    constants.HostingClusterValidity,
		Status:  metav1.ConditionTrue,
		Reason:  constants.HostingClusterValidityReasonValid,
		Message: fmt.Sprintf("hosting cluster %s is a managed cluster of the hub", hostingClusterName),
	})

	if !hostingCluster.DeletionTimestamp.IsZero() {
		if err = s.cleanupDeployWork(ctx, addon); err != nil {
			return addon, err
		}
		addonRemoveFinalizer(addon, constants.HostingManifestFinalizer)
		return addon, nil
	}

	if !addon.DeletionTimestamp.IsZero() {
		// clean up the deploy work until the hook work is completed
		if addonHasFinalizer(addon, constants.HostingPreDeleteHookFinalizer) {
			return addon, nil
		}

		if err = s.cleanupDeployWork(ctx, addon); err != nil {
			return addon, err
		}
		addonRemoveFinalizer(addon, constants.HostingManifestFinalizer)
		return addon, nil
	}

	if addonAddFinalizer(addon, constants.HostingManifestFinalizer) {
		return addon, nil
	}

	// waiting for the addon to be deleted when cluster is deleting.
	// TODO: consider to delete addon in this scenario.
	if !cluster.DeletionTimestamp.IsZero() {
		return addon, nil
	}

	currentWorks, err := s.getWorkByAddon(addon.Name, addon.Namespace)
	if err != nil {
		return addon, err
	}

	deployWorks, deleteWorks, err := s.buildWorks(constants.InstallModeHosted, hostingClusterName, cluster, currentWorks, addon)
	if err != nil {
		return addon, err
	}

	var errs []error
	for _, deleteWork := range deleteWorks {
		err = s.deleteWork(ctx, deleteWork.Namespace, deleteWork.Name)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, deployWork := range deployWorks {
		_, err = s.applyWork(ctx, constants.AddonHostingManifestApplied, deployWork, addon)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return addon, utilerrors.NewAggregate(errs)
}

// cleanupDeployWork will delete the hosting manifestWork and cache. if the hostingClusterName is empty, will try
// to find out the hosting cluster by manifestWork labels and do the cleanup.
func (s *hostedSyncer) cleanupDeployWork(ctx context.Context,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (err error) {
	if !addonHasFinalizer(addon, constants.HostingManifestFinalizer) {
		return nil
	}

	currentWorks, err := s.getWorkByAddon(addon.Name, addon.Namespace)
	if err != nil {
		return err
	}

	var errs []error
	for _, work := range currentWorks {
		err = s.deleteWork(ctx, work.Namespace, work.Name)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}
