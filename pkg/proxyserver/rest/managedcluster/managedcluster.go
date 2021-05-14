package managedcluster

import (
	"context"
	"fmt"

	clientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1lister "github.com/open-cluster-management/api/client/cluster/listers/cluster/v1"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/cache"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
)

type REST struct {
	client clientset.Interface
	// lister can enumerate managedCluster lists that enforce policy
	lister cache.ClusterLister

	clusterCache      *cache.ClusterCache
	clusterLister     clusterv1lister.ManagedClusterLister
	clusterRoleLister rbaclisters.ClusterRoleLister
	tableConverter    rest.TableConvertor
}

// NewREST returns a RESTStorage object that will work against ManagedCluster resources
func NewREST(
	client clientset.Interface,
	lister cache.ClusterLister,
	clusterCache *cache.ClusterCache,
	clusterLister clusterv1lister.ManagedClusterLister,
	clusterRoleLister rbaclisters.ClusterRoleLister,
) *REST {
	return &REST{
		client: client,
		lister: lister,

		clusterCache:      clusterCache,
		clusterLister:     clusterLister,
		clusterRoleLister: clusterRoleLister,
		tableConverter:    rest.NewDefaultTableConvertor(clusterv1.Resource("managedclusters")),
	}
}

// New returns a new managedCluster
func (s *REST) New() runtime.Object {
	return &clusterv1.ManagedCluster{}
}

func (s *REST) NamespaceScoped() bool {
	return false
}

// ShortNames implements the ShortNamesProvider interface. Returns a list of short names for a resource.
func (r *REST) ShortNames() []string {
	return []string{"mcv"}
}

// NewList returns a new managedCluster list
func (*REST) NewList() runtime.Object {
	return &clusterv1.ManagedClusterList{}
}

var _ = rest.Lister(&REST{})

// List retrieves a list of managedCluster that match label.
func (s *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterv1.Resource("managedclusters"), "", fmt.Errorf("unable to list managedCluster without a user on the context"))
	}

	labelSelector, _ := helpers.InternalListOptionsToSelectors(options)
	clusterList, err := s.lister.List(user, labelSelector)
	if err != nil {
		return nil, err
	}

	return clusterList, nil
}

func (c *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return c.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

var _ = rest.Watcher(&REST{})

func (s *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	if ctx == nil {
		return nil, fmt.Errorf("Context is nil")
	}
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterv1.Resource("managedclusters"), "", fmt.Errorf("unable to list managedCluster without a user on the context"))
	}

	includeAllExistingClusters := (options != nil) && options.ResourceVersion == "0"
	watcher := cache.NewCacheWatcher(user, s.clusterCache, includeAllExistingClusters)
	s.clusterCache.AddWatcher(watcher)

	go watcher.Watch()
	return watcher, nil
}

var _ = rest.Getter(&REST{})

// Get retrieves a managedCluster by name
func (s *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterv1.Resource("managedclusters"), "", fmt.Errorf("unable to get managedCluster without a user on the context"))
	}

	clusterList, err := s.lister.List(user, labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusterList.Items {
		if name == cluster.Name {
			return s.clusterCache.Get(name)
		}
	}

	return nil, errors.NewForbidden(clusterv1.Resource("managedclusters"), "", fmt.Errorf("the user cannot get the managedCluster %v", name))
}
