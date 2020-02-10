// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"context"
	"net"
	"net/http"
	"strings"

	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/transport"
	"k8s.io/klog"
)

// InfoGetters is getter to aggregated information getter
type InfoGetters map[string]*ConnectionInfoGetter

// NewInfoGetters returns a new aggregator connection info getters
func NewInfoGetters(o *Options, client kubeclientset.Interface) *InfoGetters {
	return &InfoGetters{
		"aggregator":        o.Search.NewGetter(client),
		"metering-receiver": o.Metering.NewGetter(client),
		//"findingsapi":       o.SA.NewGetter(client),
	}
}

// ConnectionInfoGetter is getter to get connection info
type ConnectionInfoGetter struct {
	proxyPath       string
	scheme          string
	host            string
	port            string
	useID           bool
	transportConfig *transport.Config
	client          kubeclientset.Interface
	secret          string
}

// NewConnectionInfoGetter returns a new info getter
func NewConnectionInfoGetter(o *ConnectionOption, client kubeclientset.Interface, path string) *ConnectionInfoGetter {
	scheme := "http"
	if strings.HasPrefix(o.Host, "https://") {
		scheme = "https"
	}

	hostport := strings.TrimPrefix(o.Host, scheme+"://")
	host, port, _ := net.SplitHostPort(hostport)

	connInfo := &ConnectionInfoGetter{
		proxyPath: path,
		scheme:    scheme,
		host:      host,
		port:      port,
		useID:     true,
		client:    client,
		secret:    o.Secret,
	}
	connInfo.loadSecret()

	return connInfo
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
	secret, err := k.client.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Failed to get secret %s, %v", k.secret, err)
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
