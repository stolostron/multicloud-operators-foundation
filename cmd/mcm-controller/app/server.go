// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/mcm-controller/app/options"
	clientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	clusterclientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/apiserverreloader"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/clusterbootstrap/bootstrap"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/clusterbootstrap/rbac"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/controller/cluster"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/controller/gc"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/controller/resourceview"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/controller/serviceregistry"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/controller/workset"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/signals"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/leaderelection"
	"k8s.io/client-go/dynamic"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	componentKlusterlet = "controller"
)

// NewControllerCommand creates a *cobra.Command object with default parameters
func NewControllerCommand() *cobra.Command {
	s := options.NewControllerRunOptions()
	cmd := &cobra.Command{
		Use:  componentKlusterlet,
		Long: ``,
		Run: func(cmd *cobra.Command, args []string) {
			stopCh := signals.SetupSignalHandler()
			if err := Run(s, stopCh); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	s.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the specified klusterlet.  It only returns if stopCh is closed
// or one of the ports cannot be listened on initially.
func Run(s *options.ControllerRunOptions, stopCh <-chan struct{}) error {
	clusterCfg, err := clientcmd.BuildConfigFromFlags("", s.APIServerConfigFile)
	if err != nil {
		return err
	}
	kubeClient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	run := func(stopChan <-chan struct{}) error {
		err = RunController(s, stopChan)
		if err != nil {
			klog.Fatalf("Error run controller: %s", err.Error())
		}

		return nil
	}

	if err := leaderelection.Run(s.LeaderElect, kubeClient, "kube-system", "mcm-hub-controller", stopCh, run); err != nil {
		klog.Fatalf("Error leaderelection run RunOperator: %s", err.Error())
	}

	return nil
}

// RunController start a hcm controller server
func RunController(s *options.ControllerRunOptions, stopCh <-chan struct{}) error {
	hcmCfg, err := clientcmd.BuildConfigFromFlags("", s.APIServerConfigFile)
	if err != nil {
		klog.Fatalf("Error building config to connect to api: %s", err.Error())
	}

	// Configure qps and maxburst
	hcmCfg.QPS = s.QPS
	hcmCfg.Burst = s.Burst

	hcmClient := clientset.NewForConfigOrDie(hcmCfg)
	_, err = hcmClient.ServerVersion()
	if err != nil {
		klog.Fatalf("Failed to connect to hcm apiserver: %s", err.Error())
	}
	klog.Info("Successful initial request to the hcm apiserver")

	clusterClient := clusterclientset.NewForConfigOrDie(hcmCfg)
	dynamicClient := dynamic.NewForConfigOrDie(hcmCfg)

	// TODO add leader election of controller

	informerFactory := informers.NewSharedInformerFactory(hcmClient, time.Minute*10)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, time.Minute*10)

	var kubeclientset kubernetes.Interface
	var kubeInformerFactory kubeinformers.SharedInformerFactory

	if s.EnableRBAC || s.EnableServiceRegistry || s.EnableBootstrap {
		kubeclientset = kubernetes.NewForConfigOrDie(hcmCfg)
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeclientset, time.Minute*10)
	}

	if s.EnableServiceRegistry {
		srController := serviceregistry.NewServiceRegistryController(
			kubeclientset, kubeInformerFactory, clusterInformerFactory, stopCh)
		go srController.Run()
	}

	if s.EnableBootstrap {
		bootstrapController := bootstrap.NewBootstrapController(kubeclientset, hcmClient,
			kubeInformerFactory, informerFactory, clusterInformerFactory, true, stopCh)
		go bootstrapController.Run()

		rbacController := rbac.NewClusterRBACController(kubeclientset, clusterInformerFactory, stopCh)
		go rbacController.Run()

		reloader := apiserverreloader.NewReloader(kubeclientset, stopCh)
		go reloader.Run()
	}

	// Start default controllers
	clusterController := cluster.NewClusterController(
		hcmClient, informerFactory, clusterClient, clusterInformerFactory, s.HealthCheckInterval, stopCh)
	go clusterController.Run()

	worksetController := workset.NewWorkSetController(
		hcmClient, kubeclientset, informerFactory, clusterInformerFactory, s.EnableRBAC, stopCh)
	go worksetController.Run()

	viewController := resourceview.NewResourceViewController(
		hcmClient, kubeclientset, informerFactory, clusterInformerFactory, s.EnableRBAC, stopCh)
	go viewController.Run()

	garbageCollectorController := gc.NewGarbageCollectorController(
		dynamicClient, clusterInformerFactory, informerFactory, s.GarbageCollectorPeriod, stopCh)
	go garbageCollectorController.Run()

	go informerFactory.Start(stopCh)
	go clusterInformerFactory.Start(stopCh)
	if kubeInformerFactory != nil {
		go kubeInformerFactory.Start(stopCh)
	}

	return nil
}
