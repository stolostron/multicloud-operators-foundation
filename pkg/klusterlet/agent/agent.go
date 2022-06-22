package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"reflect"

	"github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/log/drivers"
	"k8s.io/client-go/kubernetes"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
)

// Agent is the main struct for agent server
type Agent struct {
	clusterName string
	kubeClient  kubernetes.Interface
	server      *Server
	RunServer   chan v1beta1.ManagedClusterInfo
}

// NewAgent create a new Agent
func NewAgent(clusterName string, kubeClient kubernetes.Interface) *Agent {
	return &Agent{
		clusterName: clusterName,
		kubeClient:  kubeClient,
		RunServer:   make(chan v1beta1.ManagedClusterInfo),
	}
}

// ListenAndServe start Agent server
func (k *Agent) ListenAndServe(
	address net.IP,
	port uint,
	tlsOptions *TLSOptions,
	auth AuthInterface,
	insecure bool,
) {
	var certData []byte
	if tlsOptions.Config.ClientCAs == nil && !insecure {
		runServer := false
		for !runServer {
			clusterStatus := <-k.RunServer
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

	factory := drivers.NewDriverFactory(k.kubeClient)
	k.server = newServer(factory, address, port, tlsOptions, auth, certData)
	if err := k.server.listenAndServe(); err != nil {
		klog.Errorf("failed to listen and serve: %v", err)
	}
}

func (k *Agent) RefreshServerIfNeeded(clusterInfo *v1beta1.ManagedClusterInfo) {
	// do not refresh is server is not started
	if k.server == nil {
		return
	}
	pool, caData, err := k.waitCAFromClusterStatus(*clusterInfo, k.server.CAData)
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
	klog.Infof("refresh server: %v\n", caData)
	go k.server.refresh(caData, pool)
}

func (k *Agent) waitCAFromClusterStatus(clusterInfo v1beta1.ManagedClusterInfo, oldCA []byte) (*x509.CertPool, []byte, error) {
	if len(clusterInfo.Spec.LoggingCA) == 0 {
		return nil, nil, fmt.Errorf("kluster ca is empty")
	}

	if reflect.DeepEqual(clusterInfo.Spec.LoggingCA, oldCA) {
		return nil, nil, nil
	}

	data := clusterInfo.Spec.LoggingCA
	certs, err := certutil.ParseCertsPEM(data)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}

	return pool, clusterInfo.Spec.LoggingCA, nil
}
