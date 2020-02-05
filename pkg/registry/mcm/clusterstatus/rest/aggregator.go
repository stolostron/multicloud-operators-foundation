// licensed Materials - Property of IBM
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

	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/klog"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/runtime"
	netutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/proxy"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

// AggregateRest is the rest interface of aggreator
type AggregateRest struct {
	getter *aggregator.InfoGetter
}

// NewAggregateRest returns a aggregator rest
func NewAggregateRest(getter *aggregator.InfoGetter) *AggregateRest {
	return &AggregateRest{
		getter: getter,
	}
}

var aggregateProxyMethods = []string{"GET", "POST", "PUT", "OPTIONS"}

// Implement Connecter
var _ = rest.Connecter(&AggregateRest{})

// New returns an empty podProxyOptions object.
func (r *AggregateRest) New() runtime.Object {
	return &mcm.ClusterStatus{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *AggregateRest) ConnectMethods() []string {
	return aggregateProxyMethods
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *AggregateRest) NewConnectOptions() (runtime.Object, bool, string) {
	return &mcm.ClusterRestOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *AggregateRest) Connect(ctx context.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	return &AggregateRestHandler{getter: r.getter, ctx: ctx, id: id, opts: opts, responder: responder}, nil
}

type AggregateRestHandler struct {
	getter    *aggregator.InfoGetter
	ctx       context.Context
	id        string
	opts      runtime.Object
	responder rest.Responder
}

func (h *AggregateRestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	requestInfo, ok := apirequest.RequestInfoFrom(ctx)
	if !ok {
		responsewriters.InternalError(w, req, fmt.Errorf("no Request info found in the context"))
		return
	}

	subResource := getPath(requestInfo.Path)
	connectionInfoGetter, ok := h.getter.Get(subResource)
	if !ok {
		responsewriters.InternalError(w, req, fmt.Errorf("no subresource found %#v", subResource))
		return
	}

	clusterInfo, proxyPath, err := connectionInfoGetter.GetConnectionInfo(ctx, h.id)
	if err != nil {
		responsewriters.InternalError(w, req, fmt.Errorf("failed to get connection info %#v", err))
		return
	}

	host := clusterInfo.Hostname
	if host == "" {
		host = clusterInfo.IP
	}

	pathToUse := proxyPath
	if clusterInfo.UseID {
		pathToUse = proxyPath + h.id
	}

	location := &url.URL{
		Scheme: clusterInfo.Scheme,
		Host:   net.JoinHostPort(host, clusterInfo.Port),
		Path:   pathToUse,
	}

	proxyOpts, ok := h.opts.(*mcm.ClusterRestOptions)
	if !ok {
		klog.Errorf("invalid options object: %#v", h.opts)
		return
	}

	location.Path = netutil.JoinPreservingTrailingSlash(location.Path, proxyOpts.Path)

	// Return a proxy handler that uses the desired transport, wrapped with additional proxy handling (to get URL rewriting, X-Forwarded-* headers, etc)
	proxy.NewUpgradeAwareHandler(location, clusterInfo.Transport, true, false, proxy.NewErrorResponder(h.responder)).ServeHTTP(w, req)
}

// getPath returns the segment behind aggregator in a URL path.
func getPath(requestPath string) string {
	// path: /apis/mcm.ibm.com/v1beta1/namespaces/<cluster-namespace>/clusterstatuses/<cluster-name>/aggregator/<sub resource>/xxx
	requestPath = strings.Trim(requestPath, "/")
	if requestPath == "" {
		return ""
	}

	pathParts := strings.Split(requestPath, "/")

	if len(pathParts) < 9 {
		return ""
	}

	return pathParts[8]
}
