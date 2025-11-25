package cache

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/authentication/user"
	rbacv1informers "k8s.io/client-go/informers/rbac/v1"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	kubecache "k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	clusterv1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1"
	clusterpermissionv1alpha1 "open-cluster-management.io/cluster-permission/api/v1alpha1"
	cplisters "open-cluster-management.io/cluster-permission/client/listers/api/v1alpha1"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

type UserPermissionRecord struct {
	Subject     string
	Permissions []PermissionInfo
}

type PermissionInfo struct {
	ClusterRoleName string
	Bindings        []clusterviewv1alpha1.ClusterBinding
}

// userPermissionRecordKeyFn is a key func for UserPermissionRecord objects
func userPermissionRecordKeyFn(obj interface{}) (string, error) {
	record, ok := obj.(*UserPermissionRecord)
	if !ok {
		return "", fmt.Errorf("expected UserPermissionRecord")
	}
	return record.Subject, nil
}

// UserPermissionLister enforces ability to enumerate user permissions
type UserPermissionLister interface {
	// List returns the list of UserPermissions that the user can access
	List(user user.Info, selector labels.Selector) (*clusterviewv1alpha1.UserPermissionList, error)
	// Get returns a specific UserPermission by ClusterRole name
	Get(user user.Info, name string) (*clusterviewv1alpha1.UserPermission, error)
}

// UserPermissionCache caches user permissions based on ClusterPermissions and discoverable ClusterRoles
type UserPermissionCache struct {
	clusterRoleLister        rbacv1listers.ClusterRoleLister
	clusterRoleBindingLister rbacv1listers.ClusterRoleBindingLister
	roleLister               rbacv1listers.RoleLister
	roleBindingLister        rbacv1listers.RoleBindingLister
	managedClusterLister     clusterv1lister.ManagedClusterLister
	clusterPermissionLister  cplisters.ClusterPermissionLister

	// Cache for user/group -> UserPermissionRecord mapping
	userPermissionStore  kubecache.Store
	groupPermissionStore kubecache.Store

	// Discoverable ClusterRoles cache
	discoverableRoles      []*rbacv1.ClusterRole
	discoverableRolesNames sets.Set[string]

	// Synchronization
	lock             sync.RWMutex
	resourceVersionHash string
}

// NewUserPermissionCache creates a new UserPermissionCache
func NewUserPermissionCache(
	clusterRoleInformer rbacv1informers.ClusterRoleInformer,
	clusterRoleBindingInformer rbacv1informers.ClusterRoleBindingInformer,
	roleInformer rbacv1informers.RoleInformer,
	roleBindingInformer rbacv1informers.RoleBindingInformer,
	managedClusterLister clusterv1lister.ManagedClusterLister,
	clusterPermissionLister cplisters.ClusterPermissionLister,
) *UserPermissionCache {
	cache := &UserPermissionCache{
		clusterRoleLister:        clusterRoleInformer.Lister(),
		clusterRoleBindingLister: clusterRoleBindingInformer.Lister(),
		roleLister:               roleInformer.Lister(),
		roleBindingLister:        roleBindingInformer.Lister(),
		managedClusterLister:     managedClusterLister,
		clusterPermissionLister:  clusterPermissionLister,
		userPermissionStore:      kubecache.NewStore(userPermissionRecordKeyFn),
		groupPermissionStore:     kubecache.NewStore(userPermissionRecordKeyFn),
		discoverableRolesNames:   sets.New[string](),
	}

	return cache
}

// Run begins watching and synchronizing the cache
func (c *UserPermissionCache) Run(period time.Duration) {
	go utilwait.Forever(func() { c.synchronize() }, period)
}

