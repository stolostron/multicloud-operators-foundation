package app

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/api"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/informers"
)

type ProxyServer struct {
	*genericapiserver.GenericAPIServer
}

func NewProxyServer(
	informerFactory informers.SharedInformerFactory,
	apiServerConfig *genericapiserver.Config,
	proxyGetter *getter.ProxyServiceInfoGetter,
	logGetter getter.ConnectionInfoGetter) (*ProxyServer, error) {
	apiServer, err := apiServerConfig.Complete(informerFactory).New("proxy-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	if err := api.Install(proxyGetter, logGetter, apiServer); err != nil {
		return nil, err
	}

	return &ProxyServer{apiServer}, nil
}

func (p *ProxyServer) Run(stopCh <-chan struct{}) error {
	return p.GenericAPIServer.PrepareRun().Run(stopCh)
}
