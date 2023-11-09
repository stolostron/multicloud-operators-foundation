// Copyright (c) 2020 Red Hat, Inc.

package main

import (
	"context"
	"os"
	"time"

	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	viewv1beta1 "github.com/stolostron/cluster-lifecycle-api/view/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/cmd/agent/app"
	"github.com/stolostron/multicloud-operators-foundation/cmd/agent/app/options"
	actionctrl "github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/action"
	clusterclaimctl "github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	clusterinfoctl "github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterinfo"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/nodecollector"
	viewctrl "github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/view"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	restutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/rest"
	addonutils "open-cluster-management.io/addon-framework/pkg/utils"

	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	openshiftoauthclientset "github.com/openshift/client-go/oauth/clientset/versioned"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	"open-cluster-management.io/addon-framework/pkg/lease"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Needed for misc auth.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	AddonName               = "work-manager"
	leaseUpdateJitterFactor = 0.25
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = actionv1beta1.AddToScheme(scheme)
	_ = viewv1beta1.AddToScheme(scheme)
	_ = clusterv1beta1.AddToScheme(scheme)
	_ = clusterv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	o := options.NewAgentOptions()
	o.AddFlags(pflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()
	ctx := signals.SetupSignalHandler()
	startManager(o, ctx)
}

func startManager(o *options.AgentOptions, ctx context.Context) {
	hubConfig, err := clientcmd.BuildConfigFromFlags("", o.HubKubeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to get hub kube config.")
		os.Exit(1)
	}

	// create management kube config
	managementKubeConfig, err := clientcmd.BuildConfigFromFlags("", o.KubeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to get management cluster kube config.")
		os.Exit(1)
	}
	managementKubeConfig.QPS = o.QPS
	managementKubeConfig.Burst = o.Burst

	managementClusterKubeClient, err := kubernetes.NewForConfig(managementKubeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create management cluster kube client.")
		os.Exit(1)
	}

	// load managed client config, the work manager agent may not running in the managed cluster.
	managedClusterConfig := managementKubeConfig
	if o.ManagedKubeConfig != "" {
		managedClusterConfig, err = clientcmd.BuildConfigFromFlags("", o.ManagedKubeConfig)
		if err != nil {
			setupLog.Error(err, "Unable to get managed cluster kube config.")
			os.Exit(1)
		}
		managedClusterConfig.QPS = o.QPS
		managedClusterConfig.Burst = o.Burst
	}

	managedClusterDynamicClient, err := dynamic.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster dynamic client.")
		os.Exit(1)
	}
	managedClusterKubeClient, err := kubernetes.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster kube client.")
		os.Exit(1)
	}
	routeV1Client, err := routev1.NewForConfig(managementKubeConfig)
	if err != nil {
		setupLog.Error(err, "New route client config error:")
	}

	openshiftClient, err := openshiftclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster openshift config clientset.")
		os.Exit(1)
	}

	osOauthClient, err := openshiftoauthclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster openshift oauth clientset.")
		os.Exit(1)
	}

	managedClusterClusterClient, err := clusterclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster cluster clientset.")
		os.Exit(1)
	}

	restMapper, err := apiutil.NewDynamicRESTMapper(managedClusterConfig, apiutil.WithLazyDiscovery)
	if err != nil {
		setupLog.Error(err, "Unable to create restmapper.")
		os.Exit(1)
	}

	componentNamespace := o.ComponentNamespace
	if len(componentNamespace) == 0 {
		componentNamespace, err = utils.GetComponentNamespace()
		if err != nil {
			setupLog.Error(err, "Failed to get pod namespace")
			os.Exit(1)
		}
	}

	cc, err := addonutils.NewConfigChecker("work manager", o.HubKubeConfig)
	if err != nil {
		setupLog.Error(err, "unable to setup a configChecker")
		os.Exit(1)
	}

	// run healthProbes server before newManager, because it may take a long time to discover the APIs of Hub cluster in newManager.
	// the agent pods will be restarted if failed to check healthz/liveness in 30s.
	go app.ServeHealthProbes(ctx.Done(), ":8000", cc.Check)

	mgr, err := ctrl.NewManager(hubConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: o.MetricsAddr,
		Namespace:          o.ClusterName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	run := func(ctx context.Context) {
		// run agent server
		agent, err := app.AgentServerRun(o, managedClusterKubeClient)
		if err != nil {
			setupLog.Error(err, "unable to run agent server")
			os.Exit(1)
		}

		kubeInformerFactory := informers.NewSharedInformerFactory(managedClusterKubeClient, 10*time.Minute)
		clusterInformerFactory := clusterinformers.NewSharedInformerFactory(managedClusterClusterClient, 10*time.Minute)

		resourceCollector := nodecollector.NewCollector(
			kubeInformerFactory.Core().V1().Nodes(),
			managedClusterKubeClient,
			mgr.GetClient(),
			o.ClusterName,
			componentNamespace,
			o.EnableNodeCapacity)
		go resourceCollector.Start(ctx)

		leaseUpdater := lease.NewLeaseUpdater(managementClusterKubeClient, AddonName, componentNamespace).
			WithHubLeaseConfig(hubConfig, o.ClusterName)
		go leaseUpdater.Start(ctx)

		// Add controller into manager
		actionReconciler := actionctrl.NewActionReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("ManagedClusterAction"),
			mgr.GetScheme(),
			managedClusterDynamicClient,
			restutils.NewKubeControl(restMapper, managedClusterConfig),
			o.EnableImpersonation,
		)
		viewReconciler := &viewctrl.ViewReconciler{
			Client:                      mgr.GetClient(),
			Log:                         ctrl.Log.WithName("controllers").WithName("ManagedClusterView"),
			Scheme:                      mgr.GetScheme(),
			ManagedClusterDynamicClient: managedClusterDynamicClient,
			Mapper:                      restMapper,
		}

		clusterInfoReconciler := clusterinfoctl.ClusterInfoReconciler{
			Client:         mgr.GetClient(),
			Log:            ctrl.Log.WithName("controllers").WithName("ManagedClusterInfo"),
			Scheme:         mgr.GetScheme(),
			NodeLister:     kubeInformerFactory.Core().V1().Nodes().Lister(),
			NodeInformer:   kubeInformerFactory.Core().V1().Nodes(),
			ClaimInformer:  clusterInformerFactory.Cluster().V1alpha1().ClusterClaims(),
			ClaimLister:    clusterInformerFactory.Cluster().V1alpha1().ClusterClaims().Lister(),
			KubeClient:     managedClusterKubeClient,
			ClusterName:    o.ClusterName,
			AgentRoute:     o.AgentRoute,
			AgentAddress:   o.AgentAddress,
			AgentIngress:   o.AgentIngress,
			AgentPort:      int32(o.AgentPort),
			RouteV1Client:  routeV1Client,
			ConfigV1Client: openshiftClient,
			Agent:          agent,
			AgentService:   o.AgentService,
		}
		clusterClaimer := clusterclaimctl.ClusterClaimer{
			ClusterName:                     o.ClusterName,
			HubClient:                       mgr.GetClient(),
			KubeClient:                      managedClusterKubeClient,
			ConfigV1Client:                  openshiftClient,
			OauthV1Client:                   osOauthClient,
			Mapper:                          restMapper,
			EnableSyncLabelsToClusterClaims: o.EnableSyncLabelsToClusterClaims,
		}

		clusterClaimReconciler := clusterclaimctl.ClusterClaimReconciler{
			Log:               ctrl.Log.WithName("controllers").WithName("ManagedClusterInfo"),
			ClusterClient:     managedClusterClusterClient,
			ClusterInformers:  clusterInformerFactory.Cluster().V1alpha1().ClusterClaims(),
			ListClusterClaims: clusterClaimer.List,
		}

		if err = actionReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ManagedClusterAction")
			os.Exit(1)
		}

		if err = viewReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ManagedClusterView")
			os.Exit(1)
		}

		if err = clusterInfoReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ManagedClusterInfo")
			os.Exit(1)
		}

		if err = clusterClaimReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ClusterClaim")
			os.Exit(1)
		}

		go kubeInformerFactory.Start(ctx.Done())
		go clusterInformerFactory.Start(ctx.Done())

		setupLog.Info("starting manager")
		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}
	run(context.TODO())
	
	panic("unreachable")
}