// calculateResourceVersionHash computes a hash of relevant resource versions
// Only includes resources that affect user permissions:
// 1. Discoverable ClusterRoles
// 2. ClusterPermissions
// 3. ManagedClusters
// 4. ClusterRoleBindings and their referenced ClusterRoles (that grant managedcluster permissions)
// 5. RoleBindings and their referenced Roles/ClusterRoles (that grant managedcluster permissions)
func (c *UserPermissionCache) calculateResourceVersionHash() (string, error) {
	h := sha256.New()
	var versions []string

	// 1. Discoverable ClusterRoles
	clusterRoles, err := c.clusterRoleLister.List(labels.Everything())
	if err != nil {
		return "", fmt.Errorf("failed to list ClusterRoles: %w", err)
	}
	for _, cr := range clusterRoles {
		if cr.Labels != nil && cr.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] == "true" {
			versions = append(versions, fmt.Sprintf("cr:%s:%s", cr.Name, cr.ResourceVersion))
		}
	}

	// 2. ClusterPermissions
	clusterPermissions, err := c.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		return "", fmt.Errorf("failed to list ClusterPermissions: %w", err)
	}
	for _, cp := range clusterPermissions {
		versions = append(versions, fmt.Sprintf("cp:%s:%s", cp.Namespace, cp.ResourceVersion))
	}

	// 3. ManagedClusters
	managedClusters, err := c.managedClusterLister.List(labels.Everything())
	if err != nil {
		return "", fmt.Errorf("failed to list ManagedClusters: %w", err)
	}
	clusterNames := sets.New[string]()
	for _, mc := range managedClusters {
		versions = append(versions, fmt.Sprintf("mc:%s:%s", mc.Name, mc.ResourceVersion))
		clusterNames.Insert(mc.Name)
	}

	// Track ClusterRoles/Roles that grant managedcluster permissions (to avoid duplicates)
	trackedClusterRoles := sets.New[string]()
	trackedRoles := sets.New[string]() // key: namespace/name

	// 4. ClusterRoleBindings that grant managedcluster permissions
	clusterRoleBindings, err := c.clusterRoleBindingLister.List(labels.Everything())
	if err != nil {
		return "", fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}
	for _, crb := range clusterRoleBindings {
		clusterRole, err := c.clusterRoleLister.Get(crb.RoleRef.Name)
		if err != nil {
			continue
		}
		grantsAdminPerm := c.clusterRoleGrantsPermission(clusterRole, "managedclusteractions", "action.open-cluster-management.io")
		grantsViewPerm := c.clusterRoleGrantsPermission(clusterRole, "managedclusterviews", "view.open-cluster-management.io")
		if grantsAdminPerm || grantsViewPerm {
			// Track the ClusterRoleBinding
			versions = append(versions, fmt.Sprintf("crb:%s:%s", crb.Name, crb.ResourceVersion))

			// Track the ClusterRole if not already tracked and not discoverable
			if !trackedClusterRoles.Has(clusterRole.Name) {
				if clusterRole.Labels == nil || clusterRole.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] != "true" {
					versions = append(versions, fmt.Sprintf("cr:%s:%s", clusterRole.Name, clusterRole.ResourceVersion))
					trackedClusterRoles.Insert(clusterRole.Name)
				}
			}
		}
	}

	// 5. RoleBindings in managedcluster namespaces that grant managedcluster permissions
	for clusterName := range clusterNames {
		roleBindings, err := c.roleBindingLister.RoleBindings(clusterName).List(labels.Everything())
		if err != nil {
			continue
		}
		for _, rb := range roleBindings {
			var grantsAdminPerm, grantsViewPerm bool

			if rb.RoleRef.Kind == "ClusterRole" {
				clusterRole, err := c.clusterRoleLister.Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				grantsAdminPerm = c.clusterRoleGrantsPermission(clusterRole, "managedclusteractions", "action.open-cluster-management.io")
				grantsViewPerm = c.clusterRoleGrantsPermission(clusterRole, "managedclusterviews", "view.open-cluster-management.io")

				if grantsAdminPerm || grantsViewPerm {
					// Track the ClusterRole if not already tracked and not discoverable
					if !trackedClusterRoles.Has(clusterRole.Name) {
						if clusterRole.Labels == nil || clusterRole.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] != "true" {
							versions = append(versions, fmt.Sprintf("cr:%s:%s", clusterRole.Name, clusterRole.ResourceVersion))
							trackedClusterRoles.Insert(clusterRole.Name)
						}
					}
				}
			} else if rb.RoleRef.Kind == "Role" {
				role, err := c.roleLister.Roles(clusterName).Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				grantsAdminPerm = c.roleGrantsPermission(role, "managedclusteractions", "action.open-cluster-management.io")
				grantsViewPerm = c.roleGrantsPermission(role, "managedclusterviews", "view.open-cluster-management.io")

				if grantsAdminPerm || grantsViewPerm {
					// Track the Role if not already tracked
					roleKey := fmt.Sprintf("%s/%s", role.Namespace, role.Name)
					if !trackedRoles.Has(roleKey) {
						versions = append(versions, fmt.Sprintf("r:%s:%s", roleKey, role.ResourceVersion))
						trackedRoles.Insert(roleKey)
					}
				}
			}

			if grantsAdminPerm || grantsViewPerm {
				versions = append(versions, fmt.Sprintf("rb:%s/%s:%s", rb.Namespace, rb.Name, rb.ResourceVersion))
			}
		}
	}

	// Sort for deterministic hashing
	sort.Strings(versions)

	// Write all versions to hash
	for _, v := range versions {
		h.Write([]byte(v))
		h.Write([]byte("\n"))
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// synchronize runs a full synchronization of the cache
func (c *UserPermissionCache) synchronize() {
	// Calculate hash of all resource versions before acquiring the lock
	newHash, err := c.calculateResourceVersionHash()
	if err != nil {
		klog.Errorf("Failed to calculate resource version hash: %v", err)
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	// Check if resources have changed
	if c.resourceVersionHash == newHash {
		klog.V(4).Info("No changes detected in resources, skipping synchronization")
		return
	}

	if c.resourceVersionHash == "" {
		klog.V(2).Info("Initial synchronization of UserPermissionCache")
	} else {
		klog.V(2).Infof("Resource changes detected (hash changed from %s to %s), synchronizing cache", c.resourceVersionHash[:8], newHash[:8])
	}

	// Build subject -> role -> bindings mapping
	// map[subject]map[roleName][]ClusterBinding
	userPermissions := make(map[string]map[string][]clusterviewv1alpha1.ClusterBinding)
	groupPermissions := make(map[string]map[string][]clusterviewv1alpha1.ClusterBinding)

	// Process managedcluster admin/view permissions from RoleBindings and ClusterRoleBindings first
	c.processManagedClusterPermissions(userPermissions, groupPermissions)

	// Get all discoverable ClusterRoles
	discoverableRoles, err := c.getDiscoverableClusterRoles()
	if err != nil {
		klog.Errorf("Failed to get discoverable ClusterRoles: %v", err)
		return
	}

	c.discoverableRoles = discoverableRoles
	c.discoverableRolesNames = sets.New[string]()
	for _, role := range discoverableRoles {
		c.discoverableRolesNames.Insert(role.Name)
	}

	// Get all ClusterPermissions
	clusterPermissions, err := c.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ClusterPermissions: %v", err)
		return
	}

	// Then process ClusterPermissions
	for _, cp := range clusterPermissions {
		clusterName := cp.Namespace

		// Process ClusterRoleBinding (single)
		if cp.Spec.ClusterRoleBinding != nil {
			c.processClusterRoleBinding(cp.Spec.ClusterRoleBinding, clusterName, userPermissions, groupPermissions)
		}

		// Process ClusterRoleBindings (multiple)
		if cp.Spec.ClusterRoleBindings != nil {
			for _, binding := range *cp.Spec.ClusterRoleBindings {
				c.processClusterRoleBinding(&binding, clusterName, userPermissions, groupPermissions)
			}
		}

		// Process RoleBindings
		if cp.Spec.RoleBindings != nil {
			for _, binding := range *cp.Spec.RoleBindings {
				c.processRoleBinding(&binding, clusterName, userPermissions, groupPermissions)
			}
		}
	}

	// Convert to UserPermissionRecord and update stores
	for userName, roleBindings := range userPermissions {
		permissions := make([]PermissionInfo, 0, len(roleBindings))
		for roleName, bindings := range roleBindings {
			permissions = append(permissions, PermissionInfo{
				ClusterRoleName: roleName,
				Bindings:        bindings,
			})
		}

		record := &UserPermissionRecord{
			Subject:     userName,
			Permissions: permissions,
		}

		_, exists, _ := c.userPermissionStore.GetByKey(userName)
		if exists {
			c.userPermissionStore.Update(record)
		} else {
			c.userPermissionStore.Add(record)
		}
	}

	for groupName, roleBindings := range groupPermissions {
		permissions := make([]PermissionInfo, 0, len(roleBindings))
		for roleName, bindings := range roleBindings {
			permissions = append(permissions, PermissionInfo{
				ClusterRoleName: roleName,
				Bindings:        bindings,
			})
		}

		record := &UserPermissionRecord{
			Subject:     groupName,
			Permissions: permissions,
		}

		_, exists, _ := c.groupPermissionStore.GetByKey(groupName)
		if exists {
			c.groupPermissionStore.Update(record)
		} else {
			c.groupPermissionStore.Add(record)
		}
	}

	// Update the resource version hash after successful synchronization
	c.resourceVersionHash = newHash

	klog.V(2).Infof("UserPermissionCache synchronized: %d users, %d groups, %d discoverable roles",
		len(userPermissions), len(groupPermissions), len(c.discoverableRoles))
}

// getDiscoverableClusterRoles returns all ClusterRoles with the discoverable label
func (c *UserPermissionCache) getDiscoverableClusterRoles() ([]*rbacv1.ClusterRole, error) {
	allRoles, err := c.clusterRoleLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var discoverableRoles []*rbacv1.ClusterRole
	for _, role := range allRoles {
		if role.Labels != nil && role.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] == "true" {
			discoverableRoles = append(discoverableRoles, role)
		}
	}

	return discoverableRoles, nil
}

