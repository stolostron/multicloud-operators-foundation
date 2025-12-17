package userpermission

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/klog/v2"
	clusterv1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

// adminViewPermissionProcessor processes synthetic admin/view permissions from RBAC bindings
type adminViewPermissionProcessor struct {
	managedClusterLister     clusterv1lister.ManagedClusterLister
	clusterRoleLister        rbacv1listers.ClusterRoleLister
	clusterRoleBindingLister rbacv1listers.ClusterRoleBindingLister
	roleLister               rbacv1listers.RoleLister
	roleBindingLister        rbacv1listers.RoleBindingLister
}

// sync implements permissionProcessor for adminViewPermissionProcessor
// Scans RoleBindings and ClusterRoleBindings to determine which users/groups have managedcluster admin/view permissions
func (p *adminViewPermissionProcessor) sync(store *permissionStore) error {
	// Get all managed clusters to know which namespaces to check
	managedClusters, err := p.managedClusterLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ManagedClusters for managedcluster permission check: %v", err)
		return nil
	}

	// Build a set of managed cluster names (these are also namespace names)
	clusterNames := sets.New[string]()
	for _, cluster := range managedClusters {
		clusterNames.Insert(cluster.Name)
	}

	// Process ClusterRoleBindings for cluster-wide permissions
	p.processClusterRoleBindings(clusterNames, store)

	// Process RoleBindings for namespace-specific permissions
	p.processRoleBindings(clusterNames, store)

	return nil
}

// addSyntheticPermissions adds synthetic managedcluster permissions for users and groups
// Admin permission is only granted when the user has BOTH managedclusteraction create AND
// managedclusterview create permissions (grantsAdminPerm = hasActionPerm && hasViewPerm).
// When admin is granted, view permission is not added since admin is a superset of view.
func addSyntheticPermissions(
	users, groups []string,
	grantsAdminPerm, grantsViewPerm bool,
	binding clusterviewv1alpha1.ClusterBinding,
	store *permissionStore,
) {
	if grantsAdminPerm {
		for _, userName := range users {
			store.addPermissionForUser(userName, clusterviewv1alpha1.ManagedClusterAdminRole, binding)
		}
		for _, groupName := range groups {
			store.addPermissionForGroup(groupName, clusterviewv1alpha1.ManagedClusterAdminRole, binding)
		}
		// Admin permission is a superset of view, so skip adding view permission
		return
	}

	if grantsViewPerm {
		for _, userName := range users {
			store.addPermissionForUser(userName, clusterviewv1alpha1.ManagedClusterViewRole, binding)
		}
		for _, groupName := range groups {
			store.addPermissionForGroup(groupName, clusterviewv1alpha1.ManagedClusterViewRole, binding)
		}
	}
}

// processClusterRoleBindings checks ClusterRoleBindings that grant permissions
// to create managedclusteractions/managedclusterviews
func (p *adminViewPermissionProcessor) processClusterRoleBindings(
	clusterNames sets.Set[string],
	store *permissionStore,
) {
	clusterRoleBindings, err := p.clusterRoleBindingLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list ClusterRoleBindings: %v", err)
		return
	}

	for _, crb := range clusterRoleBindings {
		// Get the ClusterRole referenced by this binding
		clusterRole, err := p.clusterRoleLister.Get(crb.RoleRef.Name)
		if err != nil {
			continue
		}

		// Check if this ClusterRole grants create on managedclusteractions or managedclusterviews
		hasActionPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterActionsResource, ActionAPIGroup)
		hasViewPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterViewsResource, ViewAPIGroup)

		// Admin permission requires BOTH action AND view create permissions
		grantsAdminPerm := hasActionPerm && hasViewPerm
		grantsViewPerm := hasViewPerm

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
			addSyntheticPermissions(
				users, groups, grantsAdminPerm, grantsViewPerm, binding, store)
		}
	}
}

