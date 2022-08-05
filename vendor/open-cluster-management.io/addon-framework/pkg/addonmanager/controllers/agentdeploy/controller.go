package agentdeploy

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/cache"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/basecontroller/factory"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	addoninformerv1alpha1 "open-cluster-management.io/api/client/addon/informers/externalversions/addon/v1alpha1"
	addonlisterv1alpha1 "open-cluster-management.io/api/client/addon/listers/addon/v1alpha1"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1"
	clusterlister "open-cluster-management.io/api/client/cluster/listers/cluster/v1"
	workv1client "open-cluster-management.io/api/client/work/clientset/versioned"
	workinformers "open-cluster-management.io/api/client/work/informers/externalversions/work/v1"
	worklister "open-cluster-management.io/api/client/work/listers/work/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workapiv1 "open-cluster-management.io/api/work/v1"
)

// addonDeployController deploy addon agent resources on the managed cluster.
type addonDeployController struct {
	workClient                workv1client.Interface
	addonClient               addonv1alpha1client.Interface
	managedClusterLister      clusterlister.ManagedClusterLister
	managedClusterAddonLister addonlisterv1alpha1.ManagedClusterAddOnLister
	workLister                worklister.ManifestWorkLister
	agentAddons               map[string]agent.AgentAddon
	cache                     *workCache
}

func NewAddonDeployController(
	workClient workv1client.Interface,
	addonClient addonv1alpha1client.Interface,
	clusterInformers clusterinformers.ManagedClusterInformer,
	addonInformers addoninformerv1alpha1.ManagedClusterAddOnInformer,
	workInformers workinformers.ManifestWorkInformer,
	agentAddons map[string]agent.AgentAddon,
) factory.Controller {
	c := &addonDeployController{
		workClient:                workClient,
		addonClient:               addonClient,
		managedClusterLister:      clusterInformers.Lister(),
		managedClusterAddonLister: addonInformers.Lister(),
		workLister:                workInformers.Lister(),
		agentAddons:               agentAddons,
		cache:                     newWorkCache(),
	}

	return factory.New().WithFilteredEventsInformersQueueKeysFunc(
		func(obj runtime.Object) []string {
			key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			return []string{key}
		},
		func(obj interface{}) bool {
			accessor, _ := meta.Accessor(obj)
			if _, ok := c.agentAddons[accessor.GetName()]; !ok {
				return false
			}

			return true
		},
		addonInformers.Informer()).
		WithFilteredEventsInformersQueueKeysFunc(
			func(obj runtime.Object) []string {
				accessor, _ := meta.Accessor(obj)
				// in hosted mode, need get the addon namespace from the AddonNamespaceLabel, because
				// the namespaces of manifestWork and addon may be different.
				// in default mode, the addon and manifestWork are in the cluster namespace.
				if addonNamespace, ok := accessor.GetLabels()[constants.AddonNamespaceLabel]; ok {
					return []string{fmt.Sprintf("%s/%s", addonNamespace, accessor.GetLabels()[constants.AddonLabel])}
				}
				return []string{fmt.Sprintf("%s/%s", accessor.GetNamespace(), accessor.GetLabels()[constants.AddonLabel])}
			},
			func(obj interface{}) bool {
				accessor, _ := meta.Accessor(obj)
				if accessor.GetLabels() == nil {
					return false
				}

				// only watch the addon deploy/hook manifestWorks here.
				addonName, ok := accessor.GetLabels()[constants.AddonLabel]
				if !ok {
					return false
				}

				if _, ok := c.agentAddons[addonName]; !ok {
					return false
				}

				// in hosted mode, the addon namespace should be defined in the label of the addon deploy/hook manifestWork.
				// the names of manifestWork are `addon-<addon name>-deploy-hosting-<addon namespace>` and
				// `addon-<addon name>-pre-delete-hosting-<addon namespace>`
				if addonNamespace, ok := accessor.GetLabels()[constants.AddonNamespaceLabel]; ok {
					if accessor.GetName() == constants.DeployHostingWorkName(addonNamespace, addonName) ||
						accessor.GetName() == constants.PreDeleteHookHostingWorkName(addonNamespace, addonName) {
						return true
					}
					return false
				}
				// in default mode, the addon and addon deploy/hook manifestWork are in cluster namespace.
				// the names of manifestWork are `addon-<addon name>-deploy` and `addon-<addon name>-pre-delete`.
				if accessor.GetName() == constants.DeployWorkName(addonName) ||
					accessor.GetName() == constants.PreDeleteHookWorkName(addonName) {
					return true
				}
				return false
			},
			workInformers.Informer(),
		).
		WithSync(c.sync).ToController("addon-deploy-controller")
}

