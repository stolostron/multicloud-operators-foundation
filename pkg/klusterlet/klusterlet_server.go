// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
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
	ks.server = &http.Server{
		Addr:    net.JoinHostPort(ks.address.String(), strconv.FormatUint(uint64(ks.port), 10)),
		Handler: &ks.handler,
		TLSConfig: &tls.Config{
			GetConfigForClient: ks.getConfigForClient,
		},
		MaxHeaderBytes: 1 << 20,
	}
	if ks.tlsOptions != nil {
		cert, err := tls.LoadX509KeyPair(ks.tlsOptions.CertFile, ks.tlsOptions.KeyFile)
		if err != nil {
			return err
		}
		// This certificates is for client handshake
		ks.tlsOptions.Config.Certificates = []tls.Certificate{cert}

		return ks.server.ListenAndServeTLS(ks.tlsOptions.CertFile, ks.tlsOptions.KeyFile)
	}
	return ks.server.ListenAndServe()
}

func (ks *klusterletServer) getConfigForClient(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	return ks.tlsOptions.Config, nil
}

func (ks *klusterletServer) refresh(caData []byte, pool *x509.CertPool) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	klog.Infof("Refreshing klusterlet server...")
	ks.CAData = caData
	ks.tlsOptions.Config.ClientCAs = pool
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

func (k *Klusterlet) refreshServerIfNeeded(clusterStatus *v1alpha1.ClusterStatus) {
	// do not refresh is server is not started
	if k.server == nil {
		return
	}
	pool, caData, err := k.waitCAFromClusterStatus(*clusterStatus, k.server.CAData)
	if err != nil {
		klog.Warningf("failed to get ca data: %v", err)
		return
	}

	if caData == nil {
		klog.V(5).Infof("ca data does not change, skip refresh")
		return
	}

	if k.server.isShutDown() {
		return
	}

	go k.server.refresh(caData, pool)
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
