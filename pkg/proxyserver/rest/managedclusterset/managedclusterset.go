package managedclusterset

import (
	"context"
	"fmt"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/proxyserver/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	clientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1beta2lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1beta2"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
)

type REST struct {
	client clientset.Interface
	// lister can enumerate managedClusterSet lists that enforce policy
	lister cache.ClusterSetLister

	clusterSetCache   *cache.ClusterSetCache
	clusterSetLister  clusterv1beta2lister.ManagedClusterSetLister
	clusterRoleLister rbaclisters.ClusterRoleLister
	tableConverter    rest.TableConvertor
}

// NewREST returns a RESTStorage object that will work against ManagedClusterSet resources
func NewREST(
	client clientset.Interface,
	lister cache.ClusterSetLister,
	clusterSetCache *cache.ClusterSetCache,
	clusterSetLister clusterv1beta2lister.ManagedClusterSetLister,
	clusterRoleLister rbaclisters.ClusterRoleLister,
) *REST {
	return &REST{
		client: client,
		lister: lister,

		clusterSetCache:   clusterSetCache,
		clusterSetLister:  clusterSetLister,
		clusterRoleLister: clusterRoleLister,
		tableConverter:    rest.NewDefaultTableConvertor(clusterv1beta2.Resource("managedclustersets")),
	}
}

// New returns a new managedClusterSet
func (s *REST) New() runtime.Object {
	return &clusterv1beta2.ManagedClusterSet{}
}

func (s *REST) Destroy() {
}

func (s *REST) NamespaceScoped() bool {
	return false
}

// NewList returns a new managedClusterSet list
func (*REST) NewList() runtime.Object {
	return &clusterv1beta2.ManagedClusterSetList{}
}

var _ = rest.Lister(&REST{})

// List retrieves a list of managedClusterSet that match label.
func (s *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterv1beta2.Resource("managedclustersets"), "", fmt.Errorf("unable to list managedClusterset without a user on the context"))
	}

	labelSelector, _ := helpers.InternalListOptionsToSelectors(options)
	clusterSetList, err := s.lister.List(user, labelSelector)
	if err != nil {
		return nil, err
	}

	return clusterSetList, nil
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
		return nil, errors.NewForbidden(clusterv1beta2.Resource("managedclustersets"), "", fmt.Errorf("unable to list managedClusterSet without a user on the context"))
	}

	includeAllExistingClusterSets := (options != nil) && options.ResourceVersion == "0"
	watcher := cache.NewCacheWatcher(user, s.clusterSetCache, includeAllExistingClusterSets)
	s.clusterSetCache.AddWatcher(watcher)

	go watcher.Watch()
	return watcher, nil
}

var _ = rest.Getter(&REST{})

// Get retrieves a managedClusterSet by name
func (s *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterv1beta2.Resource("managedclustersets"), "", fmt.Errorf("unable to get managedClusterSet without a user on the context"))
	}

	clusterSetList, err := s.lister.List(user, labels.Everything())
	if err != nil {
		return nil, err
	}
	for _, clusterSet := range clusterSetList.Items {
		if name == clusterSet.Name {
			return s.clusterSetCache.Get(name)
		}
	}

	return nil, errors.NewForbidden(clusterv1beta2.Resource("managedclustersets"), "", fmt.Errorf("the user cannot get the managedClusterSet %v", name))
}
