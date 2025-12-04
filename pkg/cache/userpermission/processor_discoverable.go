package userpermission

import (
	"crypto/sha256"
	"fmt"
	"sort"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/klog/v2"
	clusterpermissionv1alpha1 "open-cluster-management.io/cluster-permission/api/v1alpha1"
	cplisters "open-cluster-management.io/cluster-permission/client/listers/api/v1alpha1"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

// discoverablePermissionProcessor processes discoverable ClusterRoles and ClusterPermissions
type discoverablePermissionProcessor struct {
	clusterRoleLister       rbacv1listers.ClusterRoleLister
	clusterPermissionLister cplisters.ClusterPermissionLister
}

// sync implements permissionProcessor for discoverablePermissionProcessor
// Gets discoverable ClusterRoles and processes ClusterPermissions
func (p *discoverablePermissionProcessor) sync(store *permissionStore) error {
	// Get all discoverable ClusterRoles
	roles, err := p.getDiscoverableClusterRoles()
	if err != nil {
		return err
	}

	// Store discoverable roles in the permission store
	store.setDiscoverableRoles(roles)

	// Process all ClusterPermissions
	if err := p.processAllClusterPermissions(store); err != nil {
		return err
	}

	return nil
}

// getDiscoverableClusterRoles returns all ClusterRoles with the discoverable label
func (p *discoverablePermissionProcessor) getDiscoverableClusterRoles() ([]*rbacv1.ClusterRole, error) {
	allRoles, err := p.clusterRoleLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	klog.V(4).Infof("Checking %d ClusterRoles for discoverable label %s=true",
		len(allRoles), clusterviewv1alpha1.DiscoverableClusterRoleLabel)

	var discoverableRoles []*rbacv1.ClusterRole
	for _, role := range allRoles {
		if role.Labels != nil && role.Labels[clusterviewv1alpha1.DiscoverableClusterRoleLabel] == "true" {
			klog.V(4).Infof("Found discoverable ClusterRole: %s", role.Name)
			discoverableRoles = append(discoverableRoles, role)
		}
	}

	klog.V(2).Infof("Found %d discoverable ClusterRoles", len(discoverableRoles))
	return discoverableRoles, nil
}

// processAllClusterPermissions gets all ClusterPermissions and processes their bindings
func (p *discoverablePermissionProcessor) processAllClusterPermissions(store *permissionStore) error {
	clusterPermissions, err := p.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		return err
	}

	klog.V(2).Infof("Processing %d ClusterPermissions", len(clusterPermissions))

	for _, cp := range clusterPermissions {
		clusterName := cp.Namespace

		klog.V(4).Infof("Processing ClusterPermission %s/%s", cp.Namespace, cp.Name)

		// Process ClusterRoleBinding (single)
		if cp.Spec.ClusterRoleBinding != nil {
			p.processClusterRoleBinding(cp.Spec.ClusterRoleBinding, clusterName, store)
		}

		// Process ClusterRoleBindings (multiple)
		if cp.Spec.ClusterRoleBindings != nil {
			klog.V(4).Infof("ClusterPermission %s/%s has %d ClusterRoleBindings",
				cp.Namespace, cp.Name, len(*cp.Spec.ClusterRoleBindings))
			for _, binding := range *cp.Spec.ClusterRoleBindings {
				p.processClusterRoleBinding(&binding, clusterName, store)
			}
		}

		// Process RoleBindings
		if cp.Spec.RoleBindings != nil {
			klog.V(4).Infof("ClusterPermission %s/%s has %d RoleBindings",
				cp.Namespace, cp.Name, len(*cp.Spec.RoleBindings))
			for _, binding := range *cp.Spec.RoleBindings {
				p.processRoleBinding(&binding, clusterName, store)
			}
		}
	}

	return nil
}

