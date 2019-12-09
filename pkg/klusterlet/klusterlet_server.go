// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"sync"

	"k8s.io/klog"

	v1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/drivers"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/server"
	certutil "k8s.io/client-go/util/cert"
)

type klusterletServer struct {
	server     *http.Server
	handler    server.Server
	tlsOptions *server.TLSOptions
	address    net.IP
	port       uint
	CAData     []byte
	mu         sync.RWMutex
}

func newKlusterletServer(factory *drivers.DriverFactory,
	address net.IP,
	port uint,
	tlsOptions *server.TLSOptions,
	auth server.AuthInterface,
	certData []byte,
) *klusterletServer {
	handler := server.NewServer(factory, auth)
	return &klusterletServer{
		handler:    handler,
		address:    address,
		port:       port,
		tlsOptions: tlsOptions,
		CAData:     certData,
		server:     nil,
	}
}

func (ks *klusterletServer) listenAndServe() error {
	ks.mu.Lock()
	ks.server = &http.Server{
		Addr:           net.JoinHostPort(ks.address.String(), strconv.FormatUint(uint64(ks.port), 10)),
		Handler:        &ks.handler,
		MaxHeaderBytes: 1 << 20,
	}
	ks.mu.Unlock()
	if ks.tlsOptions != nil {
		ks.server.TLSConfig = ks.tlsOptions.Config
		// Passing empty strings as the cert and key files means no
		// cert/keys are specified and GetCertificate in the TLSConfig
		// should be called instead.
		return ks.server.ListenAndServeTLS(ks.tlsOptions.CertFile, ks.tlsOptions.KeyFile)
	}
	return ks.server.ListenAndServe()
}

func (ks *klusterletServer) restart(caData []byte, pool *x509.CertPool) {
	ks.mu.Lock()
	klog.Infof("Restarting klusterlet server...")
	klserver := ks.server
	ks.server = nil
	if err := klserver.Shutdown(context.Background()); err != nil {
		klog.Errorf("shut down klusterlet server meet error: %v", err)
	}
	ks.mu.Unlock()

	ks.CAData = caData
	ks.tlsOptions.Config.ClientCAs = pool
	if err := ks.listenAndServe(); err != nil {
		klog.Errorf("failed to listen and serve: %v", err)
	}
}

func (ks *klusterletServer) isShutDown() bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.server == nil
}

// ListenAndServe start klusterlet server
func (k *Klusterlet) ListenAndServe(
	factory *drivers.DriverFactory,
	address net.IP,
	port uint,
	tlsOptions *server.TLSOptions,
	auth server.AuthInterface,
	insecure bool,
) {
	var certData []byte
	if tlsOptions.Config.ClientCAs == nil && !insecure {
		runServer := false
		for !runServer {
			clusterStatus := <-k.runServer
			pool, data, err := k.waitCAFromClusterStatus(clusterStatus, nil)
			if err != nil {
				klog.Errorf("failed to get ca file from hub cluster: %v", err)
				continue
			}

			tlsOptions.Config.ClientCAs = pool
			tlsOptions.Config.ClientAuth = tls.RequireAndVerifyClientCert
			runServer = true
			certData = data
		}
	}

	k.server = newKlusterletServer(factory, address, port, tlsOptions, auth, certData)
	if err := k.server.listenAndServe(); err != nil {
		klog.Errorf("failed to listen and serve: %v", err)
	}
}

func (k *Klusterlet) restartServerIfNeeded(clusterStatus *v1alpha1.ClusterStatus) {
	// do not restart is server is not started
	if k.server == nil {
		return
	}
	pool, caData, err := k.waitCAFromClusterStatus(*clusterStatus, k.server.CAData)
	if err != nil {
		klog.Warningf("failed to get ca data: %v", err)
		return
	}

	if caData == nil {
		klog.V(5).Infof("ca data does not change, skip restart")
		return
	}

	if k.server.isShutDown() {
		return
	}

	go k.server.restart(caData, pool)
}

func (k *Klusterlet) waitCAFromClusterStatus(clusterStatus v1alpha1.ClusterStatus, oldCA []byte) (*x509.CertPool, []byte, error) {
	if len(clusterStatus.Spec.KlusterletCA) == 0 {
		return nil, nil, fmt.Errorf("kluster ca is empty")
	}

	if reflect.DeepEqual(clusterStatus.Spec.KlusterletCA, oldCA) {
		return nil, nil, nil
	}

	data := clusterStatus.Spec.KlusterletCA
	certs, err := certutil.ParseCertsPEM(data)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}

	return pool, clusterStatus.Spec.KlusterletCA, nil
}
