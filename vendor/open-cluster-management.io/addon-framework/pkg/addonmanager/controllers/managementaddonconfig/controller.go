package managementaddonconfig

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	addoninformerv1alpha1 "open-cluster-management.io/api/client/addon/informers/externalversions/addon/v1alpha1"
	addonlisterv1alpha1 "open-cluster-management.io/api/client/addon/listers/addon/v1alpha1"

	"open-cluster-management.io/addon-framework/pkg/basecontroller/factory"
	"open-cluster-management.io/addon-framework/pkg/index"
	"open-cluster-management.io/addon-framework/pkg/utils"
)

const (
	controllerName = "management-addon-config-controller"
)

type enqueueFunc func(obj interface{})

// clusterManagementAddonConfigController reconciles all interested addon config types (GroupVersionResource) on the hub.
type clusterManagementAddonConfigController struct {
	addonClient                   addonv1alpha1client.Interface
	clusterManagementAddonLister  addonlisterv1alpha1.ClusterManagementAddOnLister
	clusterManagementAddonIndexer cache.Indexer
	configListers                 map[schema.GroupResource]dynamiclister.Lister
	queue                         workqueue.RateLimitingInterface
	addonFilterFunc               factory.EventFilterFunc
	configGVRs                    map[schema.GroupVersionResource]bool
}

func NewManagementAddonConfigController(
	addonClient addonv1alpha1client.Interface,
	clusterManagementAddonInformers addoninformerv1alpha1.ClusterManagementAddOnInformer,
	configInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	configGVRs map[schema.GroupVersionResource]bool,
	addonFilterFunc factory.EventFilterFunc,
) factory.Controller {
	syncCtx := factory.NewSyncContext(controllerName)

	c := &clusterManagementAddonConfigController{
		addonClient:                   addonClient,
		clusterManagementAddonLister:  clusterManagementAddonInformers.Lister(),
		clusterManagementAddonIndexer: clusterManagementAddonInformers.Informer().GetIndexer(),
		configListers:                 map[schema.GroupResource]dynamiclister.Lister{},
		queue:                         syncCtx.Queue(),
		addonFilterFunc:               addonFilterFunc,
		configGVRs:                    configGVRs,
	}

	configInformers := c.buildConfigInformers(configInformerFactory, configGVRs)

	return factory.New().
		WithSyncContext(syncCtx).
		WithInformersQueueKeysFunc(func(obj runtime.Object) []string {
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			return []string{key}
		}, clusterManagementAddonInformers.Informer()).
		WithBareInformers(configInformers...).
		WithSync(c.sync).ToController(controllerName)
}

func (c *clusterManagementAddonConfigController) buildConfigInformers(
	configInformerFactory dynamicinformer.DynamicSharedInformerFactory,
	configGVRs map[schema.GroupVersionResource]bool,
) []factory.Informer {
	configInformers := []factory.Informer{}
	for gvrRaw := range configGVRs {
		gvr := gvrRaw // copy the value since it will be used in the closure
		indexInformer := configInformerFactory.ForResource(gvr).Informer()
		_, err := indexInformer.AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				AddFunc: c.enqueueClusterManagementAddOnsByConfig(gvr),
				UpdateFunc: func(oldObj, newObj interface{}) {
					c.enqueueClusterManagementAddOnsByConfig(gvr)(newObj)
				},
				DeleteFunc: c.enqueueClusterManagementAddOnsByConfig(gvr),
			},
		)
		if err != nil {
			utilruntime.HandleError(err)
		}
		configInformers = append(configInformers, indexInformer)
		c.configListers[schema.GroupResource{Group: gvr.Group, Resource: gvr.Resource}] = dynamiclister.New(indexInformer.GetIndexer(), gvr)
	}
	return configInformers
}

func (c *clusterManagementAddonConfigController) enqueueClusterManagementAddOnsByConfig(gvr schema.GroupVersionResource) enqueueFunc {
	return func(obj interface{}) {
		namespaceName, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error to get accessor of object: %v", obj))
			return
		}

		objs, err := c.clusterManagementAddonIndexer.ByIndex(
			index.ClusterManagementAddonByConfig, fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Resource, namespaceName))
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("error to get addons: %v", err))
			return
		}

		for _, obj := range objs {
			if obj == nil {
				continue
			}
			key, _ := cache.MetaNamespaceKeyFunc(obj)
			c.queue.Add(key)
		}
	}
}

