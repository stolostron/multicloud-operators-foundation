package agent

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/log/drivers"
	"k8s.io/klog"
)

type Server struct {
	server     *http.Server
	handler    Handle
	tlsOptions *TLSOptions
	address    net.IP
	port       uint
	CAData     []byte
	mu         sync.RWMutex
}

func newServer(
	factory *drivers.DriverFactory,
	address net.IP,
	port uint,
	tlsOptions *TLSOptions,
	auth AuthInterface,
	certData []byte,
) *Server {
	return &Server{
		handler:    NewHandle(factory, auth),
		address:    address,
		port:       port,
		tlsOptions: tlsOptions,
		CAData:     certData,
		server:     nil,
	}
}

func (s *Server) listenAndServe() error {
	s.server = &http.Server{
		Addr:    net.JoinHostPort(s.address.String(), strconv.FormatUint(uint64(s.port), 10)),
		Handler: &s.handler,
		TLSConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			GetConfigForClient: s.getConfigForClient,
		},
		MaxHeaderBytes: 1 << 20,
	}
	if s.tlsOptions != nil {
		cert, err := tls.LoadX509KeyPair(s.tlsOptions.CertFile, s.tlsOptions.KeyFile)
		if err != nil {
			return err
		}
		// This certificates is for client handshake
		s.tlsOptions.Config.Certificates = []tls.Certificate{cert}

		return s.server.ListenAndServeTLS(s.tlsOptions.CertFile, s.tlsOptions.KeyFile)
	}
	return s.server.ListenAndServe()
}

func (s *Server) getConfigForClient(clientHello *tls.ClientHelloInfo) (*tls.Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.tlsOptions.Config, nil
}

func (s *Server) refresh(caData []byte, pool *x509.CertPool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	klog.Infof("Refreshing Agent server...")
	s.CAData = caData
	s.tlsOptions.Config.ClientCAs = pool
}

func (s *Server) isShutDown() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.server == nil
}
