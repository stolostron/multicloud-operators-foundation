package userpermission

import (
	"sync"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/authentication/user"
	rbacv1informers "k8s.io/client-go/informers/rbac/v1"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	clusterv1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1"
	cplisters "open-cluster-management.io/cluster-permission/client/listers/api/v1alpha1"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

// Lister enforces ability to enumerate user permissions
type Lister interface {
	// List returns the list of UserPermissions that the user can access
	List(user user.Info, selector labels.Selector) (*clusterviewv1alpha1.UserPermissionList, error)
	// Get returns a specific UserPermission by ClusterRole name
	Get(user user.Info, name string) (*clusterviewv1alpha1.UserPermission, error)
}

// Cache caches user permissions based on ClusterPermissions and discoverable ClusterRoles
type Cache struct {
	clusterRoleLister        rbacv1listers.ClusterRoleLister
	clusterRoleBindingLister rbacv1listers.ClusterRoleBindingLister
	roleLister               rbacv1listers.RoleLister
	roleBindingLister        rbacv1listers.RoleBindingLister
	managedClusterLister     clusterv1lister.ManagedClusterLister
	clusterPermissionLister  cplisters.ClusterPermissionLister

	// Permission processors (created once and reused)
	processors []permissionProcessor

	// Permission store for user and group permissions
	permissionStore *permissionStore

	// Synchronization
	lock                sync.RWMutex
	resourceVersionHash string
}

// New creates a new Cache
func New(
	clusterRoleInformer rbacv1informers.ClusterRoleInformer,
	clusterRoleBindingInformer rbacv1informers.ClusterRoleBindingInformer,
	roleInformer rbacv1informers.RoleInformer,
	roleBindingInformer rbacv1informers.RoleBindingInformer,
	managedClusterLister clusterv1lister.ManagedClusterLister,
	clusterPermissionLister cplisters.ClusterPermissionLister,
) *Cache {
	cache := &Cache{
		clusterRoleLister:        clusterRoleInformer.Lister(),
		clusterRoleBindingLister: clusterRoleBindingInformer.Lister(),
		roleLister:               roleInformer.Lister(),
		roleBindingLister:        roleBindingInformer.Lister(),
		managedClusterLister:     managedClusterLister,
		clusterPermissionLister:  clusterPermissionLister,
		permissionStore:          newPermissionStore(),
	}

	// Initialize permission processors (created once and reused)
	cache.processors = []permissionProcessor{
		&adminViewPermissionProcessor{
			managedClusterLister:     cache.managedClusterLister,
			clusterRoleLister:        cache.clusterRoleLister,
			clusterRoleBindingLister: cache.clusterRoleBindingLister,
			roleLister:               cache.roleLister,
			roleBindingLister:        cache.roleBindingLister,
		},
		&discoverablePermissionProcessor{
			clusterRoleLister:       cache.clusterRoleLister,
			clusterPermissionLister: cache.clusterPermissionLister,
		},
	}

	return cache
}

// Run begins watching and synchronizing the cache
func (c *Cache) Run(period time.Duration) {
	go utilwait.Forever(func() { c.synchronize() }, period)
}

// getPermissionRecords returns the aggregated permissions from user and groups,
// along with the discoverable roles from a consistent cache snapshot
func (c *Cache) getPermissionRecords(userInfo user.Info) (
	map[string][]clusterviewv1alpha1.ClusterBinding, []*rbacv1.ClusterRole) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	userName := userInfo.GetName()
	groups := userInfo.GetGroups()

	roleBindings := c.permissionStore.getPermissions(userName, groups)
	discoverableRoles := c.permissionStore.getDiscoverableRoles()

	return roleBindings, discoverableRoles
}

// List returns the list of UserPermissions for a user
func (c *Cache) List(
	userInfo user.Info, selector labels.Selector,
) (*clusterviewv1alpha1.UserPermissionList, error) {
	roleBindings, discoverableRoles := c.getPermissionRecords(userInfo)

	userPermissionList := &clusterviewv1alpha1.UserPermissionList{
		Items: make([]clusterviewv1alpha1.UserPermission, 0, len(roleBindings)),
	}

	for roleName, bindings := range roleBindings {
		if len(bindings) == 0 {
			continue
		}

		userPerm := clusterviewv1alpha1.UserPermission{
			ObjectMeta: metav1.ObjectMeta{
				Name: roleName,
			},
			Status: clusterviewv1alpha1.UserPermissionStatus{
				Bindings: bindings,
			},
		}

		// Add ClusterRole definition
		switch roleName {
		case clusterviewv1alpha1.ManagedClusterAdminRole:
			userPerm.Status.ClusterRoleDefinition = c.getSyntheticAdminRoleDefinition()
		case clusterviewv1alpha1.ManagedClusterViewRole:
			userPerm.Status.ClusterRoleDefinition = c.getSyntheticViewRoleDefinition()
		default:
			// Look for the role in discoverable roles
			for _, role := range discoverableRoles {
				if role.Name == roleName {
					userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
						Rules: role.Rules,
					}
					break
				}
			}
		}

		userPermissionList.Items = append(userPermissionList.Items, userPerm)
	}

	return userPermissionList, nil
}

// Get returns a specific UserPermission by ClusterRole name
func (c *Cache) Get(userInfo user.Info, name string) (*clusterviewv1alpha1.UserPermission, error) {
	roleBindings, discoverableRoles := c.getPermissionRecords(userInfo)

	bindings, exists := roleBindings[name]
	if !exists || len(bindings) == 0 {
		return nil, errors.NewNotFound(clusterviewv1alpha1.Resource("userpermissions"), name)
	}

	userPerm := &clusterviewv1alpha1.UserPermission{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: clusterviewv1alpha1.UserPermissionStatus{
			Bindings: bindings,
		},
	}

	// Add ClusterRole definition
	switch name {
	case clusterviewv1alpha1.ManagedClusterAdminRole:
		userPerm.Status.ClusterRoleDefinition = c.getSyntheticAdminRoleDefinition()
	case clusterviewv1alpha1.ManagedClusterViewRole:
		userPerm.Status.ClusterRoleDefinition = c.getSyntheticViewRoleDefinition()
	default:
		for _, role := range discoverableRoles {
			if role.Name == name {
				userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
					Rules: role.Rules,
				}
				break
			}
		}
	}

	return userPerm, nil
}

// getSyntheticAdminRoleDefinition returns the synthetic ClusterRole definition for managedcluster:admin
func (c *Cache) getSyntheticAdminRoleDefinition() clusterviewv1alpha1.ClusterRoleDefinition {
	return clusterviewv1alpha1.ClusterRoleDefinition{
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
}

// getSyntheticViewRoleDefinition returns the synthetic ClusterRole definition for managedcluster:view
func (c *Cache) getSyntheticViewRoleDefinition() clusterviewv1alpha1.ClusterRoleDefinition {
	return clusterviewv1alpha1.ClusterRoleDefinition{
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

func (c *Cache) GetUP(userInfo user.Info, name string) (*clusterviewv1alpha1.UserPermission, error) {
	up, err := c.Get(userInfo, name)
	if err != nil {
		return nil, err
	}
	return up, nil
}
