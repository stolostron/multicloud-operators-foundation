package log

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

type LogRest struct {
	LogProxyGetter *getter.LogProxyGetter
}

func NewLogRest(logProxyGetter *getter.LogProxyGetter) *LogRest {
	return &LogRest{LogProxyGetter: logProxyGetter}
}

// Implement Connecter
var _ = rest.Connecter(&LogRest{})

// New returns an empty podProxyOptions object.
func (r *LogRest) New() runtime.Object {
	return &v1beta1.ClusterStatus{}
}
func (r *LogRest) Destroy() {
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

	// Validate log path.  format: /ns/pod/container
	pathArray := strings.Split(proxyOpts.Path, "/")
	if len(pathArray) != 4 {
		return nil, fmt.Errorf("invalid log path, %v", proxyOpts.Path)
	}

	useClusterProxy, err := r.LogProxyGetter.ProxyServiceAvailable(ctx, id)
	if err != nil {
		return nil, err
	}
	if !useClusterProxy {
		return nil, fmt.Errorf("please enable managed-serviceaccount and cluster proxy addons")
	}
	return r.LogProxyGetter.NewHandler(ctx, id, pathArray[1], pathArray[2], pathArray[3])
}

// NewGetOptions creates a new options object
func (r *LogRest) NewGetOptions() (runtime.Object, bool, string) {
	return &v1beta1.ClusterStatusProxyOptions{}, false, ""
}
