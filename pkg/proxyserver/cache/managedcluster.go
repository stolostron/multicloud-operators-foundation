package cache

import (
	"time"

	clusterinformerv1 "github.com/open-cluster-management/api/client/cluster/informers/externalversions/cluster/v1"
	clusterv1lister "github.com/open-cluster-management/api/client/cluster/listers/cluster/v1"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/authentication/user"
	rbacv1informers "k8s.io/client-go/informers/rbac/v1"
)

// ClusterLister enforces ability to enumerate cluster based on role
type ClusterLister interface {
	// List returns the list of ManagedCluster items that the user can access
	List(user user.Info, selector labels.Selector) (*clusterv1.ManagedClusterList, error)
}

type ClusterCache struct {
	cache         *AuthCache
	clusterLister clusterv1lister.ManagedClusterLister
}

func NewClusterCache(clusterInformer clusterinformerv1.ManagedClusterInformer,
	clusterRoleInformer rbacv1informers.ClusterRoleInformer,
	clusterRolebindingInformer rbacv1informers.ClusterRoleBindingInformer,
) *ClusterCache {
	clusterCache := &ClusterCache{
		clusterLister: clusterInformer.Lister(),
	}
	authCache := NewAuthCache(clusterRoleInformer, clusterRolebindingInformer,
		"cluster.open-cluster-management.io", "managedclusters",
		clusterInformer.Informer(),
		clusterCache.ListResources,
	)
	clusterCache.cache = authCache

	return clusterCache
}

func (c *ClusterCache) ListResources() (sets.String, error) {
	allClusters := sets.String{}
	clusters, err := c.clusterLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters {
		allClusters.Insert(cluster.Name)
	}
	return allClusters, nil
}

func (c *ClusterCache) List(userInfo user.Info, selector labels.Selector) (*clusterv1.ManagedClusterList, error) {
	names := c.cache.listNames(userInfo)

	clusterList := &clusterv1.ManagedClusterList{}
	for key := range names {
		cluster, err := c.clusterLister.Get(key)
		if errors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		if !selector.Matches(labels.Set(cluster.Labels)) {
			continue
		}
		clusterList.Items = append(clusterList.Items, *cluster)
	}
	return clusterList, nil
}

func (c *ClusterCache) ListObjects(userInfo user.Info) (runtime.Object, error) {
	return c.List(userInfo, labels.Everything())
}

func (c *ClusterCache) Get(name string) (runtime.Object, error) {
	return c.clusterLister.Get(name)
}

func (c *ClusterCache) ConvertResource(name string) runtime.Object {
	cluster, err := c.clusterLister.Get(name)
	if err != nil {
		cluster = &clusterv1.ManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: name}}
	}

	return cluster
}

func (c *ClusterCache) RemoveWatcher(w CacheWatcher) {
	c.cache.RemoveWatcher(w)
}

func (c *ClusterCache) AddWatcher(w CacheWatcher) {
	c.cache.AddWatcher(w)
}

// Run begins watching and synchronizing the cache
func (c *ClusterCache) Run(period time.Duration) {
	go utilwait.Forever(func() { c.cache.synchronize() }, period)
}
