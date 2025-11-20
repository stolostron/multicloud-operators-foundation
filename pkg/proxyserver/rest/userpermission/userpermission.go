package userpermission

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	kubecache "k8s.io/client-go/tools/cache"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

const (
	// DiscoverableLabel is the label used to mark ClusterRoles as discoverable
	DiscoverableLabel = "clusterview.open-cluster-management.io/discoverable"
	// ManagedClusterAdminRole is the synthetic role name for managedclusteradmin permissions
	ManagedClusterAdminRole = "managedcluster:admin"
	// ManagedClusterViewRole is the synthetic role name for managedclusterview permissions
	ManagedClusterViewRole = "managedcluster:view"
)

type REST struct {
	clusterRoleLister        rbaclisters.ClusterRoleLister
	clusterPermissionLister  kubecache.GenericLister
	clusterPermissionIndexer kubecache.Indexer
	tableConverter           rest.TableConvertor
}

// NewREST returns a RESTStorage object that will work against UserPermission resources
func NewREST(
	clusterRoleLister rbaclisters.ClusterRoleLister,
	clusterPermissionIndexer kubecache.Indexer,
	clusterPermissionLister kubecache.GenericLister,
) *REST {
	return &REST{
		clusterRoleLister:        clusterRoleLister,
		clusterPermissionLister:  clusterPermissionLister,
		clusterPermissionIndexer: clusterPermissionIndexer,
		tableConverter:           rest.NewDefaultTableConvertor(clusterviewv1alpha1.Resource("userpermissions")),
	}
}

// New returns a new UserPermission
func (s *REST) New() runtime.Object {
	return &clusterviewv1alpha1.UserPermission{}
}

func (s *REST) Destroy() {
}

func (s *REST) NamespaceScoped() bool {
	return false
}

// NewList returns a new UserPermission list
func (*REST) NewList() runtime.Object {
	return &clusterviewv1alpha1.UserPermissionList{}
}

var _ = rest.Lister(&REST{})

// List retrieves a list of UserPermissions for the current user
func (s *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterviewv1alpha1.Resource("userpermissions"), "", fmt.Errorf("unable to list userpermissions without a user on the context"))
	}

	// Get all discoverable ClusterRoles
	discoverableRoles, err := s.getDiscoverableClusterRoles()
	if err != nil {
		return nil, err
	}

	// Build permission map for the user
	permissionMap, err := s.buildUserPermissionMap(user.GetName(), user.GetGroups(), discoverableRoles)
	if err != nil {
		return nil, err
	}

	// Check for managedclusteradmin/view permissions
	managedClusterPermissions := s.getManagedClusterPermissions(user.GetName(), user.GetGroups())

	// Merge managedcluster permissions into the map
	for roleName, bindings := range managedClusterPermissions {
		if existing, ok := permissionMap[roleName]; ok {
			permissionMap[roleName] = mergeBindings(existing, bindings)
		} else {
			permissionMap[roleName] = bindings
		}
	}

	// Convert map to list
	userPermissionList := &clusterviewv1alpha1.UserPermissionList{
		Items: make([]clusterviewv1alpha1.UserPermission, 0, len(permissionMap)),
	}

	for roleName, bindings := range permissionMap {
		userPerm := clusterviewv1alpha1.UserPermission{
			ObjectMeta: metav1.ObjectMeta{
				Name: roleName,
			},
			Status: clusterviewv1alpha1.UserPermissionStatus{
				Bindings: bindings,
			},
		}

		// Add ClusterRole definition for discoverable roles
		for _, role := range discoverableRoles {
			if role.Name == roleName {
				userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
					Rules: role.Rules,
				}
				break
			}
		}

		// Add synthetic role definitions for managedcluster permissions
		if roleName == ManagedClusterAdminRole {
			userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"*"},
					},
				},
			}
		} else if roleName == ManagedClusterViewRole {
			userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{"*"},
						Resources: []string{"*"},
						Verbs:     []string{"get", "list", "watch"},
					},
				},
			}
		}

		userPermissionList.Items = append(userPermissionList.Items, userPerm)
	}

	return userPermissionList, nil
}

