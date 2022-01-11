// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"context"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/certrotation"

	clusterv1client "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1informers "github.com/open-cluster-management/api/client/cluster/informers/externalversions"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alaph1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app/options"
	actionv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	clusterinfov1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	inventoryv1alpha1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterinfo"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterrbac"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterrole"

	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clusterclaim"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clusterdeployment"
	clustersetmapper "github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clustersetmapper"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/syncrolebinding"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	kubeinformers "k8s.io/client-go/informers"

	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/syncclusterrolebinding"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveinternalv1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/gc"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/inventory"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = inventoryv1alpha1.AddToScheme(scheme)
	_ = hiveinternalv1alpha1.AddToScheme(scheme)
	_ = hivev1.AddToScheme(scheme)
	_ = clusterinfov1beta1.AddToScheme(scheme)
	_ = clusterv1.Install(scheme)
	_ = actionv1beta1.AddToScheme(scheme)
	_ = clusterv1alaph1.Install(scheme)
}

func Run(o *options.ControllerRunOptions, ctx context.Context) error {

	//clusterset to cluster map
	clusterSetClusterMapper := helpers.NewClusterSetMapper()

	//clusterset to namespace resource map, like clusterdeployment, clusterpool, clusterclaim. the map value format is "<ResourceType>/<Namespace>/<Name>"
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

	clusterInformers := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	kubeInfomers := kubeinformers.NewSharedInformerFactory(kubeClient, 10*time.Minute)

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme:                 scheme,
		LeaderElectionID:       "foundation-controller",
		LeaderElection:         o.EnableLeaderElection,
		HealthProbeBindAddress: ":8000",
		Logger:                 ctrlruntimelog.NullLogger{},
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
		clusterInformers.Cluster().V1alpha1().ManagedClusterSets(),
		kubeInfomers.Rbac().V1().ClusterRoles(),
		kubeInfomers.Rbac().V1().ClusterRoleBindings(),
		utils.GetAdminResourceFromClusterRole,
	)
	clusterSetViewCache := cache.NewClusterSetCache(
		clusterInformers.Cluster().V1alpha1().ManagedClusterSets(),
		kubeInfomers.Rbac().V1().ClusterRoles(),
		kubeInfomers.Rbac().V1().ClusterRoleBindings(),
		utils.GetViewResourceFromClusterRole,
	)

	// Setup reconciler
	if o.EnableInventory {
		if err = inventory.SetupWithManager(mgr); err != nil {
			klog.Errorf("unable to setup inventory reconciler: %v", err)
			return err
		}
	}

	if err = clusterinfo.SetupWithManager(mgr, o.LogCertSecret); err != nil {
		klog.Errorf("unable to setup clusterInfo reconciler: %v", err)
		return err
	}

	if err = clustersetmapper.SetupWithManager(mgr, kubeClient, clusterSetClusterMapper, clusterSetNamespaceMapper); err != nil {
		klog.Errorf("unable to setup clustersetmapper reconciler: %v", err)
		return err
	}
	if err = clusterdeployment.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup clustersetmapper reconciler: %v", err)
		return err
	}
	if err = clusterclaim.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup clustersetmapper reconciler: %v", err)
		return err
	}

	clusterrolebindingSync := syncclusterrolebinding.NewReconciler(kubeClient, clusterSetAdminCache.Cache, clusterSetViewCache.Cache, clusterSetClusterMapper)

	rolebindingSync := syncrolebinding.NewReconciler(kubeClient, clusterSetAdminCache.Cache, clusterSetViewCache.Cache, clusterSetClusterMapper, clusterSetNamespaceMapper)

	if o.EnableRBAC {
		if err = clusterrbac.SetupWithManager(mgr, kubeClient); err != nil {
			klog.Errorf("unable to setup clusterrbac reconciler: %v", err)
			return err
		}
	}

	if err = clusterrole.SetupWithManager(mgr, kubeClient); err != nil {
		klog.Errorf("unable to setup clusterrole reconciler: %v", err)
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
	}()

	// Start manager
	if err := mgr.Start(ctx); err != nil {
		klog.Errorf("Controller-runtime manager exited non-zero, %v", err)
		return err
	}

	return nil
}