// processRoleBindings checks RoleBindings in managed cluster namespaces
// that grant permissions to create managedclusteractions/managedclusterviews
func (p *adminViewPermissionProcessor) processRoleBindings(
	clusterNames sets.Set[string],
	store *permissionStore,
) {
	// For each managed cluster namespace, check RoleBindings
	for clusterName := range clusterNames {
		roleBindings, err := p.roleBindingLister.RoleBindings(clusterName).List(labels.Everything())
		if err != nil {
			klog.V(4).Infof("Failed to list RoleBindings in namespace %s: %v", clusterName, err)
			continue
		}

		for _, rb := range roleBindings {
			// Get the Role or ClusterRole referenced by this binding
			var hasActionPerm, hasViewPerm bool

			switch rb.RoleRef.Kind {
			case "ClusterRole":
				clusterRole, err := p.clusterRoleLister.Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				hasActionPerm = clusterRoleGrantsPermission(
					clusterRole, ManagedClusterActionsResource, ActionAPIGroup)
				hasViewPerm = clusterRoleGrantsPermission(
					clusterRole, ManagedClusterViewsResource, ViewAPIGroup)
			case "Role":
				role, err := p.roleLister.Roles(clusterName).Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				// For Role, we need to check the rules directly
				hasActionPerm = roleGrantsPermission(role, ManagedClusterActionsResource, ActionAPIGroup)
				hasViewPerm = roleGrantsPermission(role, ManagedClusterViewsResource, ViewAPIGroup)
			}

			// Admin permission requires BOTH action AND view create permissions
			grantsAdminPerm := hasActionPerm && hasViewPerm
			grantsViewPerm := hasViewPerm

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
			addSyntheticPermissions(
				users, groups, grantsAdminPerm, grantsViewPerm, binding, store)
		}
	}
}

// getResourceVersionHash implements permissionProcessor for adminViewPermissionProcessor
// Computes a hash of resources relevant to admin/view synthetic permissions:
//   - ClusterRoleBindings and their referenced ClusterRoles (that grant managedcluster permissions)
//   - RoleBindings and their referenced Roles/ClusterRoles (that grant managedcluster permissions,
//     only in ManagedCluster namespaces)
func (p *adminViewPermissionProcessor) getResourceVersionHash() (string, error) {
	h := sha256.New()
	var versions []string

	// Track ClusterRoles/Roles that grant managedcluster permissions (to avoid duplicates)
	trackedClusterRoles := sets.New[string]()
	trackedRoles := sets.New[string]()

	// 1. ClusterRoleBindings that grant managedcluster permissions
	if err := p.addClusterRoleBindingVersions(&versions, trackedClusterRoles); err != nil {
		return "", err
	}

	// 2. RoleBindings that grant managedcluster permissions (only in ManagedCluster namespaces)
	p.addRoleBindingVersions(&versions, trackedClusterRoles, trackedRoles)

	// Sort for deterministic hashing
	sort.Strings(versions)

	// Write all versions to hash
	for _, v := range versions {
		_, _ = h.Write([]byte(v))
		_, _ = h.Write([]byte("\n"))
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// addClusterRoleBindingVersions adds ClusterRoleBinding versions that grant managedcluster permissions
func (p *adminViewPermissionProcessor) addClusterRoleBindingVersions(
	versions *[]string,
	trackedClusterRoles sets.Set[string],
) error {
	clusterRoleBindings, err := p.clusterRoleBindingLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}
	for _, crb := range clusterRoleBindings {
		clusterRole, err := p.clusterRoleLister.Get(crb.RoleRef.Name)
		if err != nil {
			klog.V(4).Infof("Failed to get ClusterRole %s: %v", crb.RoleRef.Name, err)
			continue
		}

		hasActionPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterActionsResource, ActionAPIGroup)
		hasViewPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterViewsResource, ViewAPIGroup)

		// Admin permission requires BOTH action AND view create permissions
		grantsAdminPerm := hasActionPerm && hasViewPerm
		grantsViewPerm := hasViewPerm

		if !grantsAdminPerm && !grantsViewPerm {
			continue
		}

		// Track the ClusterRoleBinding and ClusterRole
		*versions = append(*versions, fmt.Sprintf(ClusterRoleBindingFormat, crb.Name, crb.ResourceVersion))
		p.trackClusterRoleVersion(versions, clusterRole, trackedClusterRoles)
	}
	return nil
}

