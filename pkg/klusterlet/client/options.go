// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"crypto/x509"
	"path"
	"time"

	"github.com/spf13/pflag"
	certutil "k8s.io/client-go/util/cert"
)

// KlusterletClientOptions is the options for klusterlet client
type KlusterletClientOptions struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	CertDirectory string
	PairName      string
}

// NewKlusterletClientOptions creates a new NewKlusterletClientOptions object with default values.
func NewKlusterletClientOptions() *KlusterletClientOptions {
	s := &KlusterletClientOptions{
		CertFile:      "",
		KeyFile:       "",
		CAFile:        "",
		CertDirectory: "apiserver.local.config/certificates",
		PairName:      "klusterlet",
	}

	return s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *KlusterletClientOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.CertFile, "klusterlet-certfile", s.CertFile, ""+
		"Klusterlet client cert file")
	fs.StringVar(&s.KeyFile, "klusterlet-keyfile", s.KeyFile, ""+
		"Klusterlet client key file")
	fs.StringVar(&s.CAFile, "klusterlet-cafile", s.CAFile, ""+
		"Klusterlet ca file")
	fs.StringVar(&s.CertDirectory, "klusterlet-cert-dir", s.CertDirectory, ""+
		"Klusterlet cert directory")
}

// MaybeDefaultWithSelfSignedCerts generate self signed cert if they are not set
func (s *KlusterletClientOptions) MaybeDefaultWithSelfSignedCerts(publicAddress string) error {
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
		caKey, err := certutil.NewPrivateKey()
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

		key, err := certutil.NewPrivateKey()
		if err != nil {
			return err
		}

		config.Usages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		cert, err := certutil.NewSignedCert(config, key, caCert, caKey)
		if err != nil {
			return err
		}

		caData := certutil.EncodeCertPEM(caCert)
		keyData := certutil.EncodePrivateKeyPEM(key)
		certData := certutil.EncodeCertPEM(cert)

		if err := certutil.WriteCert(s.CertFile, certData); err != nil {
			return err
		}
		if err := certutil.WriteCert(s.CAFile, caData); err != nil {
			return err
		}

		if err := certutil.WriteKey(s.KeyFile, keyData); err != nil {
			return err
		}
	}

	return nil
}

// Config returns KlusterletClientConfig from options
func (s *KlusterletClientOptions) Config() KlusterletClientConfig {
	config := KlusterletClientConfig{
		Port:        443,
		EnableHTTPS: true,
		HTTPTimeout: 30 * time.Second,
	}

	config.CertFile = s.CertFile
	config.KeyFile = s.KeyFile
	config.CertDir = s.CertDirectory

	return config
}
