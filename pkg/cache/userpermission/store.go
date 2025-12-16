package userpermission

import (
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	kubecache "k8s.io/client-go/tools/cache"
)

// permissionStore manages user and group permissions together using kubecache.Store
type permissionStore struct {
	userStore             kubecache.Store
	groupStore            kubecache.Store
	discoverableRoles     []*rbacv1.ClusterRole
	discoverableRoleNames sets.Set[string]
}

// newPermissionStore creates a new permission store
func newPermissionStore() *permissionStore {
	return &permissionStore{
		userStore:             kubecache.NewStore(userPermissionRecordKeyFn),
		groupStore:            kubecache.NewStore(userPermissionRecordKeyFn),
		discoverableRoleNames: sets.New[string](),
	}
}

// addPermissionForSubjects adds permissions for a list of subjects
func (ps *permissionStore) addPermissionForSubjects(
	subjects []rbacv1.Subject,
	roleName string,
	binding clusterviewv1alpha1.ClusterBinding,
) {
	for _, subject := range subjects {
		switch subject.Kind {
		case rbacv1.UserKind:
			ps.addPermissionForUser(subject.Name, roleName, binding)
		case rbacv1.GroupKind:
			ps.addPermissionForGroup(subject.Name, roleName, binding)
		}
	}
}

// addPermissionForUser adds or merges a permission binding for a user
func (ps *permissionStore) addPermissionForUser(userName, roleName string, binding clusterviewv1alpha1.ClusterBinding) {
	ps.addPermissionForSubject(ps.userStore, userName, roleName, binding)
}

// addPermissionForGroup adds or merges a permission binding for a group
func (ps *permissionStore) addPermissionForGroup(groupName, roleName string, binding clusterviewv1alpha1.ClusterBinding) {
	ps.addPermissionForSubject(ps.groupStore, groupName, roleName, binding)
}

// addPermissionForSubject is a helper that adds permissions to a store
func (ps *permissionStore) addPermissionForSubject(
	store kubecache.Store,
	subjectName, roleName string,
	binding clusterviewv1alpha1.ClusterBinding,
) {
	// Get existing record or create new one
	var record *UserPermissionRecord
	obj, exists, _ := store.GetByKey(subjectName)
	if exists {
		record = obj.(*UserPermissionRecord)
	} else {
		record = &UserPermissionRecord{
			Subject:     subjectName,
			Permissions: make(map[string][]clusterviewv1alpha1.ClusterBinding),
		}
	}

	// Add or merge the binding
	record.Permissions[roleName] = mergeOrAppendBinding(record.Permissions[roleName], binding)

	// Update the store
	if exists {
		_ = store.Update(record)
	} else {
		_ = store.Add(record)
	}
}

// getPermissions returns combined permissions for a user and their groups.
// If both managedcluster:admin and managedcluster:view exist, only admin is returned since it's a superset.
func (ps *permissionStore) getPermissions(userName string, groups []string) map[string][]clusterviewv1alpha1.ClusterBinding {
	roleBindings := make(map[string][]clusterviewv1alpha1.ClusterBinding)

	// Get permissions from user
	obj, exists, _ := ps.userStore.GetByKey(userName)
	if exists {
		record := obj.(*UserPermissionRecord)
		for roleName, bindings := range record.Permissions {
			for _, binding := range bindings {
				roleBindings[roleName] = mergeOrAppendBinding(roleBindings[roleName], binding)
			}
		}
	}

	// Get permissions from groups
	for _, group := range groups {
		obj, exists, _ := ps.groupStore.GetByKey(group)
		if exists {
			record := obj.(*UserPermissionRecord)
			for roleName, bindings := range record.Permissions {
				for _, binding := range bindings {
					roleBindings[roleName] = mergeOrAppendBinding(roleBindings[roleName], binding)
				}
			}
		}
	}

	// Deduplicate per-cluster: if managedcluster:admin exists for a cluster,
	// remove managedcluster:view bindings only for that specific cluster since admin is a superset.
	// Note: Both admin and view are always cluster-scoped (never namespace-scoped) in the current
	// implementation, so we only need to check cluster names.
	adminBindings, hasAdmin := roleBindings[clusterviewv1alpha1.ManagedClusterAdminRole]
	viewBindings, hasView := roleBindings[clusterviewv1alpha1.ManagedClusterViewRole]

	if hasAdmin && hasView {
		// Build a set of clusters that have admin permissions
		adminClusters := sets.New[string]()
		for _, binding := range adminBindings {
			adminClusters.Insert(binding.Cluster)
		}

		// Filter out view bindings for clusters that already have admin
		filteredViewBindings := make([]clusterviewv1alpha1.ClusterBinding, 0, len(viewBindings))
		for _, binding := range viewBindings {
			if !adminClusters.Has(binding.Cluster) {
				filteredViewBindings = append(filteredViewBindings, binding)
			}
		}

		// Update or remove view role based on remaining bindings
		if len(filteredViewBindings) > 0 {
			roleBindings[clusterviewv1alpha1.ManagedClusterViewRole] = filteredViewBindings
		} else {
			delete(roleBindings, clusterviewv1alpha1.ManagedClusterViewRole)
		}
	}

	return roleBindings
}

// setDiscoverableRoles sets the discoverable ClusterRoles
func (ps *permissionStore) setDiscoverableRoles(roles []*rbacv1.ClusterRole) {
	ps.discoverableRoles = roles
	ps.discoverableRoleNames = sets.New[string]()
	for _, role := range roles {
		ps.discoverableRoleNames.Insert(role.Name)
	}
}

// getDiscoverableRoles returns the discoverable ClusterRoles
func (ps *permissionStore) getDiscoverableRoles() []*rbacv1.ClusterRole {
	return ps.discoverableRoles
}

// hasDiscoverableRole checks if a role name is in the discoverable roles set
func (ps *permissionStore) hasDiscoverableRole(roleName string) bool {
	return ps.discoverableRoleNames.Has(roleName)
}
