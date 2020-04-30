package options

import (
	"flag"
)

type AgentOptions struct {
	MetricsAddr          string
	SpokeKubeConfig      string
	HubKubeConfig        string
	ClusterName          string
	EnableLeaderElection bool
	EnableImpersonation  bool
}

func NewAgentOptions() *AgentOptions {
	return &AgentOptions{
		MetricsAddr:          ":8080",
		SpokeKubeConfig:      "",
		HubKubeConfig:        "/var/run/hub/kubeconfig",
		ClusterName:          "",
		EnableLeaderElection: false,
		EnableImpersonation:  false,
	}
}

func (o *AgentOptions) AddFlags() {
	flag.StringVar(&o.MetricsAddr, "metrics-addr", o.MetricsAddr, "The address the metric endpoint binds to.")
	flag.StringVar(&o.SpokeKubeConfig, "spoke-kubeconfig", o.SpokeKubeConfig,
		"The kubeconfig to connect to spoke cluster to apply resources.")
	flag.StringVar(&o.HubKubeConfig, "hub-kubeconfig", o.HubKubeConfig,
		"The kubeconfig to connect to hub cluster to watch resources.")
	flag.StringVar(&o.ClusterName, "cluster-name", o.ClusterName, "The name of the cluster.")
	flag.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&o.EnableImpersonation, "enable-impersonation", o.EnableImpersonation, "Enable impersonation.")

	flag.Parse()
}
