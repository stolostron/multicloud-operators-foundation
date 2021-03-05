package log

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	"k8s.io/apimachinery/pkg/runtime"
	netutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog"
)

type LogRest struct {
	KlusterletConn getter.ConnectionInfoGetter
}

func NewLogRest(connectionInfoGetter getter.ConnectionInfoGetter) *LogRest {
	return &LogRest{connectionInfoGetter}
}

// Implement Connecter
var _ = rest.Connecter(&LogRest{})

// New returns an empty podProxyOptions object.
func (r *LogRest) New() runtime.Object {
	return &v1beta1.ClusterStatus{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *LogRest) ConnectMethods() []string {
	return []string{"GET", "OPTIONS"}
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *LogRest) NewConnectOptions() (runtime.Object, bool, string) {
	return &v1beta1.ClusterStatusProxyOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *LogRest) Connect(ctx context.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	proxyOpts, ok := opts.(*v1beta1.ClusterStatusProxyOptions)
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
	klog.V(2).Infof("Proxy to %s", location.Path)
	// Return a proxy handler that uses the desired transport, wrapped with additional proxy handling (to get URL rewriting, X-Forwarded-* headers, etc)
	return proxy.NewUpgradeAwareHandler(location, transport, true, false, proxy.NewErrorResponder(responder)), nil
}

// NewGetOptions creates a new options object
func (r *LogRest) NewGetOptions() (runtime.Object, bool, string) {
	return &v1beta1.ClusterStatusProxyOptions{}, false, ""
}

// resourceLocation returns the resource URL for a cluster.
func resourceLocation(
	ctx context.Context,
	connInfo getter.ConnectionInfoGetter,
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