func (c *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return c.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

var _ = rest.Getter(&REST{})

// Get retrieves a specific UserPermission by ClusterRole name
func (s *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, errors.NewForbidden(clusterviewv1alpha1.Resource("userpermissions"), "", fmt.Errorf("unable to get userpermission without a user on the context"))
	}

	// Get all discoverable ClusterRoles
	discoverableRoles, err := s.getDiscoverableClusterRoles()
	if err != nil {
		return nil, err
	}

	// Build permission map for the user
	permissionMap, err := s.buildUserPermissionMap(user.GetName(), user.GetGroups(), discoverableRoles)
	if err != nil {
		return nil, err
	}

	// Check for managedclusteradmin/view permissions
	managedClusterPermissions := s.getManagedClusterPermissions(user.GetName(), user.GetGroups())

	// Merge managedcluster permissions
	for roleName, bindings := range managedClusterPermissions {
		if existing, ok := permissionMap[roleName]; ok {
			permissionMap[roleName] = mergeBindings(existing, bindings)
		} else {
			permissionMap[roleName] = bindings
		}
	}

	// Check if the user has bindings for the requested ClusterRole
	bindings, ok := permissionMap[name]
	if !ok {
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
	for _, role := range discoverableRoles {
		if role.Name == name {
			userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
				Rules: role.Rules,
			}
			break
		}
	}

	// Add synthetic role definitions
	if name == ManagedClusterAdminRole {
		userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"*"},
				},
			},
		}
	} else if name == ManagedClusterViewRole {
		userPerm.Status.ClusterRoleDefinition = clusterviewv1alpha1.ClusterRoleDefinition{
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"*"},
					Resources: []string{"*"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}
	}

	return userPerm, nil
}

var _ = rest.SingularNameProvider(&REST{})

func (s *REST) GetSingularName() string {
	return "userpermission"
}

// getDiscoverableClusterRoles returns all ClusterRoles with the discoverable label
func (s *REST) getDiscoverableClusterRoles() ([]*rbacv1.ClusterRole, error) {
	allRoles, err := s.clusterRoleLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var discoverableRoles []*rbacv1.ClusterRole
	for _, role := range allRoles {
		if role.Labels != nil && role.Labels[DiscoverableLabel] == "true" {
			discoverableRoles = append(discoverableRoles, role)
		}
	}

	return discoverableRoles, nil
}

// buildUserPermissionMap scans ClusterPermissions and builds a map of ClusterRole -> Bindings
func (s *REST) buildUserPermissionMap(username string, groups []string, discoverableRoles []*rbacv1.ClusterRole) (map[string][]clusterviewv1alpha1.ClusterBinding, error) {
	permissionMap := make(map[string][]clusterviewv1alpha1.ClusterBinding)

	// Get all ClusterPermissions across all managed cluster namespaces
	clusterPermissions, err := s.clusterPermissionLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, obj := range clusterPermissions {
		// Convert to unstructured to access ClusterPermission fields
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			continue
		}

		uns := &unstructured.Unstructured{Object: u}
		clusterName := uns.GetNamespace()

		// Check ClusterRoleBindings
		clusterRoleBindings, found, err := unstructured.NestedSlice(u, "spec", "clusterRoleBindings")
		if err == nil && found {
			for _, crb := range clusterRoleBindings {
				binding, ok := crb.(map[string]interface{})
				if !ok {
					continue
				}

				// Get roleRef name
				roleRefName, err := getRoleRefName(binding)
				if err != nil || !isDiscoverableRole(roleRefName, discoverableRoles) {
					continue
				}

				// Check if user or user's groups are in subjects
				if matchesSubjects(binding, username, groups) {
					permissionMap[roleRefName] = append(permissionMap[roleRefName], clusterviewv1alpha1.ClusterBinding{
						Cluster:    clusterName,
						Scope:      clusterviewv1alpha1.BindingScopeCluster,
						Namespaces: []string{"*"},
					})
				}
			}
		}

		// Check RoleBindings
		roleBindings, found, err := unstructured.NestedSlice(u, "spec", "roleBindings")
		if err == nil && found {
			for _, rb := range roleBindings {
				binding, ok := rb.(map[string]interface{})
				if !ok {
					continue
				}

				// Get namespace
				namespace, _, err := unstructured.NestedString(binding, "namespace")
				if err != nil {
					continue
				}

				// Get roleRef name
				roleRefName, err := getRoleRefName(binding)
				if err != nil || !isDiscoverableRole(roleRefName, discoverableRoles) {
					continue
				}

				// Check if user or user's groups are in subjects
				if matchesSubjects(binding, username, groups) {
					// Group by cluster and collect namespaces
					found := false
					for i, existing := range permissionMap[roleRefName] {
						if existing.Cluster == clusterName && existing.Scope == clusterviewv1alpha1.BindingScopeNamespace {
							// Add namespace to existing binding
							permissionMap[roleRefName][i].Namespaces = append(existing.Namespaces, namespace)
							found = true
							break
						}
					}

					if !found {
						permissionMap[roleRefName] = append(permissionMap[roleRefName], clusterviewv1alpha1.ClusterBinding{
							Cluster:    clusterName,
							Scope:      clusterviewv1alpha1.BindingScopeNamespace,
							Namespaces: []string{namespace},
						})
					}
				}
			}
		}
	}

	return permissionMap, nil
}

