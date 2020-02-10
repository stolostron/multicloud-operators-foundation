// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package options

import (
	"sync"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"github.com/spf13/pflag"
)

// SvcRegistryOptions represents the service registry options
type SvcRegistryOptions struct {
	DNSSuffix                     string
	DNSConfigmap                  string
	ClusterProxyIP                string
	ClusterName                   string
	ClusterNamespace              string
	ClusterZone                   string
	ClusterRegion                 string
	ClusterAPIServerConfigFile    string
	HubClusterAPIServerConfigFile string
	SyncPeriod                    int
	EnabledPlugins                []string
	IstioPluginOptions            *plugin.IstioPluginOptions
}

var once sync.Once
var pluginOptions *SvcRegistryOptions

// GetSvcRegistryOptions returns a singleton GetSvcRegistryOptions object
func GetSvcRegistryOptions() *SvcRegistryOptions {
	once.Do(func() {
		pluginOptions = &SvcRegistryOptions{
			DNSSuffix:                     "mcm.svc",
			DNSConfigmap:                  "",
			ClusterProxyIP:                "",
			ClusterName:                   "default",
			ClusterNamespace:              "default",
			ClusterZone:                   "",
			ClusterRegion:                 "",
			ClusterAPIServerConfigFile:    "",
			HubClusterAPIServerConfigFile: "",
			SyncPeriod:                    5,
			EnabledPlugins:                []string{"kube-service"},
			IstioPluginOptions:            plugin.NewIstioPluginOptions(),
		}
	})
	return pluginOptions
}

// AddFlags adds the command flags to KubeservicePluginOptions.
func (o *SvcRegistryOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&o.DNSSuffix,
		"dns-suffix",
		o.DNSSuffix,
		"The suffix of the registry DNS name",
	)
	fs.StringVar(
		&o.DNSConfigmap,
		"dns-configmap",
		o.DNSConfigmap,
		"The CoreDNS configmap",
	)
	fs.StringVar(
		&o.ClusterAPIServerConfigFile,
		"member-cluster-kubeconfig",
		o.ClusterAPIServerConfigFile,
		"Kubernetes configuration file of member cluster",
	)
	fs.StringVar(
		&o.ClusterProxyIP,
		"member-cluster-proxy-ip",
		o.ClusterProxyIP,
		"Proxy ip of member cluster",
	)
	fs.StringVar(
		&o.ClusterName,
		"member-cluster-name",
		o.ClusterName,
		"Name of member cluster",
	)
	fs.StringVar(
		&o.ClusterNamespace,
		"member-cluster-namespace",
		o.ClusterNamespace,
		"Namespace of member cluster",
	)
	fs.StringVar(
		&o.ClusterZone,
		"member-cluster-zone",
		o.ClusterZone,
		"Zone of member cluster",
	)
	fs.StringVar(
		&o.ClusterRegion,
		"member-cluster-region",
		o.ClusterRegion,
		"Region of member cluster",
	)
	fs.StringVar(
		&o.HubClusterAPIServerConfigFile,
		"hub-cluster-kubeconfig",
		o.HubClusterAPIServerConfigFile,
		"Kubernetes configuration file of hub cluster",
	)
	fs.IntVar(
		&o.SyncPeriod,
		"sync-period",
		o.SyncPeriod,
		"Seconds, the period to sync up endpoints and services between the hub and member cluster",
	)
	fs.StringSliceVar(
		&o.EnabledPlugins,
		"plugins",
		o.EnabledPlugins,
		"Comma-separated list of enabled plugins, supported plugins: kube-service, kube-ingress and istio",
	)

	o.IstioPluginOptions.AddFlags(fs)
}
