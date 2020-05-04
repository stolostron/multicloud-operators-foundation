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
	KlusterletPort       int64
	KlusterletAddress    string
	KlusterletIngress    string
	KlusterletRoute      string
	KlusterletService    string
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
	flag.Int64Var(&o.KlusterletPort, "klusterlet-port", o.KlusterletPort, ""+
		"Port that expose klusterlet service for hub cluster to access")
	flag.StringVar(&o.KlusterletAddress, "klusterlet-address", o.KlusterletAddress,
		"Address that expose klusterlet service for hub cluster to access, this must be an IP or resolvable fqdn")
	flag.StringVar(&o.KlusterletIngress, "klusterlet-ingress", o.KlusterletIngress, ""+
		"Klusterlet ingress created in managed cluster, in the format of namespace/name")
	flag.StringVar(&o.KlusterletRoute, "klusterlet-route", o.KlusterletRoute, ""+
		"Klusterlet route created in managed cluster, in the format of namespace/name")
	flag.StringVar(&o.KlusterletService, "klusterlet-service", o.KlusterletService, ""+
		"Klusterlet service created in managed cluster, in the format of namespace/name")
	flag.Parse()
}
