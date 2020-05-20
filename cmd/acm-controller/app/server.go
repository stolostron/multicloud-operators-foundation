// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"io/ioutil"

	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/gc"

	"sigs.k8s.io/controller-runtime/pkg/healthz"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterregistryv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/clusterrbac"
	"k8s.io/client-go/kubernetes"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/clusterinfo"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/inventory"
	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"

	inventoryv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"

	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = inventoryv1alpha1.AddToScheme(scheme)
	_ = hivev1.AddToScheme(scheme)
	_ = clusterinfov1beta1.AddToScheme(scheme)

	// TODO: deprecate clusterregistry
	_ = clusterregistryv1alpha1.AddToScheme(scheme)

	_ = clusterv1.Install(scheme)
	_ = actionv1beta1.AddToScheme(scheme)
}

func Run(o *options.ControllerRunOptions, stopCh <-chan struct{}) error {
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

	caData, err := GetKlusterletCA(o.CAFile)
	if err != nil {
		klog.Errorf("unable to get klusterlet CA file: %v", err)
		return err
	}

	kubeConfig.QPS = o.QPS
	kubeConfig.Burst = o.Burst

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme:                 scheme,
		LeaderElectionID:       "acm-controller",
		LeaderElection:         o.EnableLeaderElection,
		HealthProbeBindAddress: ":8000",
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

	if err = clusterrbac.SetupWithManager(mgr, kubeClient); err != nil {
		klog.Errorf("unable to setup clusterInfo reconciler: %v", err)
		return err
	}

	if err = gc.SetupWithManager(mgr); err != nil {
		klog.Errorf("unable to setup gc reconciler: %v", err)
		return err
	}

	// Start manager
	if err := mgr.Start(stopCh); err != nil {
		klog.Errorf("Controller-runtime manager exited non-zero, %v", err)
		return err
	}

	return nil
}

func GetKlusterletCA(caFile string) ([]byte, error) {
	pemBlock, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	return pemBlock, nil
}
