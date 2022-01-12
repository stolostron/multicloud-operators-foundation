// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"

	"github.com/stolostron/multicloud-operators-foundation/cmd/agent/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/agent"
	"k8s.io/client-go/kubernetes"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

func AgentServerRun(o *options.AgentOptions, kubeClient kubernetes.Interface) (*agent.Agent, error) {
	tlsOptions, err := InitializeTLS(o)
	if err != nil {
		klog.Errorf("failed to initialize TLS: %v", err)
		return nil, err
	}

	agent := agent.NewAgent(o.ClusterName, kubeClient)
	go agent.ListenAndServe(net.ParseIP(o.Address), uint(o.Port), tlsOptions, nil, o.InSecure)
	return agent, nil
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
			cert, key, err := certutil.GenerateSelfSignedCertKey(s.AgentAddress, nil, nil)
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
		Config:   &tls.Config{MinVersion: tls.VersionTLS12},
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

// ServeHealthProbes starts a server to check healthz and readyz probes
func ServeHealthProbes(stop <-chan struct{}, healthProbeBindAddress string, configCheck healthz.Checker) {
	healthzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"healthz-ping": healthz.Ping,
		"configz-ping": configCheck,
	}}
	readyzHandler := &healthz.Handler{Checks: map[string]healthz.Checker{
		"readyz-ping": healthz.Ping,
	}}

	mux := http.NewServeMux()
	mux.Handle("/readyz", http.StripPrefix("/readyz", readyzHandler))
	mux.Handle("/healthz", http.StripPrefix("/healthz", healthzHandler))

	server := http.Server{
		Handler: mux,
	}

	ln, err := net.Listen("tcp", healthProbeBindAddress)
	if err != nil {
		klog.Errorf("error listening on %s: %v", ":8000", err)
		return
	}

	klog.Infof("heath probes server is running...")
	// Run server
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()

	// Shutdown the server when stop is closed
	<-stop
	if err := server.Shutdown(context.Background()); err != nil {
		klog.Fatal(err)
	}
}
