// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	netutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
)

// LogREST implements the log endpoint for a Pod
type LogREST struct {
	KlusterletConn klusterlet.ConnectionInfoGetter
}

func NewLogRest(optsGetter generic.RESTOptionsGetter, connection klusterlet.ConnectionInfoGetter) *LogREST {
	return &LogREST{
		KlusterletConn: connection,
	}
}

// Implement Connecter
var _ = rest.Connecter(&LogREST{})

var proxyMethods = []string{"GET", "OPTIONS"}

// New returns an empty podProxyOptions object.
func (r *LogREST) New() runtime.Object {
	return &mcm.ClusterStatus{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *LogREST) ConnectMethods() []string {
	return proxyMethods
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *LogREST) NewConnectOptions() (runtime.Object, bool, string) {
	return &mcm.ClusterRestOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *LogREST) Connect(ctx context.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	proxyOpts, ok := opts.(*mcm.ClusterRestOptions)
	if !ok {
		return nil, fmt.Errorf("invalid options object: %#v", opts)
	}
	location, transport, err := resourceLocation(ctx, r.KlusterletConn, id, "/containerLogs/")
	if err != nil {
		return nil, err
	}

	// Validate log path
	pathArray := strings.Split(proxyOpts.Path, "/")
	if len(pathArray) != 4 {
		return nil, fmt.Errorf("invalid log path, %v", proxyOpts.Path)
	}

	location.Path = netutil.JoinPreservingTrailingSlash(location.Path, proxyOpts.Path)

	// Return a proxy handler that uses the desired transport, wrapped with additional proxy handling (to get URL rewriting, X-Forwarded-* headers, etc)
	return proxy.NewUpgradeAwareHandler(location, transport, true, false, proxy.NewErrorResponder(responder)), nil
}

// NewGetOptions creates a new options object
func (r *LogREST) NewGetOptions() (runtime.Object, bool, string) {
	return &mcm.ClusterRestOptions{}, false, ""
}

// MonitorREST implements the monitor endpoint for a Pod
type MonitorREST struct {
	KlusterletConn klusterlet.ConnectionInfoGetter
}

func NewMonitorRest(optsGetter generic.RESTOptionsGetter, connection klusterlet.ConnectionInfoGetter) *MonitorREST {
	return &MonitorREST{
		KlusterletConn: connection,
	}
}

// Implement Connecter
var _ = rest.Connecter(&MonitorREST{})

// New returns an empty podProxyOptions object.
func (r *MonitorREST) New() runtime.Object {
	return &mcm.ClusterStatus{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *MonitorREST) ConnectMethods() []string {
	return proxyMethods
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *MonitorREST) NewConnectOptions() (runtime.Object, bool, string) {
	return &mcm.ClusterRestOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *MonitorREST) Connect(ctx context.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	proxyOpts, ok := opts.(*mcm.ClusterRestOptions)
	if !ok {
		return nil, fmt.Errorf("invalid options object: %#v", opts)
	}
	location, transport, err := resourceLocation(ctx, r.KlusterletConn, id, "/monitoring/")
	if err != nil {
		return nil, err
	}

	location.Path = netutil.JoinPreservingTrailingSlash(location.Path, proxyOpts.Path)

	// Return a proxy handler that uses the desired transport, wrapped with additional proxy handling (to get URL rewriting, X-Forwarded-* headers, etc)
	return proxy.NewUpgradeAwareHandler(location, transport, true, false, proxy.NewErrorResponder(responder)), nil
}

// NewGetOptions creates a new options object
func (r *MonitorREST) NewGetOptions() (runtime.Object, bool, string) {
	return &mcm.ClusterRestOptions{}, false, ""
}

// resourceLocation returns the resource URL for a cluster.
func resourceLocation(
	ctx context.Context,
	connInfo klusterlet.ConnectionInfoGetter,
	name string,
	path string,
) (*url.URL, http.RoundTripper, error) {
	clusterInfo, err := connInfo.GetConnectionInfo(ctx, name)
	if err != nil {
		return nil, nil, err
	}

	host := clusterInfo.Hostname
	if host == "" {
		host = clusterInfo.IP
	}

	loc := &url.URL{
		Scheme: clusterInfo.Scheme,
		Host:   net.JoinHostPort(host, clusterInfo.Port),
		Path:   path,
	}

	// Backward compatible
	// TODO remove in next release
	if clusterInfo.UserRoot {
		loc.Path = "/"
	}

	return loc, clusterInfo.Transport, nil
}
