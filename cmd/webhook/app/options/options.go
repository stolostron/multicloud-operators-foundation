// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"crypto/tls"

	"github.com/spf13/pflag"
	"k8s.io/klog"
)

// Config contains the server (the webhook) cert and key.
type Options struct {
	CertFile                          string
	KeyFile                           string
	KubeConfigFile                    string
	EnableManagedClustersetValidating bool
}

// NewOptions constructs a new set of default options for webhook.
func NewOptions() *Options {
	return &Options{
		KubeConfigFile:                    "",
		CertFile:                          "",
		KeyFile:                           "",
		EnableManagedClustersetValidating: false,
	}
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.CertFile, "tls-cert-file", c.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	fs.StringVar(&c.KeyFile, "tls-private-key-file", c.KeyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
	fs.StringVar(&c.KubeConfigFile, "kube-config-file", c.KubeConfigFile, ""+
		"Kube configuration file")
	fs.BoolVar(&c.EnableManagedClustersetValidating, "enable-managedclusterset-validating", c.EnableManagedClustersetValidating,
		"enable managedclusterset validating")
}

func ConfigTLS(o *Options) *tls.Config {
	sCert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
	if err != nil {
		klog.Fatal(err)
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{sCert},
	}
}
