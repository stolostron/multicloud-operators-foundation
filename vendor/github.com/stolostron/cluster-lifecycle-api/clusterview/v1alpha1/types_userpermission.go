package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=userpermissions,scope=Cluster

// UserPermission represents the permissions a user has across the fleet of managed clusters.
// Each UserPermission item corresponds to a labeled ClusterRole that the user has been granted
// access to via ClusterPermission resources on the hub cluster.
// The name of the UserPermission resource is the name of the ClusterRole.
type UserPermission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Status contains the permission bindings for this ClusterRole across clusters
	// +optional
	Status UserPermissionStatus `json:"status,omitempty"`
}

// UserPermissionStatus defines the observed state of UserPermission
type UserPermissionStatus struct {
	// Bindings contains the list of cluster/namespace bindings where this ClusterRole is granted
	// +optional
	Bindings []ClusterBinding `json:"bindings,omitempty"`

	// ClusterRoleDefinition contains the complete ClusterRole definition
	// This includes all the rules (apiGroups, resources, verbs) that define the permissions
	// +optional
	ClusterRoleDefinition ClusterRoleDefinition `json:"clusterRoleDefinition,omitempty"`
}

// ClusterBinding represents a binding of the ClusterRole to a specific cluster and scope
type ClusterBinding struct {
	// Cluster is the name of the managed cluster (namespace name on hub)
	// +required
	Cluster string `json:"cluster"`

	// Scope indicates whether this is a cluster-wide or namespace-specific binding
	// Possible values: "cluster" or "namespace"
	// +required
	// +kubebuilder:validation:Enum=cluster;namespace
	Scope BindingScope `json:"scope"`

	// Namespaces is the list of namespaces where the binding applies
	// If Scope is "cluster", this will contain ["*"] indicating cluster-wide access
	// If Scope is "namespace", this will contain the specific namespace names
	// +required
	Namespaces []string `json:"namespaces"`
}

// BindingScope defines the scope of a ClusterRole binding
// +kubebuilder:validation:Enum=cluster;namespace
type BindingScope string

const (
	// BindingScopeCluster indicates cluster-wide binding
	BindingScopeCluster BindingScope = "cluster"
	// BindingScopeNamespace indicates namespace-specific binding
	BindingScopeNamespace BindingScope = "namespace"
)

// ClusterRoleDefinition contains the definition of a ClusterRole
type ClusterRoleDefinition struct {
	// Rules holds all the PolicyRules for this ClusterRole
	// +optional
	Rules []rbacv1.PolicyRule `json:"rules,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UserPermissionList contains a list of UserPermission
type UserPermissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserPermission `json:"items"`
}

// Label key for discoverable ClusterRoles
const (
	// DiscoverableClusterRoleLabel is the label key used to mark ClusterRoles as discoverable
	// through the UserPermission API
	DiscoverableClusterRoleLabel = "clusterview.open-cluster-management.io/discoverable"

	// ManagedClusterAdminRole is the synthetic role name for users with managedclusteradmin permissions
	ManagedClusterAdminRole = "managedcluster:admin"

	// ManagedClusterViewRole is the synthetic role name for users with managedclusterview permissions
	ManagedClusterViewRole = "managedcluster:view"
)
