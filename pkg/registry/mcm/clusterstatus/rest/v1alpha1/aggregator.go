// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/runtime"
	netutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
)

// AggregateRest is the rest interface of aggreator
type AggregateRest struct {
	getter *v1alpha1.ConnectionInfoGetter
}

// NewAggregateRest returns a aggregator rest
func NewAggregateRest(optsGetter generic.RESTOptionsGetter, getter *v1alpha1.ConnectionInfoGetter) *AggregateRest {
	return &AggregateRest{
		getter: getter,
	}
}

var aggregatProxyMethods = []string{"GET", "POST", "PUT", "OPTIONS"}

// Implement Connecter
var _ = rest.Connecter(&AggregateRest{})

// New returns an empty podProxyOptions object.
func (r *AggregateRest) New() runtime.Object {
	return &mcm.ClusterStatus{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *AggregateRest) ConnectMethods() []string {
	return aggregatProxyMethods
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *AggregateRest) NewConnectOptions() (runtime.Object, bool, string) {
	return &mcm.ClusterRestOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *AggregateRest) Connect(ctx context.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	proxyOpts, ok := opts.(*mcm.ClusterRestOptions)
	if !ok {
		return nil, fmt.Errorf("invalid options object: %#v", opts)
	}

	clusterInfo, path, err := r.getter.GetConnectionInfo(ctx, id)
	if err != nil {
		return nil, err
	}

	host := clusterInfo.Hostname
	if host == "" {
		host = clusterInfo.IP
	}

	pathToUse := path
	if clusterInfo.UseID {
		pathToUse = path + id
	}

	location := &url.URL{
		Scheme: clusterInfo.Scheme,
		Host:   net.JoinHostPort(host, clusterInfo.Port),
		Path:   pathToUse,
	}

	location.Path = netutil.JoinPreservingTrailingSlash(location.Path, proxyOpts.Path)

	// Return a proxy handler that uses the desired transport, wrapped with additional proxy handling (to get URL rewriting, X-Forwarded-* headers, etc)
	return proxy.NewUpgradeAwareHandler(location, clusterInfo.Transport, true, false, proxy.NewErrorResponder(responder)), nil
}
