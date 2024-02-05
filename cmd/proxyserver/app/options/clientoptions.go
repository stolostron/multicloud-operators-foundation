// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"crypto/x509"
	"path"
	"time"

	"github.com/spf13/pflag"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/client-go/dynamic"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

// ClientOptions is the options for agent client
type ClientOptions struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	CertDirectory      string
	PairName           string
	ProxyServiceCAFile string
	ProxyServiceName   string
	ProxyServicePort   string
}

// NewClientOptions creates a new agent ClientOptions object with default values.
func NewClientOptions() *ClientOptions {
	s := &ClientOptions{
		CertFile:           "",
		KeyFile:            "",
		CAFile:             "",
		CertDirectory:      "apiserver.local.config/certificates",
		PairName:           "agent",
		ProxyServiceCAFile: "/var/run/clusterproxy/service-ca.crt",
		ProxyServiceName:   "cluster-proxy-addon-user",
		ProxyServicePort:   "9092",
	}

	return s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *ClientOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.CertFile, "agent-certfile", s.CertFile, ""+
		"Agent client cert file")
	fs.StringVar(&s.KeyFile, "agent-keyfile", s.KeyFile, ""+
		"Agent client key file")
	fs.StringVar(&s.CAFile, "agent-cafile", s.CAFile, ""+
		"Agent ca file")
	fs.StringVar(&s.CertDirectory, "agent-cert-dir", s.CertDirectory, ""+
		"Agent cert directory")
	fs.StringVar(&s.ProxyServiceName, "proxy-service-name", s.ProxyServiceName, ""+
		"Proxy service name")
	fs.StringVar(&s.ProxyServicePort, "proxy-service-port", s.ProxyServicePort, ""+
		"Proxy service port")
	fs.StringVar(&s.ProxyServiceCAFile, "proxy-service-cafile", s.ProxyServiceCAFile, ""+
		"Proxy service CA file name")
}

// MaybeDefaultWithSelfSignedCerts generate self signed cert if they are not set
func (s *ClientOptions) MaybeDefaultWithSelfSignedCerts(publicAddress string) error {
	if len(s.CertFile) != 0 || len(s.KeyFile) != 0 || len(s.CAFile) != 0 {
		return nil
	}

	s.CertFile = path.Join(s.CertDirectory, s.PairName+".crt")
	s.KeyFile = path.Join(s.CertDirectory, s.PairName+".key")
	s.CAFile = path.Join(s.CertDirectory, s.PairName+"-ca.crt")

	canReadCertAndKey, err := certutil.CanReadCertAndKey(s.CertFile, s.KeyFile)
	if err != nil {
		return err
	}

	if !canReadCertAndKey {
		caKey, err := utils.NewPrivateKey()
		if err != nil {
			return err
		}

		config := certutil.Config{
			CommonName: publicAddress,
		}
		caCert, err := certutil.NewSelfSignedCACert(config, caKey)
		if err != nil {
			return err
		}

		key, err := utils.NewPrivateKey()
		if err != nil {
			return err
		}

		config.Usages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		cert, err := utils.NewSignedCert(config, key, caCert, caKey)
		if err != nil {
			return err
		}

		caData := utils.EncodeCertPEM(caCert)
		keyData := utils.EncodePrivateKeyPEM(key)
		certData := utils.EncodeCertPEM(cert)

		if err := certutil.WriteCert(s.CertFile, certData); err != nil {
			return err
		}
		if err := certutil.WriteCert(s.CAFile, caData); err != nil {
			return err
		}

		if err := keyutil.WriteKey(s.KeyFile, keyData); err != nil {
			return err
		}
	}

	return nil
}

// Config returns agent ClientConfig from options
func (s *ClientOptions) Config(dynamicClient dynamic.Interface) getter.ClientConfig {
	config := getter.ClientConfig{
		Port:          443,
		EnableHTTPS:   true,
		HTTPTimeout:   30 * time.Second,
		DynamicClient: dynamicClient,
	}

	config.CertFile = s.CertFile
	config.KeyFile = s.KeyFile
	config.CertDir = s.CertDirectory

	return config
}
