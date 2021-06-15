// Copyright (c) 2020 Red Hat, Inc.

package main

import (
	"context"
	"os"
	"time"

	clusterclientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/agent/app"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/agent/app/options"
	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	actionctrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/action"
	clusterclaimctl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	clusterinfoctl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterinfo"
	leasectrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/lease"
	viewctrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/view"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Needed for misc auth.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
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

	stopCh := signals.SetupSignalHandler()
	startManager(o, stopCh)
}

func startManager(o *options.AgentOptions, stopCh <-chan struct{}) {
	hubConfig, err := clientcmd.BuildConfigFromFlags("", o.HubKubeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to get hub kube config.")
		os.Exit(1)
	}
	managedClusterConfig, err := clientcmd.BuildConfigFromFlags("", o.KubeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to get managed cluster kube config.")
		os.Exit(1)
	}
	managedClusterConfig.QPS = o.QPS
	managedClusterConfig.Burst = o.Burst

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
	routeV1Client, err := routev1.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "New route client config error:")
	}

	managedClusterClient, err := kubernetes.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster clientset.")
		os.Exit(1)
	}

	managedClusterClusterClient, err := clusterclientset.NewForConfig(managedClusterConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create managed cluster cluster clientset.")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(hubConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: o.MetricsAddr,
		Namespace:          o.ClusterName,
		Logger:             ctrlruntimelog.NullLogger{},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	go app.ServeHealthProbes(stopCh, ":8000")

	leaseReconciler := leasectrl.LeaseReconciler{
		KubeClient:           managedClusterKubeClient,
		LeaseName:            AddonName,
		LeaseNamespace:       o.ClusterName,
		LeaseDurationSeconds: int32(o.LeaseDurationSeconds),
		HubKubeConfigPath:    o.HubKubeConfig,
	}

	go wait.JitterUntilWithContext(context.TODO(), leaseReconciler.Reconcile, time.Duration(o.LeaseDurationSeconds)*time.Second, leaseUpdateJitterFactor, true)

	run := func(ctx context.Context) {
		// run agent server
		agent, err := app.AgentServerRun(o, managedClusterClient)
		if err != nil {
			setupLog.Error(err, "unable to run agent server")
			os.Exit(1)
		}

		// run mapper
		discoveryClient := cacheddiscovery.NewMemCacheClient(managedClusterClient.Discovery())
		mapper := restutils.NewMapper(discoveryClient, stopCh)
		mapper.Run()

		// Add controller into manager
		actionReconciler := actionctrl.NewActionReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("ManagedClusterAction"),
			mgr.GetScheme(),
			managedClusterDynamicClient,
			restutils.NewKubeControl(mapper, managedClusterConfig),
			o.EnableImpersonation,
		)
		viewReconciler := &viewctrl.ViewReconciler{
			Client:                      mgr.GetClient(),
			Log:                         ctrl.Log.WithName("controllers").WithName("ManagedClusterView"),
			Scheme:                      mgr.GetScheme(),
			ManagedClusterDynamicClient: managedClusterDynamicClient,
			Mapper:                      mapper,
		}

		clusterInfoReconciler := clusterinfoctl.ClusterInfoReconciler{
			Client:                      mgr.GetClient(),
			Log:                         ctrl.Log.WithName("controllers").WithName("ManagedClusterInfo"),
			Scheme:                      mgr.GetScheme(),
			KubeClient:                  managedClusterKubeClient,
			ManagedClusterDynamicClient: managedClusterDynamicClient,
			ClusterName:                 o.ClusterName,
			AgentRoute:                  o.AgentRoute,
			AgentAddress:                o.AgentAddress,
			AgentIngress:                o.AgentIngress,
			AgentPort:                   int32(o.AgentPort),
			RouteV1Client:               routeV1Client,
			Agent:                       agent,
			AgentService:                o.AgentService,
		}

		clusterClaimReconciler := clusterclaimctl.ClusterClaimReconciler{
			Log:               ctrl.Log.WithName("controllers").WithName("ManagedClusterInfo"),
			ClusterClient:     managedClusterClusterClient,
			ListClusterClaims: clusterInfoReconciler.GetClusterClaims,
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

		setupLog.Info("starting manager")
		if err := mgr.Start(stopCh); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}

	if !o.EnableLeaderElection {
		run(context.TODO())
		panic("unreachable")
	}

	lec, err := app.NewLeaderElection(scheme, managedClusterClient, run)
	if err != nil {
		setupLog.Error(err, "cannot create leader election")
		os.Exit(1)
	}

	leaderelection.RunOrDie(context.TODO(), *lec)
	panic("unreachable")
}
