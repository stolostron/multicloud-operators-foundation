// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package genericoptions

import (
	hcmclientset "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	clusterclientset "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/componentcontrol"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// GenericOptions is the generic option for operator
type GenericOptions struct {
	KubeConfigFile          string
	BootstrapSecret         string
	ComponentControlOptions *componentcontrol.ControlOptions
}

// GenericConfig is the common config for operator
type GenericConfig struct {
	Hcmclient        hcmclientset.Interface
	Kubeclient       kubernetes.Interface
	ClusterClient    clusterclientset.Interface
	DynamicClient    dynamic.Interface
	ClusterCfg       *restclient.Config
	BootstrapSecret  *corev1.Secret
	ComponentControl *componentcontrol.Controller
}

// NewGenericOptions creates a new GenericOptions object with default values.
func NewGenericOptions() *GenericOptions {
	s := GenericOptions{
		KubeConfigFile:          "",
		BootstrapSecret:         "",
		ComponentControlOptions: componentcontrol.NewControlOptions(),
	}

	return &s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *GenericOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.KubeConfigFile, "local-config-file", "",
		"Klusterlet configuration file to connect to local api-server")
	fs.StringVar(&s.BootstrapSecret, "bootstrap-secret", "",
		"Bootstrap secret in the format of namespace/name")
	s.ComponentControlOptions.AddFlags(fs)
}

// BuildConfig build client and configuration
func (s *GenericOptions) BuildConfig() (*GenericConfig, error) {
	clusterCfg, err := clientcmd.BuildConfigFromFlags("", s.KubeConfigFile)
	if err != nil {
		return nil, err
	}
	kubeclient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return nil, err
	}
	hcmclient, err := hcmclientset.NewForConfig(clusterCfg)
	if err != nil {
		return nil, err
	}
	clusterclient, err := clusterclientset.NewForConfig(clusterCfg)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		return nil, err
	}
	secNamespace, secName, err := cache.SplitMetaNamespaceKey(s.BootstrapSecret)
	if err != nil {
		return nil, err
	}
	btsec, err := kubeclient.CoreV1().Secrets(secNamespace).Get(secName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &GenericConfig{
		Kubeclient:       kubeclient,
		Hcmclient:        hcmclient,
		DynamicClient:    dynamicClient,
		ClusterCfg:       clusterCfg,
		ClusterClient:    clusterclient,
		ComponentControl: s.ComponentControlOptions.ComponentControl(kubeclient),
		BootstrapSecret:  btsec,
	}, nil
}