type addonDeploySyncer interface {
	sync(ctx context.Context, syncCtx factory.SyncContext,
		cluster *clusterv1.ManagedCluster,
		addon *addonapiv1alpha1.ManagedClusterAddOn) (*addonapiv1alpha1.ManagedClusterAddOn, error)
}

func (c *addonDeployController) sync(ctx context.Context, syncCtx factory.SyncContext, key string) error {
	clusterName, addonName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// ignore addon whose key is not in format: namespace/name
		return nil
	}

	agentAddon, ok := c.agentAddons[addonName]
	if !ok {
		return nil
	}

	addon, err := c.managedClusterAddonLister.ManagedClusterAddOns(clusterName).Get(addonName)
	if errors.IsNotFound(err) {
		return c.cleanupCache(ctx, clusterName, addonName)
	}
	if err != nil {
		return err
	}

	cluster, err := c.managedClusterLister.Get(clusterName)
	if errors.IsNotFound(err) {
		// the managedCluster is nil in this case,and sync cannot handle nil managedCluster.
		// TODO: consider to force delete the addon and its deploy manifestWorks.
		return nil
	}
	if err != nil {
		return err
	}

	syncers := []addonDeploySyncer{
		&defaultSyncer{controller: c, agentAddon: agentAddon},
		&hostedSyncer{controller: c, agentAddon: agentAddon},
		&defaultHookSyncer{controller: c, agentAddon: agentAddon},
		&hostedHookSyncer{controller: c, agentAddon: agentAddon},
	}

	oldAddon := addon
	addon = addon.DeepCopy()
	var errs []error
	for _, s := range syncers {
		var err error
		addon, err = s.sync(ctx, syncCtx, cluster, addon)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if err = c.updateAddon(ctx, addon, oldAddon); err != nil {
		return err
	}
	return errorsutil.NewAggregate(errs)
}

// updateAddon updates finalizers and conditions of addon.
// to avoid conflict updateAddon updates finalizers firstly if finalizers has change.
func (c *addonDeployController) updateAddon(ctx context.Context, new, old *addonapiv1alpha1.ManagedClusterAddOn) error {
	if !equality.Semantic.DeepEqual(new.GetFinalizers(), old.GetFinalizers()) {
		_, err := c.addonClient.AddonV1alpha1().ManagedClusterAddOns(new.Namespace).Update(ctx, new, metav1.UpdateOptions{})
		return err
	}

	return utils.PatchAddonCondition(ctx, c.addonClient, new, old)
}

// cleanupCache is only to remove cache for deploy work in default mode when the addon is not found.
// could append the other cleanup in the future. currently there are 2 scenarios here:
// 1.there is no finalizer on the addon in default mode without pre-delete hook. need to remove cache here.
// 2.there are finalizer on the addon in default mod with pre-delete hook and hosted mode. cleanup will be
// controlled by finalizers in sync.
func (c *addonDeployController) cleanupCache(ctx context.Context, clusterName, addonName string) error {
	c.cache.removeCache(constants.DeployWorkName(addonName), clusterName)
	return nil
}

func (c *addonDeployController) findHostingCluster(addonNamespace, addonName string) (string, error) {
	nsReq, err := labels.NewRequirement(constants.AddonNamespaceLabel, selection.Equals, []string{addonNamespace})
	if err != nil {
		return "", fmt.Errorf("new namespace requirement for addon %s/%s error: %s", addonNamespace, addonName, err)
	}
	nameReq, err := labels.NewRequirement(constants.AddonLabel, selection.Equals, []string{addonName})
	if err != nil {
		return "", fmt.Errorf("new name requirement for addon %s/%s error: %s", addonNamespace, addonName, err)
	}

	mws, err := c.workLister.ManifestWorks(metav1.NamespaceAll).List(labels.NewSelector().Add(*nsReq, *nameReq))
	if err != nil {
		return "", fmt.Errorf("list manifestwork for addon %s/%s error: %s", addonNamespace, addonName, err)
	}
	for _, mw := range mws {
		if mw.Name == constants.DeployHostingWorkName(addonNamespace, addonName) {
			return mw.Namespace, nil
		}
	}

	return "", fmt.Errorf("hosting cluster not found")
}

