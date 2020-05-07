package app

import (
	"crypto/tls"
	"fmt"
	"net"
	"path"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-agent/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	"k8s.io/client-go/kubernetes"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog"
)

func AgentServerRun(o *options.AgentOptions, kubeClient kubernetes.Interface) (*agent.Klusterlet, error) {
	tlsOptions, err := InitializeTLS(o)
	if err != nil {
		klog.Errorf("failed to initialize TLS: %v", err)
		return nil, err
	}

	klusterlet := agent.NewKlusterlet(o.ClusterName, kubeClient)
	go klusterlet.ListenAndServe(net.ParseIP(o.Address), uint(o.Port), tlsOptions, nil, o.InSecure)
	return klusterlet, nil
}

// InitializeTLS checks for a configured TLSCertFile and TLSPrivateKeyFile: if unspecified a new self-signed
// certificate and key file are generated. Returns a configured server.TLSOptions object.
func InitializeTLS(s *options.AgentOptions) (*agent.TLSOptions, error) {
	if s.TLSCertFile == "" && s.TLSPrivateKeyFile == "" {
		s.TLSCertFile = path.Join(s.CertDir, "kubelet.crt")
		s.TLSPrivateKeyFile = path.Join(s.CertDir, "kubelet.key")

		canReadCertAndKey, err := certutil.CanReadCertAndKey(s.TLSCertFile, s.TLSPrivateKeyFile)
		if err != nil {
			return nil, err
		}
		if !canReadCertAndKey {
			cert, key, err := certutil.GenerateSelfSignedCertKey(s.KlusterletAddress, nil, nil)
			if err != nil {
				return nil, fmt.Errorf("unable to generate self signed cert: %v", err)
			}

			if err := certutil.WriteCert(s.TLSCertFile, cert); err != nil {
				return nil, err
			}

			if err := keyutil.WriteKey(s.TLSPrivateKeyFile, key); err != nil {
				return nil, err
			}

			klog.V(5).Infof("Using self-signed cert (%s, %s)", s.TLSCertFile, s.TLSPrivateKeyFile)
		}
	}

	tlsOptions := &agent.TLSOptions{
		CertFile: s.TLSCertFile,
		KeyFile:  s.TLSPrivateKeyFile,
		Config:   &tls.Config{},
	}

	if len(s.ClientCAFile) > 0 {
		clientCAs, err := certutil.NewPool(s.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load client CA file %s: %v", s.ClientCAFile, err)
		}
		// Specify allowed CAs for client certificates
		tlsOptions.Config.ClientCAs = clientCAs
		// Populate PeerCertificates in requests, but don't reject connections without verified certificates
		tlsOptions.Config.ClientAuth = tls.RequestClientCert
	}

	return tlsOptions, nil
}
