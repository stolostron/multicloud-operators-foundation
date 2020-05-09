package options

import (
	"github.com/spf13/pflag"
	"k8s.io/component-base/cli/flag"
)

type AgentOptions struct {
	MetricsAddr          string
	SpokeKubeConfig      string
	HubKubeConfig        string
	ClusterName          string
	KlusterletAddress    string
	KlusterletIngress    string
	KlusterletRoute      string
	KlusterletService    string
	Address              string
	CertDir              string
	TLSCertFile          string
	TLSPrivateKeyFile    string
	ClientCAFile         string
	Port                 int
	KlusterletPort       int
	EnableLeaderElection bool
	EnableImpersonation  bool
	InSecure             bool
}

func NewAgentOptions() *AgentOptions {
	return &AgentOptions{
		MetricsAddr:          ":8080",
		SpokeKubeConfig:      "",
		HubKubeConfig:        "/var/run/hub/kubeconfig",
		ClusterName:          "",
		EnableLeaderElection: true,
		EnableImpersonation:  false,
		Address:              "0.0.0.0",
		Port:                 443,
		CertDir:              "/tmp/acm/cert",
		InSecure:             false,
		TLSCertFile:          "",
		TLSPrivateKeyFile:    "",
		ClientCAFile:         "",
	}
}

func (o *AgentOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.MetricsAddr, "metrics-addr", o.MetricsAddr, "The address the metric endpoint binds to.")
	fs.StringVar(&o.SpokeKubeConfig, "spoke-kubeconfig", o.SpokeKubeConfig,
		"The kubeconfig to connect to spoke cluster to apply resources.")
	fs.StringVar(&o.HubKubeConfig, "hub-kubeconfig", o.HubKubeConfig,
		"The kubeconfig to connect to hub cluster to watch resources.")
	fs.StringVar(&o.ClusterName, "cluster-name", o.ClusterName, "The name of the cluster.")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.BoolVar(&o.EnableImpersonation, "enable-impersonation", o.EnableImpersonation, "Enable impersonation.")
	fs.IntVar(&o.KlusterletPort, "klusterlet-port", o.KlusterletPort, ""+
		"Port that expose klusterlet service for hub cluster to access")
	fs.StringVar(&o.KlusterletAddress, "klusterlet-address", o.KlusterletAddress,
		"Address that expose klusterlet service for hub cluster to access, this must be an IP or resolvable fqdn")
	fs.StringVar(&o.KlusterletIngress, "klusterlet-ingress", o.KlusterletIngress, ""+
		"Klusterlet ingress created in managed cluster, in the format of namespace/name")
	fs.StringVar(&o.KlusterletRoute, "klusterlet-route", o.KlusterletRoute, ""+
		"Klusterlet route created in managed cluster, in the format of namespace/name")
	fs.StringVar(&o.KlusterletService, "klusterlet-service", o.KlusterletService, ""+
		"Klusterlet service created in managed cluster, in the format of namespace/name")
	fs.StringVar(&o.Address, "address", o.Address, ""+
		"The IP address for the Kubelet to serve on (set to `0.0.0.0` for all IPv4 interfaces and `::` for all IPv6 interfaces)")
	fs.IntVar(&o.Port, "port", o.Port, "The port for the agent to serve on.")
	fs.StringVar(&o.CertDir, "cert-directory", o.CertDir, "certificate directory")
	fs.BoolVar(&o.InSecure, "insecure", o.InSecure, "If set, agent server run in the in-secure mode")
	fs.StringVar(
		&o.TLSCertFile, "tls-cert-file", o.TLSCertFile,
		"File containing x509 Certificate used for serving HTTPS (with intermediate certs, if any, concatenated after server cert). "+
			"If --tls-cert-file and --tls-private-key-file are not provided, a self-signed certificate and key "+
			"are generated for the public address and saved to the directory passed to --cert-dir.")
	fs.StringVar(
		&o.TLSPrivateKeyFile, "tls-private-key-file", o.TLSPrivateKeyFile,
		"File containing x509 private key matching --tls-cert-file.")
	fs.StringVar(&o.ClientCAFile, "client-ca-file", o.ClientCAFile, ""+
		"If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file "+
		"is authenticated with an identity corresponding to the CommonName of the client certificate.")

	flag.InitFlags()
}
