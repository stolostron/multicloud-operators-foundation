// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"github.com/spf13/pflag"
)

type AgentOptions struct {
	LeaseDurationSeconds            int
	MetricsAddr                     string
	KubeConfig                      string
	HubKubeConfig                   string
	ManagedKubeConfig               string
	ClusterName                     string
	AgentName                       string
	EnableLeaderElection            bool
	EnableImpersonation             bool
	EnableSyncLabelsToClusterClaims bool
	EnableNodeCapacity              bool
	ComponentNamespace              string
	QPS                             float32
	Burst                           int
}

func NewAgentOptions() *AgentOptions {
	return &AgentOptions{
		LeaseDurationSeconds:            60,
		MetricsAddr:                     ":8080",
		KubeConfig:                      "",
		HubKubeConfig:                   "/var/run/hub/kubeconfig",
		ClusterName:                     "",
		AgentName:                       "klusterlet-addon-workmgr",
		EnableLeaderElection:            false,
		EnableImpersonation:             false,
		EnableSyncLabelsToClusterClaims: true,
		EnableNodeCapacity:              true,
		QPS:                             50,
		Burst:                           100,
	}
}

func (o *AgentOptions) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.LeaseDurationSeconds, "lease-duration", o.LeaseDurationSeconds, "The lease duration in seconds, default 60 sec.")
	fs.StringVar(&o.MetricsAddr, "metrics-addr", o.MetricsAddr, "The address the metric endpoint binds to.")
	fs.StringVar(&o.KubeConfig, "kubeconfig", o.KubeConfig,
		"The kubeconfig file of the managed cluster")
	fs.StringVar(&o.HubKubeConfig, "hub-kubeconfig", o.HubKubeConfig,
		"The kubeconfig file of the hub cluster")
	fs.StringVar(&o.ManagedKubeConfig, "managed-kubeconfig", o.ManagedKubeConfig,
		"The kubeconfig file of the managed cluster. "+
			"If this is not set, will use '--kubeconfig' to build client to connect to the managed cluster.")
	fs.StringVar(&o.ClusterName, "cluster-name", o.ClusterName, "The name of the managed cluster.")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection,
		"This flag is deprecated, you should not use this flag any more")
	fs.BoolVar(&o.EnableImpersonation, "enable-impersonation", o.EnableImpersonation, "Enable impersonation.")
	fs.BoolVar(&o.EnableSyncLabelsToClusterClaims, "enable-sync-labels-to-clusterclaims", o.EnableSyncLabelsToClusterClaims, "Enable to create clusterclaims on the managed cluster")
	fs.BoolVar(&o.EnableNodeCapacity, "enable-node-capacity", o.EnableNodeCapacity, "Enable node capacity.")
	fs.StringVar(&o.ComponentNamespace, "", o.ComponentNamespace, ""+
		"Namespace of the agent running If not set, use the value in /var/run/secrets/kubernetes.io/serviceaccount/namespace")
	fs.Float32Var(&o.QPS, "max-qps", o.QPS,
		"Maximum QPS to the local server.")
	fs.IntVar(&o.Burst, "max-burst", o.Burst,
		"Maximum burst for throttle.")
}