// getRoleRefName extracts the roleRef name from a binding map
func getRoleRefName(binding map[string]interface{}) (string, error) {
	roleRef, found, err := unstructured.NestedMap(binding, "roleRef")
	if err != nil || !found {
		return "", fmt.Errorf("roleRef not found")
	}

	name, found, err := unstructured.NestedString(roleRef, "name")
	if err != nil || !found {
		return "", fmt.Errorf("roleRef name not found")
	}

	return name, nil
}

// matchesSubjects checks if the username or any groups match the binding subjects
func matchesSubjects(binding map[string]interface{}, username string, groups []string) bool {
	subjects, found, err := unstructured.NestedSlice(binding, "subjects")
	if err != nil || !found {
		return false
	}

	for _, subj := range subjects {
		subject, ok := subj.(map[string]interface{})
		if !ok {
			continue
		}

		kind, _, _ := unstructured.NestedString(subject, "kind")
		name, _, _ := unstructured.NestedString(subject, "name")

		if kind == "User" && name == username {
			return true
		}

		if kind == "Group" {
			for _, group := range groups {
				if name == group {
					return true
				}
			}
		}
	}

	return false
}

// getManagedClusterPermissions checks if user has managedclusteradmin or managedclusterview permissions
func (s *REST) getManagedClusterPermissions(username string, groups []string) map[string][]clusterviewv1alpha1.ClusterBinding {
	// TODO: Implement logic to check for open-cluster-management:admin:<cluster-name> and
	// open-cluster-management:view:<cluster-name> ClusterRole bindings
	// This would require checking ClusterRoleBindings from the hub cluster

	return make(map[string][]clusterviewv1alpha1.ClusterBinding)
}

// isDiscoverableRole checks if a ClusterRole name is in the list of discoverable roles
func isDiscoverableRole(name string, discoverableRoles []*rbacv1.ClusterRole) bool {
	for _, role := range discoverableRoles {
		if role.Name == name {
			return true
		}
	}
	return false
}

// mergeBindings merges two slices of ClusterBindings
func mergeBindings(a, b []clusterviewv1alpha1.ClusterBinding) []clusterviewv1alpha1.ClusterBinding {
	result := make([]clusterviewv1alpha1.ClusterBinding, len(a))
	copy(result, a)

	for _, binding := range b {
		found := false
		for i, existing := range result {
			if existing.Cluster == binding.Cluster && existing.Scope == binding.Scope {
				// Merge namespaces
				result[i].Namespaces = append(existing.Namespaces, binding.Namespaces...)
				found = true
				break
			}
		}
		if !found {
			result = append(result, binding)
		}
	}

	return result
}
