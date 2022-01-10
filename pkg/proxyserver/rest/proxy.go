package rest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"

	v1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apiserver/pkg/registry/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
)

// ProxyREST implements the proxy subresource for a Service
type ProxyRest struct {
	*getter.ProxyServiceInfoGetter
}

func NewProxyRest(serviceInfoGetter *getter.ProxyServiceInfoGetter) *ProxyRest {
	return &ProxyRest{serviceInfoGetter}
}

var _ = rest.Connecter(&ProxyRest{})

// implement storage interface
func (r *ProxyRest) New() runtime.Object {
	return &v1beta1.ClusterStatus{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *ProxyRest) ConnectMethods() []string {
	//TODO: may need more methods
	return []string{"GET", "POST", "PUT", "OPTIONS"}
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *ProxyRest) NewConnectOptions() (runtime.Object, bool, string) {
	return &v1beta1.ClusterStatusProxyOptions{}, true, "path"
}

// Connect returns a handler for the pod proxy
func (r *ProxyRest) Connect(
	ctx context.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	return &proxyRestHandler{ctx: ctx, id: id, opts: opts, responder: responder, getter: r.ProxyServiceInfoGetter}, nil
}

type proxyRestHandler struct {
	ctx       context.Context
	id        string
	opts      runtime.Object
	responder rest.Responder
	getter    *getter.ProxyServiceInfoGetter
}

func (h *proxyRestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if _, ok := apirequest.RequestInfoFrom(ctx); !ok {
		http.Error(w, fmt.Sprintf("no request info found in the context"), http.StatusBadRequest)
		return
	}

	// the request URL must comply with the rules
	subResource, err := getSubResource(req.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("the request %s is forbidden", req.URL.Path), http.StatusForbidden)
		return
	}

	serviceInfo := h.getter.Get(subResource)
	if serviceInfo == nil {
		klog.Warningf("The proxy service cannot be found for %s", req.URL.Path)
		http.Error(w, fmt.Sprintf("the proxy service (%s) is not found", subResource), http.StatusNotFound)
		return
	}

	transport, err := restclient.TransportFor(serviceInfo.RestConfig)
	if err != nil {
		klog.Errorf("failed to build transport for %s", serviceInfo.Name)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	proxyPath := serviceInfo.RootPath
	if serviceInfo.UseID {
		proxyPath = path.Join(proxyPath, h.id)
	}
	proxyOpts, ok := h.opts.(*v1beta1.ClusterStatusProxyOptions)
	if !ok {
		klog.Errorf("invalid options object: %#v", h.opts)
		http.Error(w, "failed to get proxy path", http.StatusInternalServerError)
		return
	}
	proxyPath = path.Join("/", proxyPath, proxyOpts.Path)

	host := fmt.Sprintf("%s.%s.svc", serviceInfo.ServiceName, serviceInfo.ServiceNamespace)

	location := &url.URL{
		Scheme: "https", // should always be https
		Host:   net.JoinHostPort(host, serviceInfo.ServicePort),
		Path:   proxyPath,
	}

	klog.V(5).Infof("Proxy %s to %s", req.URL.Path, location.Path)
	// Return a proxy handler that uses the desired transport, wrapped with additional proxy handling
	// (to get URL rewriting, X-Forwarded-* headers, etc)
	proxyHandler := proxy.NewUpgradeAwareHandler(location, transport, true, false, proxy.NewErrorResponder(h.responder))
	proxyHandler.ServeHTTP(w, req)
}

// getSubResource returns the segment behind aggregator in a URL path.
func getSubResource(requestPath string) (string, error) {
	// path:
	// /apis/proxy.open-cluster-management.io/v1beta1/namespaces/<cluster-namespace>/
	// clusterstatuses/<cluster-name>/aggregator/<sub resource>/xxx
	requestPath = strings.Trim(requestPath, "/")
	if requestPath == "" {
		return "", fmt.Errorf("empty path")
	}

	pathParts := strings.Split(requestPath, "/")

	if len(pathParts) < 9 {
		return "", fmt.Errorf("wrong path format")
	}

	return pathParts[8], nil
}
