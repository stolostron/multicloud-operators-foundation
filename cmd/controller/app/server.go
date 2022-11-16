// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/controller-filtered-cache/filteredcache"
	routev1 "github.com/openshift/api/route/v1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveinternalv1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/addon"
	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/certrotation"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterca"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterinfo"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterrole"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clusterclaim"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clusterdeployment"
	clustersetmapper "github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clustersetmapper"
	globalclusterset "github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/globalclusterset"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/syncclusterrolebinding"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/syncrolebinding"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/gc"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/imageregistry"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/addonmanager"
	addonagent "open-cluster-management.io/addon-framework/pkg/agent"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alaph1 "open-cluster-management.io/api/cluster/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = hiveinternalv1alpha1.AddToScheme(scheme)
	_ = hivev1.AddToScheme(scheme)
	_ = clusterinfov1beta1.AddToScheme(scheme)
	_ = clusterv1.Install(scheme)
	_ = actionv1beta1.AddToScheme(scheme)
	_ = clusterv1alaph1.Install(scheme)
	_ = clusterv1beta1.Install(scheme)
	_ = clusterv1beta2.Install(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = routev1.Install(scheme)
}

func Run(o *options.ControllerRunOptions, ctx context.Context) error {

	// clusterset to cluster map
	clusterSetClusterMapper := helpers.NewClusterSetMapper()

	globalClusterSetClusterMapper := helpers.NewClusterSetMapper()

	// clusterset to namespace resource map, like clusterdeployment, clusterpool, clusterclaim. the map value format is "<ResourceType>/<Namespace>/<Name>"
	clusterSetNamespaceMapper := helpers.NewClusterSetMapper()

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", o.KubeConfig)
	if err != nil {
		klog.Errorf("unable to get kube config: %v", err)
		return err
	}

	kubeConfig.QPS = o.QPS
	kubeConfig.Burst = o.Burst

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("unable to create kube client: %v", err)
		return err
	}

	clusterClient, err := clusterv1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	addonClient, err := addonv1alpha1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	clusterInformers := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	kubeInfomers := kubeinformers.NewSharedInformerFactory(kubeClient, 10*time.Minute)

	// Get the LogCertSecret namespace
	logCertSecretNamespace, _, err := k8scache.SplitMetaNamespaceKey(
		o.LogCertSecret,
	)
	if err != nil {
		return err
	}
	if logCertSecretNamespace == "" {
		logCertSecretNamespace, err = utils.GetComponentNamespace()
		if err != nil {
			return err
		}
	}

	// Filter and cache the specific namespace's secret
	// Update the filteredcache.Selector if you want to watch other namespace's secret or want to filter other resouce.
	gvkLabelMap := map[schema.GroupVersionKind]filteredcache.Selector{
		corev1.SchemeGroupVersion.WithKind("Secret"): {
			FieldSelector: fmt.Sprintf("metadata.namespace==%s", logCertSecretNamespace),
		},
	}

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme:                 scheme,
		LeaderElectionID:       "foundation-controller",
		LeaderElection:         o.EnableLeaderElection,
		HealthProbeBindAddress: ":8000",
		NewCache:               filteredcache.NewFilteredCacheBuilder(gvkLabelMap),
	})
	if err != nil {
		klog.Errorf("unable to start manager: %v", err)
		return err
	}

	// add healthz/readyz check handler
	if err := mgr.AddHealthzCheck("healthz-ping", healthz.Ping); err != nil {
		klog.Errorf("unable to add healthz check handler: %v", err)
		return err
	}

	if err := mgr.AddReadyzCheck("readyz-ping", healthz.Ping); err != nil {
		klog.Errorf("unable to add readyz check handler: %v", err)
		return err
	}

	clusterSetAdminCache := cache.NewClusterSetCache(
		clusterInformers.Cluster().V1beta2().ManagedClusterSets(),
		kubeInfomers.Rbac().V1().ClusterRoles(),
		kubeInfomers.Rbac().V1().ClusterRoleBindings(),
		utils.GetAdminResourceFromClusterRole,
	)
	clusterSetViewCache := cache.NewClusterSetCache(
		clusterInformers.Cluster().V1beta2().ManagedClusterSets(),
		kubeInfomers.Rbac().V1().ClusterRoles(),
		kubeInfomers.Rbac().V1().ClusterRoleBindings(),
		utils.GetViewResourceFromClusterRole,
	)

	addonMgr, err := addonmanager.New(kubeConfig)
	if err != nil {
		klog.Errorf("unable to setup addon manager: %v", err)
		return err
	}
	if o.EnableAddonDeploy {
		registrationOption := addon.NewRegistrationOption(kubeClient, addon.WorkManagerAddonName)
		agentAddon, err := addonfactory.NewAgentAddonFactory(addon.WorkManagerAddonName, addon.ChartFS, addon.ChartDir).
			WithScheme(scheme).
			WithConfigGVRs(addonfactory.AddOnDeploymentConfigGVR).
			WithGetValuesFuncs(
				addon.NewGetValuesFunc(o.AddonImage),
				addonfactory.GetValuesFromAddonAnnotation,
				addonfactory.GetAddOnDeloymentConfigValues(
					addonfactory.NewAddOnDeloymentConfigGetter(addonClient),
					addonfactory.ToAddOnNodePlacementValues,
					addonfactory.ToAddOnCustomizedVariableValues,
				),
			).
			WithAgentRegistrationOption(registrationOption).
			WithInstallStrategy(addonagent.InstallAllStrategy(o.AddonInstallNamespace)).
			WithAgentHostedModeEnabledOption().
			BuildHelmAgentAddon()
		if err != nil {
			klog.Errorf("failed to build agent %v", err)
			return err
		}
		err = addonMgr.AddAgent(agentAddon)
		if err != nil {
			klog.Fatal(err)
		}
	}

	if err = clusterinfo.SetupWithManager(mgr, o.LogCertSecret); err != nil {
		klog.Errorf("unable to setup clusterInfo reconciler: %v", err)
		return err
	}

	if err = clustersetmapper.SetupWithManager(mgr, kubeClient, globalClusterSetClusterMapper, clusterSetClusterMapper, clusterSetNamespaceMapper); err != nil {
		klog.Errorf("unable to setup clustersetmapper reconciler: %v", err)
		return err
	}

	if err = globalclusterset.SetupWithManager(mgr, kubeClient); err != nil {
		klog.Errorf("unable to setup globalclusterset reconciler: %v", err)
		return err
	}

	if err = clusterdeployment.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup clusterdeployment reconciler: %v", err)
		return err
	}
	if err = clusterclaim.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup clusterclaim reconciler: %v", err)
		return err
	}
	if err = imageregistry.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup imageregistry reconciler: %v", err)
		return err
	}

	clusterrolebindingSync := syncclusterrolebinding.NewReconciler(
		kubeClient,
		clusterSetAdminCache.Cache,
		clusterSetViewCache.Cache,
		globalClusterSetClusterMapper,
		clusterSetClusterMapper,
	)

	rolebindingSync := syncrolebinding.NewReconciler(
		kubeClient,
		clusterSetAdminCache.Cache,
		clusterSetViewCache.Cache,
		globalClusterSetClusterMapper,
		clusterSetClusterMapper,
		clusterSetNamespaceMapper,
	)

	if err = clusterrole.SetupWithManager(mgr, kubeClient); err != nil {
		klog.Errorf("unable to setup clusterrole reconciler: %v", err)
		return err
	}

	if err = clusterca.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup clusterca reconciler: %v", err)
		return err
	}
	if err = gc.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup gc reconciler: %v", err)
		return err
	}

	if err = certrotation.SetupWithManager(mgr, o.LogCertSecret); err != nil {
		klog.Errorf("unable to setup cert rotation reconciler: %v", err)
		return err
	}
	cleanGarbageFinalizer := gc.NewCleanGarbageFinalizer(kubeClient)

	go func() {
		<-mgr.Elected()
		go clusterInformers.Start(ctx.Done())
		go kubeInfomers.Start(ctx.Done())

		go clusterSetViewCache.Run(5 * time.Second)
		go clusterSetAdminCache.Run(5 * time.Second)
		go clusterrolebindingSync.Run(5 * time.Second)
		go rolebindingSync.Run(5 * time.Second)

		go cleanGarbageFinalizer.Run(ctx.Done())

		if o.EnableAddonDeploy {
			go addonMgr.Start(ctx)
		}
	}()

	// Start manager
	if err := mgr.Start(ctx); err != nil {
		klog.Errorf("Controller-runtime manager exited non-zero, %v", err)
		return err
	}

	return nil
}