// getPermissionRecords returns the aggregated permissions from user and groups
func (c *UserPermissionCache) getPermissionRecords(userInfo user.Info) map[string][]clusterviewv1alpha1.ClusterBinding {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// Map of roleName -> []ClusterBinding
	roleBindings := make(map[string][]clusterviewv1alpha1.ClusterBinding)
	userName := userInfo.GetName()
	groups := userInfo.GetGroups()

	// Get permissions from user
	obj, exists, _ := c.userPermissionStore.GetByKey(userName)
	if exists {
		record := obj.(*UserPermissionRecord)
		for _, perm := range record.Permissions {
			for _, binding := range perm.Bindings {
				roleBindings[perm.ClusterRoleName] = mergeOrAppendBinding(roleBindings[perm.ClusterRoleName], binding)
			}
		}
	}

	// Get permissions from groups
	for _, group := range groups {
		obj, exists, _ := c.groupPermissionStore.GetByKey(group)
		if exists {
			record := obj.(*UserPermissionRecord)
			for _, perm := range record.Permissions {
				for _, binding := range perm.Bindings {
					roleBindings[perm.ClusterRoleName] = mergeOrAppendBinding(roleBindings[perm.ClusterRoleName], binding)
				}
			}
		}
	}

	return roleBindings
}

