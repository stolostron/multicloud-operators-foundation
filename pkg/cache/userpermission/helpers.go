package userpermission

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterpermissionv1alpha1 "open-cluster-management.io/cluster-permission/api/v1alpha1"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

// mergeOrAppendBinding merges or appends a binding to the existing bindings list
// - If the same cluster exists with cluster scope, do nothing (already covered)
// - If the same cluster exists with namespace scope, merge the namespaces
// - If different cluster, append as new binding
func mergeOrAppendBinding(
	existingBindings []clusterviewv1alpha1.ClusterBinding,
	newBinding clusterviewv1alpha1.ClusterBinding,
) []clusterviewv1alpha1.ClusterBinding {
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
			if existing.Scope == clusterviewv1alpha1.BindingScopeNamespace &&
				newBinding.Scope == clusterviewv1alpha1.BindingScopeNamespace {
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

// extractSubjectsFromClusterRoleBinding extracts subjects from a ClusterRoleBinding
// Handles both Subject (singular, embedded) and Subjects (plural, array) fields
// Per API docs: "If both subject and subjects exist then only subjects will be used"
func extractSubjectsFromClusterRoleBinding(binding *clusterpermissionv1alpha1.ClusterRoleBinding) ([]rbacv1.Subject, bool) {
	if len(binding.Subjects) > 0 {
		return binding.Subjects, true
	}
	if binding.Subject != nil {
		return []rbacv1.Subject{*binding.Subject}, true
	}
	return nil, false
}

// extractSubjectsFromRoleBinding extracts subjects from a RoleBinding
// Handles both Subject (singular, embedded) and Subjects (plural, array) fields
// Per API docs: "If both subject and subjects exist then only subjects will be used"
func extractSubjectsFromRoleBinding(binding *clusterpermissionv1alpha1.RoleBinding) ([]rbacv1.Subject, bool) {
	if len(binding.Subjects) > 0 {
		return binding.Subjects, true
	}
	if binding.Subject != nil {
		return []rbacv1.Subject{*binding.Subject}, true
	}
	return nil, false
}