func (c *addonDeployController) buildManifestWorks(ctx context.Context,
	agentAddon agent.AgentAddon,
	installMode, workNamespace string,
	cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (deployWork, hookWork *workapiv1.ManifestWork, err error) {
	var appliedType string
	var workBuilder *manifestWorkBuiler
	var owner *metav1.OwnerReference

	switch installMode {
	case constants.InstallModeHosted:
		appliedType = constants.AddonHostingManifestApplied
		workBuilder = newHostingManifestWorkBuilder(agentAddon.GetAgentAddonOptions().HostedModeEnabled)
	case constants.InstallModeDefault:
		appliedType = constants.AddonManifestApplied
		workBuilder = newManagedManifestWorkBuilder(agentAddon.GetAgentAddonOptions().HostedModeEnabled)
		owner = metav1.NewControllerRef(addon, addonapiv1alpha1.GroupVersion.WithKind("ManagedClusterAddOn"))
	default:
		return nil, nil, fmt.Errorf("invalid install mode %v", installMode)
	}

	objects, err := agentAddon.Manifests(cluster, addon)
	if err != nil {
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    appliedType,
			Status:  metav1.ConditionFalse,
			Reason:  constants.AddonManifestAppliedReasonWorkApplyFailed,
			Message: fmt.Sprintf("failed to get manifest from agent interface: %v", err),
		})
		return nil, nil, err
	}
	if len(objects) == 0 {
		return nil, nil, nil
	}

	deployWork, hookWork, err = workBuilder.buildManifestWorkFromObject(workNamespace, addon, objects)
	if err != nil {
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    appliedType,
			Status:  metav1.ConditionFalse,
			Reason:  constants.AddonManifestAppliedReasonWorkApplyFailed,
			Message: fmt.Sprintf("failed to build manifestwork: %v", err),
		})
		return deployWork, hookWork, err
	}
	if deployWork != nil {
		if owner != nil {
			deployWork.OwnerReferences = []metav1.OwnerReference{*owner}
		}
		setStatusFeedbackRule(deployWork, agentAddon)
	}

	if hookWork != nil && owner != nil {
		hookWork.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	return deployWork, hookWork, nil
}

func (c *addonDeployController) applyWork(ctx context.Context, appliedType string,
	work *workapiv1.ManifestWork, addon *addonapiv1alpha1.ManagedClusterAddOn) (*workapiv1.ManifestWork, error) {

	work, err := applyWork(ctx, c.workClient, c.workLister, c.cache, work)
	if err != nil {
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    appliedType,
			Status:  metav1.ConditionFalse,
			Reason:  constants.AddonManifestAppliedReasonWorkApplyFailed,
			Message: fmt.Sprintf("failed to apply manifestWork: %v", err),
		})
		return work, err
	}

	// Update addon status based on work's status
	if meta.IsStatusConditionTrue(work.Status.Conditions, workapiv1.WorkApplied) {
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    appliedType,
			Status:  metav1.ConditionTrue,
			Reason:  constants.AddonManifestAppliedReasonManifestsApplied,
			Message: "manifests of addon are applied successfully",
		})
	} else {
		meta.SetStatusCondition(&addon.Status.Conditions, metav1.Condition{
			Type:    appliedType,
			Status:  metav1.ConditionFalse,
			Reason:  constants.AddonManifestAppliedReasonManifestsApplyFailed,
			Message: "failed to apply the manifests of addon",
		})
	}
	return work, nil
}

func setStatusFeedbackRule(work *workapiv1.ManifestWork, agentAddon agent.AgentAddon) {
	if agentAddon.GetAgentAddonOptions().HealthProber == nil {
		return
	}

	if agentAddon.GetAgentAddonOptions().HealthProber.Type != agent.HealthProberTypeWork {
		return
	}

	if agentAddon.GetAgentAddonOptions().HealthProber.WorkProber == nil {
		return
	}

	probeRules := agentAddon.GetAgentAddonOptions().HealthProber.WorkProber.ProbeFields

	work.Spec.ManifestConfigs = []workapiv1.ManifestConfigOption{}

	for _, rule := range probeRules {
		work.Spec.ManifestConfigs = append(work.Spec.ManifestConfigs, workapiv1.ManifestConfigOption{
			ResourceIdentifier: rule.ResourceIdentifier,
			FeedbackRules:      rule.ProbeRules,
		})
	}
}
