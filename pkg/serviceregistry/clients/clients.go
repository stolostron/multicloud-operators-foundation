// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clients

import (
	"sync"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

var loadMemberKubeClientOnce sync.Once
var memberKubeClient *kubernetes.Clientset

// MemberKubeClient returns client in current cluster
func MemberKubeClient() *kubernetes.Clientset {
	loadMemberKubeClientOnce.Do(func() {
		config, err := clientcmd.BuildConfigFromFlags("", options.GetSvcRegistryOptions().ClusterAPIServerConfigFile)
		if err != nil {
			klog.Fatalf("cannot get kubernetes config: %v", err)
		}

		memberKubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			klog.Fatalf("cannot create kubernetes client: %v", err)
		}

		if _, err := memberKubeClient.ServerVersion(); err != nil {
			klog.Fatalf("failed to connect to current cluster: %s", err.Error())
		}

		klog.Info("successfully initialize request to the current cluster")
	})
	return memberKubeClient
}

var loadMemberDynamicKubeClientOnce sync.Once
var memberDynamicKubeClient dynamic.Interface

// MemberDynamicKubeClient returns dynamic client in current cluster
func MemberDynamicKubeClient() dynamic.Interface {
	loadMemberDynamicKubeClientOnce.Do(func() {
		config, err := clientcmd.BuildConfigFromFlags("", options.GetSvcRegistryOptions().ClusterAPIServerConfigFile)
		if err != nil {
			klog.Fatalf("cannot get kubernetes config: %v", err)
		}

		if memberDynamicKubeClient, err = dynamic.NewForConfig(config); err != nil {
			klog.Fatalf("failed to connect to current cluster (dynamic): %s", err.Error())
		}

		klog.Info("successfully initialize request to the current cluster (dynamic)")
	})
	return memberDynamicKubeClient
}

var loadHubKubeClientOnce sync.Once
var hubKubeClient *kubernetes.Clientset

// HubKubeClient returns client in Hub cluster
func HubKubeClient(options *options.SvcRegistryOptions) *kubernetes.Clientset {
	loadHubKubeClientOnce.Do(func() {
		config, err := clientcmd.BuildConfigFromFlags("", options.HubClusterAPIServerConfigFile)
		if err != nil {
			klog.Fatalf("cannot get hub cluster kubernetes config: %v", err)
		}

		hubKubeClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			klog.Fatalf("cannot create hub cluster kubernetes client: %v", err)
		}

		if _, err = hubKubeClient.ServerVersion(); err != nil {
			klog.Fatalf("failed to connect to hub cluster: %s", err.Error())
		}
		klog.Info("successfully initialize request to the hub cluster")
	})
	return hubKubeClient
}
