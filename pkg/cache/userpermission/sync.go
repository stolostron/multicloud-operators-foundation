package userpermission

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

// permissionProcessor processes permissions and populates the permission store
type permissionProcessor interface {
	// process adds permissions to the permission store
	process(store *permissionStore) error
}

// synchronize runs a full synchronization of the cache
func (c *Cache) synchronize() {
	startTime := time.Now()
	defer func() {
		klog.V(2).Infof("synchronize took %v", time.Since(startTime))
	}()

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
		klog.V(2).Infof(
			"Resource changes detected (hash changed from %s to %s), synchronizing cache",
			c.resourceVersionHash[:8], newHash[:8])
	}

	// Build permission store
	store := newPermissionStore()

	// Define permission processors in order
	processors := []permissionProcessor{
		&adminViewPermissionProcessor{
			managedClusterLister:     c.managedClusterLister,
			clusterRoleLister:        c.clusterRoleLister,
			clusterRoleBindingLister: c.clusterRoleBindingLister,
			roleLister:               c.roleLister,
			roleBindingLister:        c.roleBindingLister,
		},
		&discoverablePermissionProcessor{
			clusterRoleLister:       c.clusterRoleLister,
			clusterPermissionLister: c.clusterPermissionLister,
		},
	}

	// Process permissions using all processors
	for _, processor := range processors {
		if err := processor.process(store); err != nil {
			klog.Errorf("Failed to process permissions: %v", err)
			return
		}
	}

	// Replace the cache stores with the new ones
	c.permissionStore = store

	// Update the resource version hash after successful synchronization
	c.resourceVersionHash = newHash

	klog.V(2).Infof("UserPermissionCache synchronized: %d users, %d groups, %d discoverable roles",
		len(store.userStore.List()), len(store.groupStore.List()), len(store.getDiscoverableRoles()))
}

// calculateResourceVersionHash computes a hash of relevant resource versions
// Only includes resources that affect user permissions:
// 1. Discoverable ClusterRoles
// 2. ClusterPermissions
// 3. ClusterRoleBindings and their referenced ClusterRoles (that grant managedcluster permissions)
// 4. RoleBindings and their referenced Roles/ClusterRoles (that grant managedcluster permissions)
func (c *Cache) calculateResourceVersionHash() (string, error) {
	startTime := time.Now()
	defer func() {
		klog.V(2).Infof("calculateResourceVersionHash took %v", time.Since(startTime))
	}()

	h := sha256.New()
	var versions []string

	// 1. Discoverable ClusterRoles
	if err := c.addDiscoverableClusterRoleVersions(&versions); err != nil {
		return "", err
	}

	// 2. ClusterPermissions
	if err := c.addClusterPermissionVersions(&versions); err != nil {
		return "", err
	}

	// Track ClusterRoles/Roles that grant managedcluster permissions (to avoid duplicates)
	trackedClusterRoles := sets.New[string]()
	trackedRoles := sets.New[string]()

	// 3. ClusterRoleBindings that grant managedcluster permissions
	if err := c.addClusterRoleBindingVersions(&versions, trackedClusterRoles); err != nil {
		return "", err
	}

	// 4. RoleBindings that grant managedcluster permissions (only in ManagedCluster namespaces)
	c.addRoleBindingVersions(&versions, trackedClusterRoles, trackedRoles)

	// Sort for deterministic hashing
	sort.Strings(versions)

	// Write all versions to hash
	for _, v := range versions {
		_, _ = h.Write([]byte(v))
		_, _ = h.Write([]byte("\n"))
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// addDiscoverableClusterRoleVersions adds discoverable ClusterRole versions to the list
func (c *Cache) addDiscoverableClusterRoleVersions(versions *[]string) error {
	clusterRoles, err := c.clusterRoleLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list ClusterRoles: %w", err)
	}
	for _, cr := range clusterRoles {
		if cr.Labels != nil && cr.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] == "true" {
			*versions = append(*versions, fmt.Sprintf(ClusterRoleVersionFormat, cr.Name, cr.ResourceVersion))
		}
	}
	return nil
}

// addClusterPermissionVersions adds ClusterPermission versions to the list
func (c *Cache) addClusterPermissionVersions(versions *[]string) error {
	clusterPermissions, err := c.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list ClusterPermissions: %w", err)
	}
	for _, cp := range clusterPermissions {
		*versions = append(*versions, fmt.Sprintf(ClusterPermissionFormat, cp.Namespace, cp.ResourceVersion))
	}
	return nil
}

// addClusterRoleBindingVersions adds ClusterRoleBinding versions that grant managedcluster permissions
func (c *Cache) addClusterRoleBindingVersions(
	versions *[]string,
	trackedClusterRoles sets.Set[string],
) error {
	clusterRoleBindings, err := c.clusterRoleBindingLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}
	for _, crb := range clusterRoleBindings {
		clusterRole, err := c.clusterRoleLister.Get(crb.RoleRef.Name)
		if err != nil {
			klog.V(4).Infof("Failed to get ClusterRole %s: %v", crb.RoleRef.Name, err)
			continue
		}

		grantsAdminPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterActionsResource, ActionAPIGroup)
		grantsViewPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterViewsResource, ViewAPIGroup)
		if !grantsAdminPerm && !grantsViewPerm {
			continue
		}

		// Track the ClusterRoleBinding and ClusterRole
		*versions = append(*versions, fmt.Sprintf(ClusterRoleBindingFormat, crb.Name, crb.ResourceVersion))
		c.trackClusterRoleVersion(versions, clusterRole, trackedClusterRoles)
	}
	return nil
}