// List returns the list of UserPermissions for a user
func (c *UserPermissionCache) List(userInfo user.Info, selector labels.Selector) (*clusterviewv1alpha1.UserPermissionList, error) {
	roleBindings := c.getPermissionRecords(userInfo)

	userPermissionList := &clusterviewv1alpha1.UserPermissionList{
		Items: make([]clusterviewv1alpha1.UserPermission, 0, len(roleBindings)),
	}

	c.lock.RLock()
	discoverableRoles := c.discoverableRoles
	c.lock.RUnlock()

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
func (c *UserPermissionCache) Get(userInfo user.Info, name string) (*clusterviewv1alpha1.UserPermission, error) {
	roleBindings := c.getPermissionRecords(userInfo)

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
		c.lock.RLock()
		discoverableRoles := c.discoverableRoles
		c.lock.RUnlock()

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
func (c *UserPermissionCache) getSyntheticAdminRoleDefinition() clusterviewv1alpha1.ClusterRoleDefinition {
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
func (c *UserPermissionCache) getSyntheticViewRoleDefinition() clusterviewv1alpha1.ClusterRoleDefinition {
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

// Helper functions

// addPermissionForUser adds or merges a permission binding for a user
func addPermissionForUser(userPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding, userName, roleName string, binding clusterviewv1alpha1.ClusterBinding) {
	if userPermissions[userName] == nil {
		userPermissions[userName] = make(map[string][]clusterviewv1alpha1.ClusterBinding)
	}
	userPermissions[userName][roleName] = mergeOrAppendBinding(userPermissions[userName][roleName], binding)
}

// addPermissionForGroup adds or merges a permission binding for a group
func addPermissionForGroup(groupPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding, groupName, roleName string, binding clusterviewv1alpha1.ClusterBinding) {
	if groupPermissions[groupName] == nil {
		groupPermissions[groupName] = make(map[string][]clusterviewv1alpha1.ClusterBinding)
	}
	groupPermissions[groupName][roleName] = mergeOrAppendBinding(groupPermissions[groupName][roleName], binding)
}

// mergeOrAppendBinding merges or appends a binding to the existing bindings list
// - If the same cluster exists with cluster scope, do nothing (already covered)
// - If the same cluster exists with namespace scope, merge the namespaces
// - If different cluster, append as new binding
func mergeOrAppendBinding(existingBindings []clusterviewv1alpha1.ClusterBinding, newBinding clusterviewv1alpha1.ClusterBinding) []clusterviewv1alpha1.ClusterBinding {
	for i, existing := range existingBindings {
		if existing.Cluster == newBinding.Cluster {
			// Same cluster found
			if existing.Scope == clusterviewv1alpha1.BindingScopeCluster {
				// Cluster scope already covers everything, no need to add
				return existingBindings
			}
			if newBinding.Scope == clusterviewv1alpha1.BindingScopeCluster {
				// New binding is cluster scope, replace the namespace-scoped one
				existingBindings[i] = newBinding
				return existingBindings
			}
			// Both are namespace-scoped, merge namespaces
			if existing.Scope == clusterviewv1alpha1.BindingScopeNamespace && newBinding.Scope == clusterviewv1alpha1.BindingScopeNamespace {
				// Deduplicate namespaces
				namespaceSet := sets.New(existing.Namespaces...)
				namespaceSet.Insert(newBinding.Namespaces...)
				existingBindings[i].Namespaces = namespaceSet.UnsortedList()
				return existingBindings
			}
		}
	}
	// Different cluster, append
	return append(existingBindings, newBinding)
}

// processClusterRoleBinding processes a ClusterRoleBinding and adds permissions to the maps
func (c *UserPermissionCache) processClusterRoleBinding(
	binding *clusterpermissionv1alpha1.ClusterRoleBinding,
	clusterName string,
	userPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
	groupPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
) {
	if binding == nil || binding.RoleRef.Name == "" {
		return
	}

	roleRefName := binding.RoleRef.Name
	if !c.discoverableRolesNames.Has(roleRefName) {
		return
	}

	clusterBinding := clusterviewv1alpha1.ClusterBinding{
		Cluster:    clusterName,
		Scope:      clusterviewv1alpha1.BindingScopeCluster,
		Namespaces: []string{"*"},
	}

	for _, subject := range binding.Subjects {
		switch subject.Kind {
		case rbacv1.UserKind:
			addPermissionForUser(userPermissions, subject.Name, roleRefName, clusterBinding)
		case rbacv1.GroupKind:
			addPermissionForGroup(groupPermissions, subject.Name, roleRefName, clusterBinding)
		}
	}
}

// processRoleBinding processes a RoleBinding and adds permissions to the maps
func (c *UserPermissionCache) processRoleBinding(
	binding *clusterpermissionv1alpha1.RoleBinding,
	clusterName string,
	userPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
	groupPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
) {
	if binding == nil || binding.RoleRef.Name == "" {
		return
	}

	roleRefName := binding.RoleRef.Name
	if !c.discoverableRolesNames.Has(roleRefName) {
		return
	}

	namespace := binding.Namespace

	namespaceBinding := clusterviewv1alpha1.ClusterBinding{
		Cluster:    clusterName,
		Scope:      clusterviewv1alpha1.BindingScopeNamespace,
		Namespaces: []string{namespace},
	}

	for _, subject := range binding.Subjects {
		switch subject.Kind {
		case rbacv1.UserKind:
			addPermissionForUser(userPermissions, subject.Name, roleRefName, namespaceBinding)
		case rbacv1.GroupKind:
			addPermissionForGroup(groupPermissions, subject.Name, roleRefName, namespaceBinding)
		}
	}
}

// processManagedClusterPermissions scans RoleBindings and ClusterRoleBindings to determine
// which users/groups have managedcluster admin/view permissions
func (c *UserPermissionCache) processManagedClusterPermissions(
	userPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
	groupPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
) {
	// Get all managed clusters to know which namespaces to check
	managedClusters, err := c.managedClusterLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ManagedClusters for managedcluster permission check: %v", err)
		return
	}

	// Build a set of managed cluster names (these are also namespace names)
	clusterNames := sets.New[string]()
	for _, cluster := range managedClusters {
		clusterNames.Insert(cluster.Name)
	}

	// Process ClusterRoleBindings for cluster-wide permissions
	c.processClusterRoleBindingsForManagedCluster(clusterNames, userPermissions, groupPermissions)

	// Process RoleBindings for namespace-specific permissions
	c.processRoleBindingsForManagedCluster(clusterNames, userPermissions, groupPermissions)
}

// processClusterRoleBindingsForManagedCluster checks ClusterRoleBindings that grant permissions
// to create managedclusteractions/managedclusterviews
func (c *UserPermissionCache) processClusterRoleBindingsForManagedCluster(
	clusterNames sets.Set[string],
	userPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
	groupPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
) {
	clusterRoleBindings, err := c.clusterRoleBindingLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ClusterRoleBindings: %v", err)
		return
	}

	for _, crb := range clusterRoleBindings {
		// Get the ClusterRole referenced by this binding
		clusterRole, err := c.clusterRoleLister.Get(crb.RoleRef.Name)
		if err != nil {
			continue
		}

		// Check if this ClusterRole grants create on managedclusteractions or managedclusterviews
		grantsAdminPerm := c.clusterRoleGrantsPermission(clusterRole, "managedclusteractions", "action.open-cluster-management.io")
		grantsViewPerm := c.clusterRoleGrantsPermission(clusterRole, "managedclusterviews", "view.open-cluster-management.io")

		if !grantsAdminPerm && !grantsViewPerm {
			continue
		}

		// Extract users and groups from the binding
		users, groups := authorizationutil.RBACSubjectsToUsersAndGroups(crb.Subjects, "")

		// For each managed cluster, add synthetic permissions
		for clusterName := range clusterNames {
			binding := clusterviewv1alpha1.ClusterBinding{
				Cluster:    clusterName,
				Scope:      clusterviewv1alpha1.BindingScopeCluster,
				Namespaces: []string{"*"},
			}

			if grantsAdminPerm {
				for _, userName := range users {
					addPermissionForUser(userPermissions, userName, clusterviewv1alpha1.ManagedClusterAdminRole, binding)
				}
				for _, groupName := range groups {
					addPermissionForGroup(groupPermissions, groupName, clusterviewv1alpha1.ManagedClusterAdminRole, binding)
				}
			}

			if grantsViewPerm {
				for _, userName := range users {
					addPermissionForUser(userPermissions, userName, clusterviewv1alpha1.ManagedClusterViewRole, binding)
				}
				for _, groupName := range groups {
					addPermissionForGroup(groupPermissions, groupName, clusterviewv1alpha1.ManagedClusterViewRole, binding)
				}
			}
		}
	}
}

// processRoleBindingsForManagedCluster checks RoleBindings in managed cluster namespaces
// that grant permissions to create managedclusteractions/managedclusterviews
func (c *UserPermissionCache) processRoleBindingsForManagedCluster(
	clusterNames sets.Set[string],
	userPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
	groupPermissions map[string]map[string][]clusterviewv1alpha1.ClusterBinding,
) {
	// For each managed cluster namespace, check RoleBindings
	for clusterName := range clusterNames {
		roleBindings, err := c.roleBindingLister.RoleBindings(clusterName).List(labels.Everything())
		if err != nil {
			klog.V(4).Infof("Failed to list RoleBindings in namespace %s: %v", clusterName, err)
			continue
		}

		for _, rb := range roleBindings {
			// Get the Role or ClusterRole referenced by this binding
			var grantsAdminPerm, grantsViewPerm bool

			if rb.RoleRef.Kind == "ClusterRole" {
				clusterRole, err := c.clusterRoleLister.Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				grantsAdminPerm = c.clusterRoleGrantsPermission(clusterRole, "managedclusteractions", "action.open-cluster-management.io")
				grantsViewPerm = c.clusterRoleGrantsPermission(clusterRole, "managedclusterviews", "view.open-cluster-management.io")
			} else if rb.RoleRef.Kind == "Role" {
				role, err := c.roleLister.Roles(clusterName).Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				// For Role, we need to check the rules directly
				grantsAdminPerm = c.roleGrantsPermission(role, "managedclusteractions", "action.open-cluster-management.io")
				grantsViewPerm = c.roleGrantsPermission(role, "managedclusterviews", "view.open-cluster-management.io")
			}

			if !grantsAdminPerm && !grantsViewPerm {
				continue
			}

			// Extract users and groups from the binding
			users, groups := authorizationutil.RBACSubjectsToUsersAndGroups(rb.Subjects, rb.Namespace)

			binding := clusterviewv1alpha1.ClusterBinding{
				Cluster:    clusterName,
				Scope:      clusterviewv1alpha1.BindingScopeCluster,
				Namespaces: []string{"*"},
			}

			if grantsAdminPerm {
				for _, userName := range users {
					addPermissionForUser(userPermissions, userName, clusterviewv1alpha1.ManagedClusterAdminRole, binding)
				}
				for _, groupName := range groups {
					addPermissionForGroup(groupPermissions, groupName, clusterviewv1alpha1.ManagedClusterAdminRole, binding)
				}
			}

			if grantsViewPerm {
				for _, userName := range users {
					addPermissionForUser(userPermissions, userName, clusterviewv1alpha1.ManagedClusterViewRole, binding)
				}
				for _, groupName := range groups {
					addPermissionForGroup(groupPermissions, groupName, clusterviewv1alpha1.ManagedClusterViewRole, binding)
				}
			}
		}
	}
}

// rulesGrantPermission checks if policy rules grant create permission on a specific resource
func (c *UserPermissionCache) rulesGrantPermission(rules []rbacv1.PolicyRule, resource, apiGroup string) bool {
	for _, rule := range rules {
		// Check if the rule applies to the correct API group
		if !c.ruleCoversAPIGroup(rule, apiGroup) {
			continue
		}

		// Check if the rule covers the resource
		if !c.ruleCoversResource(rule, resource) {
			continue
		}

		// Check if the rule grants create verb
		if c.ruleCoversVerb(rule, "create") {
			return true
		}
	}
	return false
}

// clusterRoleGrantsPermission checks if a ClusterRole grants create permission on a specific resource
func (c *UserPermissionCache) clusterRoleGrantsPermission(clusterRole *rbacv1.ClusterRole, resource, apiGroup string) bool {
	return c.rulesGrantPermission(clusterRole.Rules, resource, apiGroup)
}

// roleGrantsPermission checks if a Role grants create permission on a specific resource
func (c *UserPermissionCache) roleGrantsPermission(role *rbacv1.Role, resource, apiGroup string) bool {
	return c.rulesGrantPermission(role.Rules, resource, apiGroup)
}

// ruleCoversAPIGroup checks if a policy rule covers the specified API group
func (c *UserPermissionCache) ruleCoversAPIGroup(rule rbacv1.PolicyRule, apiGroup string) bool {
	for _, ruleAPIGroup := range rule.APIGroups {
		if ruleAPIGroup == "*" || ruleAPIGroup == apiGroup {
			return true
		}
	}
	return false
}

// ruleCoversResource checks if a policy rule covers the specified resource
func (c *UserPermissionCache) ruleCoversResource(rule rbacv1.PolicyRule, resource string) bool {
	for _, ruleResource := range rule.Resources {
		if ruleResource == "*" || ruleResource == resource {
			return true
		}
	}
	return false
}

// ruleCoversVerb checks if a policy rule covers the specified verb
func (c *UserPermissionCache) ruleCoversVerb(rule rbacv1.PolicyRule, verb string) bool {
	for _, ruleVerb := range rule.Verbs {
		if ruleVerb == "*" || ruleVerb == verb {
			return true
		}
	}
	return false
}

// ListObjects returns the list of UserPermissions as runtime.Object
func (c *UserPermissionCache) ListObjects(userInfo user.Info) (runtime.Object, error) {
	return c.List(userInfo, labels.Everything())
}

// ConvertResource converts a role name to a UserPermission object
func (c *UserPermissionCache) ConvertResource(name string) runtime.Object {
	return &clusterviewv1alpha1.UserPermission{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
