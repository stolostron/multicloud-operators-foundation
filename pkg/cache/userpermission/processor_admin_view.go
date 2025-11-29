package userpermission

import (
	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
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

// process implements permissionProcessor for adminViewPermissionProcessor
// Scans RoleBindings and ClusterRoleBindings to determine which users/groups have managedcluster admin/view permissions
func (p *adminViewPermissionProcessor) process(store *permissionStore) error {
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
		grantsAdminPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterActionsResource, ActionAPIGroup)
		grantsViewPerm := clusterRoleGrantsPermission(
			clusterRole, ManagedClusterViewsResource, ViewAPIGroup)

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
			var grantsAdminPerm, grantsViewPerm bool

			switch rb.RoleRef.Kind {
			case "ClusterRole":
				clusterRole, err := p.clusterRoleLister.Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				grantsAdminPerm = clusterRoleGrantsPermission(
					clusterRole, ManagedClusterActionsResource, ActionAPIGroup)
				grantsViewPerm = clusterRoleGrantsPermission(
					clusterRole, ManagedClusterViewsResource, ViewAPIGroup)
			case "Role":
				role, err := p.roleLister.Roles(clusterName).Get(rb.RoleRef.Name)
				if err != nil {
					continue
				}
				// For Role, we need to check the rules directly
				grantsAdminPerm = roleGrantsPermission(role, ManagedClusterActionsResource, ActionAPIGroup)
				grantsViewPerm = roleGrantsPermission(role, ManagedClusterViewsResource, ViewAPIGroup)
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
			addSyntheticPermissions(
				users, groups, grantsAdminPerm, grantsViewPerm, binding, store)
		}
	}
}
