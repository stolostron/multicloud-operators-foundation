package api

import (
	"context"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/rest/log"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/rest/managedcluster"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/rest/managedclusterset"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/rest/project"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/rest/proxy"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/rest/userpermission"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/client-go/informers"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	cplisters "open-cluster-management.io/cluster-permission/client/listers/api/v1alpha1"

	apisclusterview "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/clusterview"
	clusterviewv1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/clusterview/v1"
	clusterviewv1alpha1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/clusterview/v1alpha1"

	apisproxy "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/proxy"
	proxyv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/getter"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	kubecache "k8s.io/client-go/tools/cache"
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
	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	apisproxy.Install(Scheme)
	apisclusterview.Install(Scheme)
}

func Install(proxyServiceInfoGetter *getter.ProxyServiceInfoGetter,
	logProxyGetter *getter.LogProxyGetter,
	server *genericapiserver.GenericAPIServer,
	client clusterclient.Interface,
	informerFactory informers.SharedInformerFactory,
	clusterInformer clusterinformers.SharedInformerFactory,
	clusterPermissionInformer kubecache.SharedIndexInformer,
	clusterPermissionLister cplisters.ClusterPermissionLister) error {
	if err := installProxyGroup(proxyServiceInfoGetter, logProxyGetter, server); err != nil {
		return err
	}
	if err := installClusterViewGroup(server, client, informerFactory, clusterInformer,
		clusterPermissionInformer, clusterPermissionLister); err != nil {
		return err
	}
	return nil
}

func installClusterViewGroup(server *genericapiserver.GenericAPIServer,
	client clusterclient.Interface,
	informerFactory informers.SharedInformerFactory,
	clusterInformer clusterinformers.SharedInformerFactory,
	clusterPermissionInformer kubecache.SharedIndexInformer,
	clusterPermissionLister cplisters.ClusterPermissionLister,
) error {

	clusterCache := cache.NewClusterCache(
		clusterInformer.Cluster().V1().ManagedClusters(),
		informerFactory.Rbac().V1().ClusterRoles(),
		informerFactory.Rbac().V1().ClusterRoleBindings(),
		utils.GetViewResourceFromClusterRole,
	)

	clusterPermissionInformer.AddIndexers(
		kubecache.Indexers{
			project.ClusterPermissionSubjectIndexKey: project.IndexClusterPermissionBySubject,
		},
	)

	v1storage := map[string]rest.Storage{
		"managedclusters": managedcluster.NewREST(
			client, clusterCache, clusterCache,
			clusterInformer.Cluster().V1().ManagedClusters().Lister(),
			informerFactory.Rbac().V1().ClusterRoles().Lister(),
		),
		"kubevirtprojects": project.NewREST(clusterPermissionInformer.GetIndexer(), clusterPermissionLister),
	}

	clusterSetCache := cache.NewClusterSetCache(
		clusterInformer.Cluster().V1beta2().ManagedClusterSets(),
		informerFactory.Rbac().V1().ClusterRoles(),
		informerFactory.Rbac().V1().ClusterRoleBindings(),
		utils.GetViewResourceFromClusterRole,
	)

	userPermissionCache := cache.NewUserPermissionCache(
		informerFactory.Rbac().V1().ClusterRoles(),
		informerFactory.Rbac().V1().ClusterRoleBindings(),
		informerFactory.Rbac().V1().Roles(),
		informerFactory.Rbac().V1().RoleBindings(),
		clusterInformer.Cluster().V1().ManagedClusters().Lister(),
		clusterPermissionLister,
	)

	v1beta1storage := map[string]rest.Storage{
		"managedclustersets": managedclusterset.NewREST(
			client, clusterSetCache, clusterSetCache,
			clusterInformer.Cluster().V1beta2().ManagedClusterSets().Lister(),
			informerFactory.Rbac().V1().ClusterRoles().Lister(),
		),
		"userpermissions": userpermission.NewREST(userPermissionCache),
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(clusterviewv1.GroupName, Scheme, ParameterCodec, Codecs)

	apiGroupInfo.VersionedResourcesStorageMap[clusterviewv1.SchemeGroupVersion.Version] = v1storage
	apiGroupInfo.VersionedResourcesStorageMap[clusterviewv1alpha1.SchemeGroupVersion.Version] = v1beta1storage

	go clusterCache.Run(1 * time.Second)
	go clusterSetCache.Run(1 * time.Second)
	go userPermissionCache.Run(2 * time.Second)
	return server.InstallAPIGroup(&apiGroupInfo)
}

func installProxyGroup(proxyServiceInfoGetter *getter.ProxyServiceInfoGetter,
	logProxyGetter *getter.LogProxyGetter,
	server *genericapiserver.GenericAPIServer) error {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(proxyv1beta1.GroupName, Scheme, ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[proxyv1beta1.SchemeGroupVersion.Version] = map[string]rest.Storage{
		"clusterstatuses":            &clusterStatusStorage{},
		"clusterstatuses/aggregator": proxy.NewProxyRest(proxyServiceInfoGetter),
		"clusterstatuses/log":        log.NewLogRest(logProxyGetter),
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

	_ = rest.SingularNameProvider(&clusterStatusStorage{})
)

// Storage interface
func (s *clusterStatusStorage) New() runtime.Object {
	return &proxyv1beta1.ClusterStatus{}
}
func (s *clusterStatusStorage) Destroy() {
}

// KindProvider interface
func (s *clusterStatusStorage) Kind() string {
	return "ClusterStatus"
}

// Lister interface
func (s *clusterStatusStorage) NewList() runtime.Object {
	return &proxyv1beta1.ClusterStatusList{}
}

// Lister interface
func (s *clusterStatusStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return &proxyv1beta1.ClusterStatusList{}, nil
}

// Getter interface
func (s *clusterStatusStorage) Get(ctx context.Context, name string, opts *metav1.GetOptions) (runtime.Object, error) {
	return &proxyv1beta1.ClusterStatus{}, nil
}

// Scoper interface
func (s *clusterStatusStorage) NamespaceScoped() bool {
	return true
}

func (s *clusterStatusStorage) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return nil, nil
}

// SingularNameProvider interface
func (s *clusterStatusStorage) GetSingularName() string {
	return "clusterstatus"
}
