package main

import (
	"context"
	"os"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-agent/app"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-agent/app/options"
	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	actionctrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/action"
	clusterinfoctl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterinfo"
	viewctrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/view"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Needed for misc auth.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = actionv1beta1.AddToScheme(scheme)
	_ = viewv1beta1.AddToScheme(scheme)
	_ = clusterv1beta1.AddToScheme(scheme)
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
	spokeConfig, err := clientcmd.BuildConfigFromFlags("", o.SpokeKubeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to get spoke kube config.")
		os.Exit(1)
	}
	spokeDynamicClient, err := dynamic.NewForConfig(spokeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create spoke dynamic client.")
		os.Exit(1)
	}
	spokeKubeClient, err := kubernetes.NewForConfig(spokeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create spoke kube client.")
		os.Exit(1)
	}
	routeV1Client, err := routev1.NewForConfig(spokeConfig)
	if err != nil {
		setupLog.Error(err, "New route client config error:")
	}

	spokeClient, err := kubernetes.NewForConfig(spokeConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create spoke clientset.")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(hubConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: o.MetricsAddr,
		Namespace:          o.ClusterName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	go app.ServeHealthProbes(stopCh)

	run := func(ctx context.Context) {
		// run agent server
		klusterlet, err := app.AgentServerRun(o, spokeClient)
		if err != nil {
			setupLog.Error(err, "unable to run agent server")
			os.Exit(1)
		}

		// run mapper
		discoveryClient := cacheddiscovery.NewMemCacheClient(spokeClient.Discovery())
		mapper := restutils.NewMapper(discoveryClient, stopCh)
		mapper.Run()

		// Add controller into manager
		actionReconciler := actionctrl.NewActionReconciler(
			mgr.GetClient(),
			ctrl.Log.WithName("controllers").WithName("ClusterAction"),
			mgr.GetScheme(),
			spokeDynamicClient,
			restutils.NewKubeControl(mapper, spokeConfig),
			o.EnableImpersonation,
		)
		spokeViewReconciler := &viewctrl.SpokeViewReconciler{
			Client:             mgr.GetClient(),
			Log:                ctrl.Log.WithName("controllers").WithName("SpokeView"),
			Scheme:             mgr.GetScheme(),
			SpokeDynamicClient: spokeDynamicClient,
			Mapper:             mapper,
		}

		clusterInfoReconciler := clusterinfoctl.ClusterInfoReconciler{
			Client:             mgr.GetClient(),
			Log:                ctrl.Log.WithName("controllers").WithName("ClusterInfo"),
			Scheme:             mgr.GetScheme(),
			KubeClient:         spokeKubeClient,
			SpokeDynamicClient: spokeDynamicClient,
			KlusterletRoute:    o.KlusterletRoute,
			KlusterletAddress:  o.KlusterletAddress,
			KlusterletIngress:  o.KlusterletIngress,
			KlusterletPort:     int32(o.KlusterletPort),
			RouteV1Client:      routeV1Client,
			Klusterlet:         klusterlet,
		}

		if err = actionReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ClusterAction")
			os.Exit(1)
		}

		if err = spokeViewReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "SpokeView")
			os.Exit(1)
		}

		if err = clusterInfoReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ClusterInfo")
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

	lec, err := app.NewLeaderElection(scheme, spokeClient, run)
	if err != nil {
		setupLog.Error(err, "cannot create leader election")
		os.Exit(1)
	}

	leaderelection.RunOrDie(context.TODO(), *lec)
	panic("unreachable")
}
