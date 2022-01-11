package getter

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"k8s.io/klog"
)

// ClientConfig is to define the configuration to connect klusterlet
type ClientConfig struct {
	// Default port - used if no information about klusterlet port can be found in Node.NodeStatus.DaemonEndpoints.
	Port uint
	// EnableHTTPS enable https
	EnableHTTPS bool

	// TLSClientConfig contains settings to enable transport layer security
	restclient.TLSClientConfig

	// Server requires Bearer authentication
	BearerToken string

	// HTTPTimeout is used by the client to timeout http requests to klusterlet.
	HTTPTimeout time.Duration

	// Dial is a custom dialer used for the client
	Dial utilnet.DialFunc

	// DynamicClient is the kube client
	DynamicClient dynamic.Interface

	// CertDir is the directory to put cert
	CertDir string
}

// ConnectionInfo provides the information needed to connect to a kubelet
type ConnectionInfo struct {
	Scheme    string
	Hostname  string
	IP        string
	Port      string
	Transport http.RoundTripper
	UserRoot  bool
	UseID     bool
}

var clusterInfoGVR = v1beta1.GroupVersion.WithResource("managedclusterinfos")

// ConnectionInfoGetter provides ConnectionInfo for the kubelet running on a named node
type ConnectionInfoGetter interface {
	GetConnectionInfo(ctx context.Context, clusterName string) (*ConnectionInfo, error)
}

func NewLogConnectionInfoGetter(clientConfig ClientConfig) (ConnectionInfoGetter, error) {
	clusterGetter := ClusterGetterFunc(
		func(ctx context.Context, name string, options metav1.GetOptions) (*v1beta1.ManagedClusterInfo, error) {
			obj, err := clientConfig.DynamicClient.Resource(clusterInfoGVR).Namespace(name).Get(context.TODO(), name, options)
			if err != nil {
				klog.Errorf("failed to get managedclusterinfos %v, error: %v", name, err)
				return nil, err
			}

			clusterInfo := &v1beta1.ManagedClusterInfo{}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), clusterInfo)
			if err != nil {
				klog.Errorf("failed to convert %v, error: %v", obj, err)
				return nil, err
			}

			return clusterInfo, nil
		})

	connectionInfoGetter, err := NewClusterConnectionInfoGetter(clusterGetter, clientConfig)
	if err != nil {
		return nil, err
	}

	return connectionInfoGetter, nil
}

// MakeTransport return a tranport for http
func MakeTransport(config *ClientConfig) (http.RoundTripper, error) {
	tlsConfig, err := transport.TLSConfigFor(config.transportConfig())
	if err != nil {
		return nil, err
	}

	rt := http.DefaultTransport
	if config.Dial != nil || tlsConfig != nil {
		rt = utilnet.SetOldTransportDefaults(&http.Transport{
			DialContext:     config.Dial,
			TLSClientConfig: tlsConfig,
		})
	}

	return transport.HTTPWrappersForConfig(config.transportConfig(), rt)
}

// transportConfig converts a client config to an appropriate transport config.
func (c *ClientConfig) transportConfig() *transport.Config {
	cfg := &transport.Config{
		TLS: transport.TLSConfig{
			CAFile:   c.CAFile,
			CAData:   c.CAData,
			CertFile: c.CertFile,
			CertData: c.CertData,
			KeyFile:  c.KeyFile,
			KeyData:  c.KeyData,
		},
		BearerToken: c.BearerToken,
	}
	if c.EnableHTTPS && !cfg.HasCA() {
		cfg.TLS.Insecure = true
	}
	return cfg
}

// ClusterGetter defines an interface for looking up a cluster by name
type ClusterGetter interface {
	Get(ctx context.Context, name string, options metav1.GetOptions) (*v1beta1.ManagedClusterInfo, error)
}

// ClusterGetterFunc allows implementing NodeGetter with a function
type ClusterGetterFunc func(ctx context.Context, name string, options metav1.GetOptions) (*v1beta1.ManagedClusterInfo, error)

// Get defines a cluster getter function
func (f ClusterGetterFunc) Get(ctx context.Context, name string, options metav1.GetOptions) (*v1beta1.ManagedClusterInfo, error) {
	return f(ctx, name, options)
}

// ClusterConnectionInfoGetter obtains connection info from the status of a Node API object
type ClusterConnectionInfoGetter struct {
	// nodes is used to look up cluster objects
	clusters ClusterGetter
	// scheme is the scheme to use to connect to all klusterlets
	scheme string
	// defaultPort is the port to use if no klusterlet endpoint port is recorded in the cluster status
	defaultPort int
	// config is the cofig to use to send a request to all klusterlet
	config *ClientConfig
}

// NewClusterConnectionInfoGetter is a getter to get cluster status
func NewClusterConnectionInfoGetter(clusters ClusterGetter, config ClientConfig) (ConnectionInfoGetter, error) {
	scheme := "http"
	if config.EnableHTTPS {
		scheme = "https"
	}

	return &ClusterConnectionInfoGetter{
		clusters:    clusters,
		scheme:      scheme,
		defaultPort: int(config.Port),
		config:      &config,
	}, nil
}

// GetConnectionInfo return the connection into to klusterlet
func (k *ClusterConnectionInfoGetter) GetConnectionInfo(ctx context.Context, clusterName string) (*ConnectionInfo, error) {
	cluster, err := k.clusters.Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Use the kubelet-reported port, if present
	port := int(cluster.Status.LoggingPort.Port)
	if port <= 0 {
		port = k.defaultPort
	}

	hostname := cluster.Status.LoggingEndpoint.Hostname
	ip := cluster.Status.LoggingEndpoint.IP

	// need a deep copy
	config := &ClientConfig{
		Port:        k.config.Port,
		EnableHTTPS: k.config.EnableHTTPS,
		HTTPTimeout: k.config.HTTPTimeout,
		CertDir:     k.config.CertDir,
	}
	config.CertFile = k.config.CertFile
	config.KeyFile = k.config.KeyFile

	// Set customized dialer
	if hostname != "" && ip != "" {
		defaultTransport := http.DefaultTransport.(*http.Transport)
		config.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			updatedaddr := ip + ":" + strconv.FormatInt(int64(port), 10)
			return defaultTransport.DialContext(ctx, network, updatedaddr)
		}
	}

	transport, err := MakeTransport(config)
	if err != nil {
		return nil, err
	}

	return &ConnectionInfo{
		Scheme:    k.scheme,
		Hostname:  hostname,
		IP:        ip,
		Port:      strconv.Itoa(port),
		Transport: transport,
		UserRoot:  false,
	}, nil
}
