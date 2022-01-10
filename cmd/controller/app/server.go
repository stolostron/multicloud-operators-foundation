// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"io/ioutil"
	"path"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alaph1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	hiveinternalv1alpha1 "github.com/openshift/hive/pkg/apis/hiveinternal/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app/options"
	actionv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	clusterinfov1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	inventoryv1alpha1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/autodetect"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterinfo"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterrbac"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterrole"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clusterrolebinding"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/clustersetmapper"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/clusterset/syncclusterrolebinding"
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

func Run(o *options.ControllerRunOptions, stopCh <-chan struct{}) error {
	clustersetToSubject := helpers.NewClustersetSubjectsMapper()
	clusterSetMapper := helpers.NewClusterSetMapper()
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", o.KubeConfig)
	if err != nil {
		klog.Errorf("unable to get kube config: %v", err)
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.Errorf("unable to create kube client: %v", err)
		return err
	}

	caData, err := GetAgentCA(o.CAFile)
	if err != nil {
		klog.Warningf("unable to get foundation agent server CA file: %v", err)
	}

	kubeConfig.QPS = o.QPS
	kubeConfig.Burst = o.Burst

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

	// Setup reconciler
	if o.EnableInventory {
		if err = inventory.SetupWithManager(mgr); err != nil {
			klog.Errorf("unable to setup inventory reconciler: %v", err)
			return err
		}
	}

	if err = clusterinfo.SetupWithManager(mgr, caData); err != nil {
		klog.Errorf("unable to setup clusterInfo reconciler: %v", err)
		return err
	}

	if err = autodetect.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup auto detect reconciler: %v", err)
		return err
	}

	if err = clusterrolebinding.SetupWithManager(mgr, clustersetToSubject); err != nil {
		klog.Errorf("unable to setup clusterrolebinding reconciler: %v", err)
		return err
	}

	if err = syncclusterrolebinding.SetupWithManager(mgr, clustersetToSubject, clusterSetMapper); err != nil {
		klog.Errorf("unable to setup clusterrolebinding reconciler: %v", err)
		return err
	}

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

	if err = clustersetmapper.SetupWithManager(mgr, clusterSetMapper); err != nil {
		klog.Errorf("unable to setup clustersetmapper reconciler: %v", err)
		return err
	}

	// Start manager
	if err := mgr.Start(stopCh); err != nil {
		klog.Errorf("Controller-runtime manager exited non-zero, %v", err)
		return err
	}

	return nil
}

func GetAgentCA(caFile string) ([]byte, error) {
	pemBlock, err := ioutil.ReadFile(path.Clean(caFile))
	if err != nil {
		return nil, err
	}
	return pemBlock, nil
}
