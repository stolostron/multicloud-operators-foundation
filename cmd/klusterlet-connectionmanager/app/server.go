// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/klusterlet-connectionmanager/app/options"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/genericoptions"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/manager"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/leaderelection"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// Run runs the specified klusterlet operator.  It only returns if stopCh is closed
// or one of the ports cannot be listened on initially.
func Run(s *options.RunOptions, stopCh <-chan struct{}) error {
	genericConfig, err := s.Generic.BuildConfig()
	if err != nil {
		klog.Fatalf("Error build config: %s", err.Error())
	}

	run := func(stopChan <-chan struct{}) error {
		err = RunOperator(s, genericConfig, stopChan)
		if err != nil {
			klog.Fatalf("Error run operator: %s", err.Error())
		}

		return nil
	}

	namespace, _, err := cache.SplitMetaNamespaceKey(s.Generic.ComponentControlOptions.KlusterletSecret)
	if err != nil {
		klog.Fatalf("Error get namespace: %s", err.Error())
	}

	if err := leaderelection.Run(s.LeaderElect, genericConfig.Kubeclient, namespace, "mcm-klusterlet-operator", stopCh, run); err != nil {
		klog.Fatalf("Error leaderelection run RunOperator: %s", err.Error())
	}

	return nil
}

// RunOperator start a klusterlet operator
func RunOperator(s *options.RunOptions, genericConfig *genericoptions.GenericConfig, stopCh <-chan struct{}) error {
	clusterNamespace, clusterName, err := cache.SplitMetaNamespaceKey(s.Cluster)
	if err != nil {
		klog.Fatalf("Error get cluster name: %s", err.Error())
	}

	kloperator := manager.NewOperator(genericConfig, clusterName, clusterNamespace, stopCh)
	go kloperator.Run()

	return nil
}
