package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Needed for misc auth.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	actionctrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/action"
	viewctrl "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/view"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = actionv1beta1.AddToScheme(scheme)
	_ = viewv1beta1.AddToScheme(scheme)
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")

	var spokeKubeconfig string
	flag.StringVar(&spokeKubeconfig, "spoke-kubeconfig", "", "The kubeconfig to connect to spoke cluster to apply resources")

	var clusterName string
	flag.StringVar(&clusterName, "cluster-name", "", "The name of the cluster.")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", spokeKubeconfig)
	if err != nil {
		setupLog.Error(err, "Unable to get spoke kube config.")
		os.Exit(1)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "Unable to create spoke dynamic client.")
		os.Exit(1)
	}

	startManager(metricsAddr, clusterName, dynamicClient)
}

func startManager(metricsAddr, clusterName string, dynamicClient dynamic.Interface) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Namespace:          clusterName,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Add controller into manager
	actionReconciler := &actionctrl.ActionReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("ClusterAction"),
		Scheme:             mgr.GetScheme(),
		SpokeDynamicClient: dynamicClient,
	}
	viewReconciler := &viewctrl.SpokeViewReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("SpokeView"),
		Scheme:             mgr.GetScheme(),
		SpokeDynamicClient: dynamicClient,
	}

	if err = actionReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterAction")
		os.Exit(1)
	}

	if err = viewReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SpokeView")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
