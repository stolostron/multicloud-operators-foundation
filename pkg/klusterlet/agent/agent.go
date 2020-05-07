package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"reflect"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/log/drivers"
	"k8s.io/client-go/kubernetes"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
)

// Klusterlet is the main struct for klusterlet server
type Klusterlet struct {
	clusterName string
	kubeClient  kubernetes.Interface
	server      *Server
	RunServer   chan v1beta1.ClusterInfo
}

// NewKlusterlet create a new klusterlet
func NewKlusterlet(clusterName string, kubeClient kubernetes.Interface) *Klusterlet {
	return &Klusterlet{
		clusterName: clusterName,
		kubeClient:  kubeClient,
		RunServer:   make(chan v1beta1.ClusterInfo),
	}
}

// ListenAndServe start klusterlet server
func (k *Klusterlet) ListenAndServe(
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

func (k *Klusterlet) RefreshServerIfNeeded(clusterInfo *v1beta1.ClusterInfo) {
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

	go k.server.refresh(caData, pool)
}

func (k *Klusterlet) waitCAFromClusterStatus(clusterInfo v1beta1.ClusterInfo, oldCA []byte) (*x509.CertPool, []byte, error) {
	if len(clusterInfo.Spec.KlusterletCA) == 0 {
		return nil, nil, fmt.Errorf("kluster ca is empty")
	}

	if reflect.DeepEqual(clusterInfo.Spec.KlusterletCA, oldCA) {
		return nil, nil, nil
	}

	data := clusterInfo.Spec.KlusterletCA
	certs, err := certutil.ParseCertsPEM(data)
	if err != nil {
		return nil, nil, err
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}

	return pool, clusterInfo.Spec.KlusterletCA, nil
}