// trackClusterRoleVersion tracks a ClusterRole version if needed
func (p *adminViewPermissionProcessor) trackClusterRoleVersion(
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
func (p *adminViewPermissionProcessor) addRoleBindingVersions(
	versions *[]string,
	trackedClusterRoles sets.Set[string],
	trackedRoles sets.Set[string],
) {
	// List all RoleBindings across all namespaces
	roleBindings, err := p.roleBindingLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("Failed to list RoleBindings: %v", err)
		return
	}

	for _, rb := range roleBindings {
		// Check if this RoleBinding's namespace corresponds to a ManagedCluster
		_, err := p.managedClusterLister.Get(rb.Namespace)
		if err != nil {
			// Not a ManagedCluster namespace, skip
			continue
		}

		p.processRoleBindingVersion(versions, rb, rb.Namespace, trackedClusterRoles, trackedRoles)
	}
}

// processRoleBindingVersion processes a single RoleBinding for version tracking
func (p *adminViewPermissionProcessor) processRoleBindingVersion(
	versions *[]string,
	rb *rbacv1.RoleBinding,
	clusterName string,
	trackedClusterRoles sets.Set[string],
	trackedRoles sets.Set[string],
) {
	var grantsAdminPerm, grantsViewPerm bool

	switch rb.RoleRef.Kind {
	case "ClusterRole":
		grantsAdminPerm, grantsViewPerm = p.checkClusterRoleBindingPermissions(
			versions, rb.RoleRef.Name, trackedClusterRoles)
	case "Role":
		grantsAdminPerm, grantsViewPerm = p.checkRoleBindingPermissions(
			versions, rb.RoleRef.Name, clusterName, trackedRoles)
	}

	if grantsAdminPerm || grantsViewPerm {
		*versions = append(*versions, fmt.Sprintf(RoleBindingFormat, rb.Namespace, rb.Name, rb.ResourceVersion))
	}
}

// checkClusterRoleBindingPermissions checks if a ClusterRole grants permissions
func (p *adminViewPermissionProcessor) checkClusterRoleBindingPermissions(
	versions *[]string,
	roleName string,
	trackedClusterRoles sets.Set[string],
) (bool, bool) {
	clusterRole, err := p.clusterRoleLister.Get(roleName)
	if err != nil {
		klog.V(4).Infof("Failed to get ClusterRole %s: %v", roleName, err)
		return false, false
	}

	grantsAdminPerm, grantsViewPerm := checkRulesForPermissions(clusterRole.Rules)
	if grantsAdminPerm || grantsViewPerm {
		p.trackClusterRoleVersion(versions, clusterRole, trackedClusterRoles)
	}
	return grantsAdminPerm, grantsViewPerm
}

// checkRoleBindingPermissions checks if a Role grants permissions
func (p *adminViewPermissionProcessor) checkRoleBindingPermissions(
	versions *[]string,
	roleName string,
	clusterName string,
	trackedRoles sets.Set[string],
) (bool, bool) {
	role, err := p.roleLister.Roles(clusterName).Get(roleName)
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
// Admin permission requires BOTH action AND view create permissions
func checkRulesForPermissions(rules []rbacv1.PolicyRule) (grantsAdminPerm, grantsViewPerm bool) {
	hasActionPerm := rulesGrantPermission(rules, ManagedClusterActionsResource, ActionAPIGroup)
	hasViewPerm := rulesGrantPermission(rules, ManagedClusterViewsResource, ViewAPIGroup)
	return hasActionPerm && hasViewPerm, hasViewPerm
}
