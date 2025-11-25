// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/api"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/informers"
	kubecache "k8s.io/client-go/tools/cache"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	cplisters "open-cluster-management.io/cluster-permission/client/listers/api/v1alpha1"
)

type ProxyServer struct {
	*genericapiserver.GenericAPIServer
}

func NewProxyServer(
	client clusterv1client.Interface,
	informerFactory informers.SharedInformerFactory,
	clusterInformer clusterv1informers.SharedInformerFactory,
	clusterPermissionInformer kubecache.SharedIndexInformer,
	clusterPermissionLister cplisters.ClusterPermissionLister,
	apiServerConfig *genericapiserver.Config,
	proxyGetter *getter.ProxyServiceInfoGetter,
	logProxyGetter *getter.LogProxyGetter) (*ProxyServer, error) {
	apiServer, err := apiServerConfig.Complete(informerFactory).New("proxy-server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	if err := api.Install(proxyGetter, logProxyGetter, apiServer, client,
		informerFactory, clusterInformer, clusterPermissionInformer, clusterPermissionLister); err != nil {
		return nil, err
	}

	return &ProxyServer{apiServer}, nil
}

func (p *ProxyServer) Run(stopCh <-chan struct{}) error {
	return p.GenericAPIServer.PrepareRun().Run(stopCh)
}
