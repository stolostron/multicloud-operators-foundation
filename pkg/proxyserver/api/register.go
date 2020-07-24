package api

import (
	"context"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apis"
	v1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apis/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	proxyrest "github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/rest"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

var (
	// Scheme contains the types needed by the resource metrics API.
	Scheme = runtime.NewScheme()
	// ParameterCodec handles versioning of objects that are converted to query parameters.
	ParameterCodec = runtime.NewParameterCodec(Scheme)
	// Codecs is a codec factory for serving the resource metrics API.
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	apis.Install(Scheme)

	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

func Install(proxyServiceInfoGetter *getter.ProxyServiceInfoGetter,
	logConnectionInfoGetter getter.ConnectionInfoGetter,
	server *genericapiserver.GenericAPIServer) error {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(v1beta1.GroupName, Scheme, ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[v1beta1.SchemeGroupVersion.Version] = map[string]rest.Storage{
		"clusterstatuses":            &clusterStatusStorage{},
		"clusterstatuses/aggregator": proxyrest.NewProxyRest(proxyServiceInfoGetter),
		"clusterstatuses/log":        proxyrest.NewLogRest(logConnectionInfoGetter),
	}

	return server.InstallAPIGroup(&apiGroupInfo)
}

type clusterStatusStorage struct{}

var (
	_ = rest.Storage(&clusterStatusStorage{})
	_ = rest.KindProvider(&clusterStatusStorage{})
	_ = rest.Lister(&clusterStatusStorage{})
	_ = rest.Getter(&clusterStatusStorage{})
	_ = rest.Scoper(&clusterStatusStorage{})
)

// Storage interface
func (s *clusterStatusStorage) New() runtime.Object {
	return &v1beta1.ClusterStatus{}
}

// KindProvider interface
func (s *clusterStatusStorage) Kind() string {
	return "ClusterStatus"
}

// Lister interface
func (s *clusterStatusStorage) NewList() runtime.Object {
	return &v1beta1.ClusterStatusList{}
}

// Lister interface
func (s *clusterStatusStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return &v1beta1.ClusterStatusList{}, nil
}

// Getter interface
func (s *clusterStatusStorage) Get(ctx context.Context, name string, opts *metav1.GetOptions) (runtime.Object, error) {
	return &v1beta1.ClusterStatus{}, nil
}

// Scoper interface
func (s *clusterStatusStorage) NamespaceScoped() bool {
	return true
}

func (s *clusterStatusStorage) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return nil, nil
}
