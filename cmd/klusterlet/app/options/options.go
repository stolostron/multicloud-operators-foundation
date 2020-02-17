// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been
// deposited with the U.S. Copyright Office.

package options

import (
	"os"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/drivers"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
)

// KlusterletRunOptions for the klusterlet.
type KlusterletRunOptions struct {
	ClusterName                string
	ClusterNamespace           string
	ClusterLabels              string
	ClusterConfigFile          string
	ControllerConfigFile       string
	Address                    string
	TillerEndpoint             string
	TillerCertFile             string
	TillerKeyFile              string
	TillerCAFile               string
	CertDir                    string
	CertSecret                 string
	HubKubeconfigSecret        string
	MasterAddresses            string
	KlusterletAddress          string
	KlusterletIngress          string
	KlusterletRoute            string
	KlusterletService          string
	HelmReleasePrefix          string
	TLSCertFile                string
	TLSPrivateKeyFile          string
	ClientCAFile               string
	DriverOptions              *drivers.DriverFactoryOptions
	Port                       int32
	KlusterletPort             int32
	InSecure                   bool
	UseBearerTokenForPometheus bool
	EnableImpersonation        bool
}

// NewKlusterletRunOptions creates a new ServerRunOptions object with default values.
func NewKlusterletRunOptions() *KlusterletRunOptions {
	s := KlusterletRunOptions{
		ClusterName:          "mycluster",
		ClusterNamespace:     "default",
		ClusterLabels:        "",
		ClusterConfigFile:    "",
		ControllerConfigFile: "",
		Address:              "0.0.0.0",
		Port:                 443,
		CertDir:              "/tmp/hcm/cert",
		HubKubeconfigSecret:  "kube-system/hub-config-secret",
		TillerEndpoint:       "",
		TillerCertFile:       "",
		TillerKeyFile:        "",
		TillerCAFile:         "",
		CertSecret:           "kube-system/klusterlet-cert",
		MasterAddresses:      "",
		KlusterletPort:       443,
		KlusterletAddress:    "",
		KlusterletService:    "",
		HelmReleasePrefix:    "md",
		KlusterletIngress:    "",
		KlusterletRoute:      "",
		TLSCertFile:          "",
		TLSPrivateKeyFile:    "",
		InSecure:             false,
		ClientCAFile:         "",
		DriverOptions:        drivers.NewDriverFactoryOptions(),
		EnableImpersonation:  false,
	}

	s.ClusterConfigFile = os.Getenv(clientcmd.RecommendedConfigPathEnvVar)

	return &s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *KlusterletRunOptions) AddFlags(fs *pflag.FlagSet) {
	// Cluster configuration
	fs.StringVar(&s.ClusterName, "cluster-name", s.ClusterName, "Cluster name of the managed cluster")
	fs.StringVar(&s.ClusterNamespace, "cluster-namespace", s.ClusterNamespace, "namespace of the cluster")
	fs.StringVar(&s.ClusterLabels, "cluster-labels", s.ClusterLabels, "labels of the cluster")
	fs.StringVar(&s.ClusterConfigFile, "cluster-config-file", s.ClusterConfigFile, ""+
		"Configuration file to work with managed kubernetes cluster")
	fs.StringVar(&s.ControllerConfigFile, "controller-config-file", s.ControllerConfigFile, ""+
		"Configuration file to work to hcm controller")
	fs.StringVar(&s.Address, "address", s.Address, ""+
		"The IP address for the Kubelet to serve on (set to `0.0.0.0` for all IPv4 interfaces and `::` for all IPv6 interfaces)")
	fs.Int32Var(&s.Port, "port", s.Port, "The port for the klusterlet to serve on.")
	fs.Int32Var(&s.KlusterletPort, "klusterlet-port", s.KlusterletPort, ""+
		"Port that expose klusterlet service for hub cluster to access")
	fs.StringVar(&s.KlusterletAddress, "klusterlet-address", s.KlusterletAddress,
		"Address that expose klusterlet service for hub cluster to access, this must be an IP or resolvable fqdn")
	fs.StringVar(&s.KlusterletIngress, "klusterlet-ingress", s.KlusterletIngress, ""+
		"Klusterlet ingress created in managed cluster, in the format of namespace/name")
	fs.StringVar(&s.KlusterletRoute, "klusterlet-route", s.KlusterletRoute, ""+
		"Klusterlet route created in managed cluster, in the format of namespace/name")
	fs.StringVar(&s.KlusterletService, "klusterlet-service", s.KlusterletService, ""+
		"Klusterlet service created in managed cluster, in the format of namespace/name")

	// Authentication
	fs.StringVar(&s.CertDir, "cert-directory", s.CertDir, "certificate directory")
	fs.StringVar(&s.CertSecret, "client-cert-secret", s.CertSecret, "client certificate stored in the target cluster")
	fs.StringVar(
		&s.HubKubeconfigSecret, "hub-config-secret", s.HubKubeconfigSecret, "hub kubeconfig file stored in the target cluster")
	fs.StringVar(
		&s.TLSCertFile, "tls-cert-file", s.TLSCertFile,
		"File containing x509 Certificate used for serving HTTPS (with intermediate certs, if any, concatenated after server cert). "+
			"If --tls-cert-file and --tls-private-key-file are not provided, a self-signed certificate and key "+
			"are generated for the public address and saved to the directory passed to --cert-dir.")
	fs.StringVar(
		&s.TLSPrivateKeyFile, "tls-private-key-file", s.TLSPrivateKeyFile,
		"File containing x509 private key matching --tls-cert-file.")
	fs.StringVar(&s.ClientCAFile, "client-ca-file", s.ClientCAFile, ""+
		"If set, any request presenting a client certificate signed by one of the authorities in the client-ca-file "+
		"is authenticated with an identity corresponding to the CommonName of the client certificate.")
	fs.BoolVar(&s.InSecure, "insecure", s.InSecure, "If set, klusterlet server run in the in-secure mode")

	fs.BoolVar(&s.EnableImpersonation, "enable-impersonation", s.EnableImpersonation, "enable impersonation")

	s.DriverOptions.AddFlags(fs)
}