// processClusterRoleBinding processes a ClusterRoleBinding and adds permissions to the store
func (p *discoverablePermissionProcessor) processClusterRoleBinding(
	binding *clusterpermissionv1alpha1.ClusterRoleBinding,
	clusterName string,
	store *permissionStore,
) {
	if binding == nil || binding.RoleRef.Name == "" {
		klog.V(4).Info("Skipping nil or empty ClusterRoleBinding")
		return
	}

	roleRefName := binding.RoleRef.Name
	klog.V(4).Infof("Processing ClusterRoleBinding for role %s in cluster %s", roleRefName, clusterName)

	if !store.hasDiscoverableRole(roleRefName) {
		klog.V(4).Infof("Skipping ClusterRoleBinding: role %s is not in discoverable roles set", roleRefName)
		return
	}

	clusterBinding := clusterviewv1alpha1.ClusterBinding{
		Cluster:    clusterName,
		Scope:      clusterviewv1alpha1.BindingScopeCluster,
		Namespaces: []string{"*"},
	}

	subjects, ok := extractSubjectsFromClusterRoleBinding(binding)
	if !ok {
		klog.V(4).Infof("Skipping ClusterRoleBinding for role %s: no subjects found", roleRefName)
		return
	}

	klog.V(4).Infof("Adding ClusterRoleBinding for discoverable role %s with %d subject(s)",
		roleRefName, len(subjects))

	store.addPermissionForSubjects(subjects, roleRefName, clusterBinding)
}

// processRoleBinding processes a RoleBinding and adds permissions to the store
func (p *discoverablePermissionProcessor) processRoleBinding(
	binding *clusterpermissionv1alpha1.RoleBinding,
	clusterName string,
	store *permissionStore,
) {
	if binding == nil || binding.RoleRef.Name == "" {
		klog.V(4).Info("Skipping nil or empty RoleBinding")
		return
	}

	roleRefName := binding.RoleRef.Name
	klog.V(4).Infof("Processing RoleBinding for role %s in cluster %s, namespace %s",
		roleRefName, clusterName, binding.Namespace)

	if !store.hasDiscoverableRole(roleRefName) {
		klog.V(4).Infof("Skipping RoleBinding: role %s is not in discoverable roles set", roleRefName)
		return
	}

	namespace := binding.Namespace

	namespaceBinding := clusterviewv1alpha1.ClusterBinding{
		Cluster:    clusterName,
		Scope:      clusterviewv1alpha1.BindingScopeNamespace,
		Namespaces: []string{namespace},
	}

	subjects, ok := extractSubjectsFromRoleBinding(binding)
	if !ok {
		klog.V(4).Infof("Skipping RoleBinding for role %s: no subjects found", roleRefName)
		return
	}

	klog.V(4).Infof("Adding RoleBinding for discoverable role %s with %d subject(s)",
		roleRefName, len(subjects))

	store.addPermissionForSubjects(subjects, roleRefName, namespaceBinding)
}

// getResourceVersionHash implements permissionProcessor for discoverablePermissionProcessor
// Computes a hash of resources relevant to discoverable permissions:
// - Discoverable ClusterRoles
// - ClusterPermissions
func (p *discoverablePermissionProcessor) getResourceVersionHash() (string, error) {
	h := sha256.New()
	var versions []string

	// 1. Discoverable ClusterRoles
	if err := p.addDiscoverableClusterRoleVersions(&versions); err != nil {
		return "", err
	}

	// 2. ClusterPermissions
	if err := p.addClusterPermissionVersions(&versions); err != nil {
		return "", err
	}

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
func (p *discoverablePermissionProcessor) addDiscoverableClusterRoleVersions(versions *[]string) error {
	clusterRoles, err := p.clusterRoleLister.List(labels.Everything())
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
func (p *discoverablePermissionProcessor) addClusterPermissionVersions(versions *[]string) error {
	clusterPermissions, err := p.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to list ClusterPermissions: %w", err)
	}
	for _, cp := range clusterPermissions {
		*versions = append(*versions, fmt.Sprintf(ClusterPermissionFormat, cp.Namespace, cp.ResourceVersion))
	}
	return nil
}