// trackClusterRoleVersion tracks a ClusterRole version if needed
func (c *Cache) trackClusterRoleVersion(
	versions *[]string,
	clusterRole *rbacv1.ClusterRole,
	trackedClusterRoles sets.Set[string],
) {
	if trackedClusterRoles.Has(clusterRole.Name) {
		return
	}
	if clusterRole.Labels != nil && clusterRole.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] == "true" {
		return
	}
	*versions = append(*versions, fmt.Sprintf(ClusterRoleVersionFormat, clusterRole.Name, clusterRole.ResourceVersion))
	trackedClusterRoles.Insert(clusterRole.Name)
}

// addRoleBindingVersions adds RoleBinding versions that grant managedcluster permissions
// Only processes RoleBindings in namespaces that correspond to ManagedClusters
func (c *Cache) addRoleBindingVersions(
	versions *[]string,
	trackedClusterRoles sets.Set[string],
	trackedRoles sets.Set[string],
) {
	// List all RoleBindings across all namespaces
	roleBindings, err := c.roleBindingLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list RoleBindings: %v", err)
		return
	}

	for _, rb := range roleBindings {
		// Check if this RoleBinding's namespace corresponds to a ManagedCluster
		_, err := c.managedClusterLister.Get(rb.Namespace)
		if err != nil {
			// Not a ManagedCluster namespace, skip
			continue
		}

		c.processRoleBindingVersion(versions, rb, rb.Namespace, trackedClusterRoles, trackedRoles)
	}
}

// processRoleBindingVersion processes a single RoleBinding
func (c *Cache) processRoleBindingVersion(
	versions *[]string,
	rb *rbacv1.RoleBinding,
	clusterName string,
	trackedClusterRoles sets.Set[string],
	trackedRoles sets.Set[string],
) {
	var grantsAdminPerm, grantsViewPerm bool

	switch rb.RoleRef.Kind {
	case "ClusterRole":
		grantsAdminPerm, grantsViewPerm = c.checkClusterRoleBindingPermissions(
			versions, rb.RoleRef.Name, trackedClusterRoles)
	case "Role":
		grantsAdminPerm, grantsViewPerm = c.checkRoleBindingPermissions(
			versions, rb.RoleRef.Name, clusterName, trackedRoles)
	}

	if grantsAdminPerm || grantsViewPerm {
		*versions = append(*versions, fmt.Sprintf(RoleBindingFormat, rb.Namespace, rb.Name, rb.ResourceVersion))
	}
}

// checkClusterRoleBindingPermissions checks if a ClusterRole grants permissions
func (c *Cache) checkClusterRoleBindingPermissions(
	versions *[]string,
	roleName string,
	trackedClusterRoles sets.Set[string],
) (bool, bool) {
	clusterRole, err := c.clusterRoleLister.Get(roleName)
	if err != nil {
		klog.V(4).Infof("Failed to get ClusterRole %s: %v", roleName, err)
		return false, false
	}

	grantsAdminPerm, grantsViewPerm := checkRulesForPermissions(clusterRole.Rules)
	if grantsAdminPerm || grantsViewPerm {
		c.trackClusterRoleVersion(versions, clusterRole, trackedClusterRoles)
	}
	return grantsAdminPerm, grantsViewPerm
}

// checkRoleBindingPermissions checks if a Role grants permissions
func (c *Cache) checkRoleBindingPermissions(
	versions *[]string,
	roleName string,
	clusterName string,
	trackedRoles sets.Set[string],
) (bool, bool) {
	role, err := c.roleLister.Roles(clusterName).Get(roleName)
	if err != nil {
		klog.V(4).Infof("Failed to get Role %s in namespace %s: %v", roleName, clusterName, err)
		return false, false
	}

	grantsAdminPerm, grantsViewPerm := checkRulesForPermissions(role.Rules)
	if grantsAdminPerm || grantsViewPerm {
		roleKey := fmt.Sprintf("%s/%s", role.Namespace, role.Name)
		if !trackedRoles.Has(roleKey) {
			*versions = append(*versions, fmt.Sprintf(RoleFormat, roleKey, role.ResourceVersion))
			trackedRoles.Insert(roleKey)
		}
	}
	return grantsAdminPerm, grantsViewPerm
}

// checkRulesForPermissions checks if policy rules grant admin/view permissions
func checkRulesForPermissions(rules []rbacv1.PolicyRule) (grantsAdminPerm, grantsViewPerm bool) {
	grantsAdminPerm = rulesGrantPermission(rules, ManagedClusterActionsResource, ActionAPIGroup)
	grantsViewPerm = rulesGrantPermission(rules, ManagedClusterViewsResource, ViewAPIGroup)
	return grantsAdminPerm, grantsViewPerm
}
