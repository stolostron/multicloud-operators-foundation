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
	AgentAddress                    string
	AgentName                       string
	Address                         string
	CertDir                         string
	TLSCertFile                     string
	TLSPrivateKeyFile               string
	ClientCAFile                    string
	Port                            int
	AgentPort                       int
	EnableLeaderElection            bool
	EnableImpersonation             bool
	EnableSyncLabelsToClusterClaims bool
	EnableNodeCapacity              bool
	InSecure                        bool
	DisableLoggingInfoSyncer        bool
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
		DisableLoggingInfoSyncer:        false,
		ClusterName:                     "",
		AgentName:                       "klusterlet-addon-workmgr",
		EnableLeaderElection:            false,
		EnableImpersonation:             false,
		EnableSyncLabelsToClusterClaims: true,
		EnableNodeCapacity:              true,
		Address:                         "0.0.0.0",
		Port:                            4443,
		AgentPort:                       443,
		CertDir:                         "/tmp/acm/cert",
		InSecure:                        false,
		TLSCertFile:                     "",
		TLSPrivateKeyFile:               "",
		ClientCAFile:                    "",
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
	fs.BoolVar(&o.DisableLoggingInfoSyncer, "disable-logging-syncer", o.DisableLoggingInfoSyncer, "Disable logging syncer, is false by default.")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection,
		"This flag is deprecated, you should not use this flag any more")
	fs.BoolVar(&o.EnableImpersonation, "enable-impersonation", o.EnableImpersonation, "Enable impersonation.")
	fs.BoolVar(&o.EnableSyncLabelsToClusterClaims, "enable-sync-labels-to-clusterclaims", o.EnableSyncLabelsToClusterClaims, "Enable to create clusterclaims on the managed cluster")
	fs.BoolVar(&o.EnableNodeCapacity, "enable-node-capacity", o.EnableNodeCapacity, "Enable node capacity.")
	fs.IntVar(&o.AgentPort, "agent-port", o.AgentPort, ""+
		"Port that is agent service port for hub cluster to access")
	fs.StringVar(&o.AgentAddress, "agent-address", o.AgentAddress,
		"Address that is agent service address for hub cluster to access, this must be an IP or resolvable fqdn")
	fs.StringVar(&o.AgentName, "agent-name", o.AgentName,
		"agent resource name,default is the name of deployment ")
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
	fs.StringVar(&o.ComponentNamespace, "", o.ComponentNamespace, ""+
		"Namespace of the agent running If not set, use the value in /var/run/secrets/kubernetes.io/serviceaccount/namespace")
	fs.Float32Var(&o.QPS, "max-qps", o.QPS,
		"Maximum QPS to the local server.")
	fs.IntVar(&o.Burst, "max-burst", o.Burst,
		"Maximum burst for throttle.")
}