func (c *clusterManagementAddonConfigController) sync(ctx context.Context, syncCtx factory.SyncContext, key string) error {
	_, addonName, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// ignore addon whose key is invalid
		return nil
	}

	cma, err := c.clusterManagementAddonLister.Get(addonName)
	if errors.IsNotFound(err) {
		// addon cloud be deleted, ignore
		return nil
	}
	if err != nil {
		return err
	}

	if !c.addonFilterFunc(cma) {
		return nil
	}

	cmaCopy := cma.DeepCopy()
	if err := c.updateConfigSpecHash(cmaCopy); err != nil {
		return err
	}

	return c.patchConfigReferences(ctx, cma, cmaCopy)
}

func (c *clusterManagementAddonConfigController) updateConfigSpecHash(cma *addonapiv1alpha1.ClusterManagementAddOn) error {

	for i, defaultConfigReference := range cma.Status.DefaultConfigReferences {
		if !utils.ContainGR(
			c.configGVRs,
			defaultConfigReference.ConfigGroupResource.Group,
			defaultConfigReference.ConfigGroupResource.Resource) {
			continue
		}

		if defaultConfigReference.DesiredConfig == nil || defaultConfigReference.DesiredConfig.Name == "" {
			continue
		}

		specHash, err := c.getConfigSpecHash(defaultConfigReference.ConfigGroupResource, defaultConfigReference.DesiredConfig.ConfigReferent)
		if err != nil {
			return nil
		}
		cma.Status.DefaultConfigReferences[i].DesiredConfig.SpecHash = specHash
	}

	for i, installProgression := range cma.Status.InstallProgressions {
		for j, configReference := range installProgression.ConfigReferences {
			if configReference.DesiredConfig == nil || configReference.DesiredConfig.Name == "" {
				continue
			}

			if !utils.ContainGR(
				c.configGVRs,
				configReference.ConfigGroupResource.Group,
				configReference.ConfigGroupResource.Resource) {
				continue
			}

			specHash, err := c.getConfigSpecHash(configReference.ConfigGroupResource, configReference.DesiredConfig.ConfigReferent)
			if err != nil {
				return nil
			}
			cma.Status.InstallProgressions[i].ConfigReferences[j].DesiredConfig.SpecHash = specHash
		}
	}

	return nil
}

func (c *clusterManagementAddonConfigController) patchConfigReferences(ctx context.Context, old, new *addonapiv1alpha1.ClusterManagementAddOn) error {
	if equality.Semantic.DeepEqual(new.Status.DefaultConfigReferences, old.Status.DefaultConfigReferences) &&
		equality.Semantic.DeepEqual(new.Status.InstallProgressions, old.Status.InstallProgressions) {
		return nil
	}

	oldData, err := json.Marshal(&addonapiv1alpha1.ClusterManagementAddOn{
		Status: addonapiv1alpha1.ClusterManagementAddOnStatus{
			DefaultConfigReferences: old.Status.DefaultConfigReferences,
			InstallProgressions:     old.Status.InstallProgressions,
		},
	})
	if err != nil {
		return err
	}

	newData, err := json.Marshal(&addonapiv1alpha1.ClusterManagementAddOn{
		ObjectMeta: metav1.ObjectMeta{
			UID:             new.UID,
			ResourceVersion: new.ResourceVersion,
		},
		Status: addonapiv1alpha1.ClusterManagementAddOnStatus{
			DefaultConfigReferences: new.Status.DefaultConfigReferences,
			InstallProgressions:     new.Status.InstallProgressions,
		},
	})
	if err != nil {
		return err
	}

	patchBytes, err := jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return fmt.Errorf("failed to create patch for addon %s: %w", new.Name, err)
	}

	klog.V(4).Infof("Patching addon %s/%s config reference with %s", new.Namespace, new.Name, string(patchBytes))
	_, err = c.addonClient.AddonV1alpha1().ClusterManagementAddOns().Patch(
		ctx,
		new.Name,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
		"status",
	)
	return err
}

func (c *clusterManagementAddonConfigController) getConfigSpecHash(gr addonapiv1alpha1.ConfigGroupResource,
	cr addonapiv1alpha1.ConfigReferent) (string, error) {
	lister, ok := c.configListers[schema.GroupResource{Group: gr.Group, Resource: gr.Resource}]
	if !ok {
		return "", nil
	}

	var config *unstructured.Unstructured
	var err error
	if cr.Namespace == "" {
		config, err = lister.Get(cr.Name)
	} else {
		config, err = lister.Namespace(cr.Namespace).Get(cr.Name)
	}
	if errors.IsNotFound(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return utils.GetSpecHash(config)
}
