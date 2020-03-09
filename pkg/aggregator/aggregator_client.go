// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"context"
	"net/http"
	"sync"

	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/transport"
	"k8s.io/klog"
)

// ClientOptions is the options for aggregator client
type ClientOptions struct {
	name        string
	service     string
	port        string
	useID       bool
	path        string
	subResource string
	secret      string
}

// ConnectionInfoGetter is getter to get connection info
type ConnectionInfoGetter struct {
	name            string
	proxyPath       string
	scheme          string
	host            string
	port            string
	useID           bool
	secret          string
	transportConfig *transport.Config
	client          kubeclientset.Interface
}

// InfoGetters is getter to aggregated information getter
type InfoGetters struct {
	mutex               sync.RWMutex
	nameToSubResource   map[string]string
	subResourceToGetter map[string]*ConnectionInfoGetter
	client              kubeclientset.Interface
}

// NewInfoGetters returns a new aggregator connection info getter
func NewInfoGetters(client kubeclientset.Interface) *InfoGetters {
	return &InfoGetters{
		nameToSubResource:   make(map[string]string),
		subResourceToGetter: make(map[string]*ConnectionInfoGetter),
		client:              client}
}

func (a *InfoGetters) AddAndUpdate(o *ClientOptions) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if subResource, existed := a.nameToSubResource[o.name]; existed {
		delete(a.subResourceToGetter, subResource)
	}

	a.nameToSubResource[o.name] = o.subResource
	a.subResourceToGetter[o.subResource] = NewConnectionInfoGetter(o, a.client)
	klog.V(5).Infof("add/update aggregator getter %v/%v: %#v", o.name, o.subResource, a.subResourceToGetter[o.subResource])
}

func (a *InfoGetters) Delete(name string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if subResource, ok := a.nameToSubResource[name]; ok {
		delete(a.nameToSubResource, name)
		delete(a.subResourceToGetter, subResource)
	}
}

func (a *InfoGetters) Get(subResource string) (*ConnectionInfoGetter, bool) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if getter, existed := a.subResourceToGetter[subResource]; existed {
		return getter, true
	}

	return nil, false
}

// NewConnectionInfoGetter returns a new info getter
func NewConnectionInfoGetter(o *ClientOptions, client kubeclientset.Interface) *ConnectionInfoGetter {
	connInfo := &ConnectionInfoGetter{
		name:      o.name,
		proxyPath: "/" + o.path + "/",
		scheme:    "https",
		port:      o.port,
		useID:     o.useID,
		secret:    o.secret,
		client:    client,
	}

	connInfo.loadHost(o.service)

	//TODO: reload secret if secret is updated
	connInfo.loadSecret()

	return connInfo
}

func (k *ConnectionInfoGetter) loadHost(service string) {
	namespace, name, err := cache.SplitMetaNamespaceKey(service)
	if err != nil {
		klog.Errorf("service %s format is not correct, %v", service, err)
		return
	}

	if namespace == "" {
		k.host = name
	} else {
		k.host = name + "." + namespace
	}
}

func (k *ConnectionInfoGetter) loadSecret() {
	if k.client == nil {
		klog.Warningf("kube client is nil, skip secret load")
		return
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(k.secret)
	if err != nil {
		klog.Errorf("Secret %s format is not correct, %v", k.secret, err)
		return
	}

	if namespace == "" {
		defaultNamespace, _, err := cache.SplitMetaNamespaceKey(k.name)
		if err != nil && defaultNamespace != "" {
			klog.Errorf("connInfo name %s format is not correct, %v", k.name, err)
			return
		}
		namespace = defaultNamespace
	}

	secret, err := k.client.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get secret %s, %v", k.secret, err)
		return
	}

	cfg := &transport.Config{
		TLS: transport.TLSConfig{
			CAData:   secret.Data["ca.crt"],
			CertData: secret.Data["tls.crt"],
			KeyData:  secret.Data["tls.key"],
		},
	}

	k.transportConfig = cfg
}

// GetConnectionInfo return the connection into to klusterlet
func (k *ConnectionInfoGetter) GetConnectionInfo(ctx context.Context, clusterName string) (*klusterlet.ConnectionInfo, string, error) {
	transportConfig := &transport.Config{}
	if k.transportConfig == nil {
		k.loadSecret()
	}

	rt := http.DefaultTransport
	if k.transportConfig != nil {
		tlsConfig, err := transport.TLSConfigFor(k.transportConfig)
		if err != nil {
			return nil, "", err
		}

		if tlsConfig != nil {
			rt = utilnet.SetOldTransportDefaults(&http.Transport{
				TLSClientConfig: tlsConfig,
			})
		}
		transportConfig = k.transportConfig
	}

	transport, err := transport.HTTPWrappersForConfig(transportConfig, rt)
	if err != nil {
		return nil, "", err
	}

	return &klusterlet.ConnectionInfo{
		Scheme:    k.scheme,
		Hostname:  k.host,
		Port:      k.port,
		Transport: transport,
		UseID:     k.useID,
	}, k.proxyPath, nil
}
