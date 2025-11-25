/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionTypeAppliedRBACManifestWork   string = "AppliedRBACManifestWork"
	ConditionTypeValidation                string = "Validation"
	ConditionTypeValidateRolesExist        string = "ValidateRolesExist"
	ConditionTypeValidateClusterRolesExist string = "ValidateClusterRolesExist"
)

// ClusterPermissionSpec defines the desired state of ClusterPermission
type ClusterPermissionSpec struct {
	// Validate enables validation of roles and clusterroles on the managed cluster using ManifestWork
	// When enabled, the controller will create a validation ManifestWork to check if the referenced
	// roles and clusterroles exist on the managed cluster
	// +optional
	Validate *bool `json:"validate,omitempty"`

	// ClusterRole represents the ClusterRole that is being created on the managed cluster
	// +optional
	ClusterRole *ClusterRole `json:"clusterRole,omitempty"`

	// ClusterRoleBinding represents the ClusterRoleBinding that is being created on the managed cluster
	// +optional
	// +kubebuilder:validation:XValidation:rule="has(self.subject) || has(self.subjects)",message="Either subject or subjects has to exist in clusterRoleBinding"
	ClusterRoleBinding *ClusterRoleBinding `json:"clusterRoleBinding,omitempty"`

	// ClusterRoleBindings represents multiple ClusterRoleBindings that are being created on the managed cluster
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.all(i, has(i.subject) || has(i.subjects))",message="Either subject or subjects has to exist in every clusterRoleBinding"
	ClusterRoleBindings *[]ClusterRoleBinding `json:"clusterRoleBindings,omitempty"`

	// Roles represents roles that are being created on the managed cluster
	// +optional
	Roles *[]Role `json:"roles,omitempty"`

	// RoleBindings represents RoleBindings that are being created on the managed cluster
	// +optional
	// +kubebuilder:validation:XValidation:rule="self.all(i, has(i.subject) || has(i.subjects))",message="Either subject or subjects has to exist in every roleBinding"
	RoleBindings *[]RoleBinding `json:"roleBindings,omitempty"`
}

// ClusterRole represents the ClusterRole that is being created on the managed cluster
type ClusterRole struct {
	// Rules holds all the PolicyRules for this ClusterRole
	// +required
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// ClusterRoleBinding represents the ClusterRoleBinding that is being created on the managed cluster
type ClusterRoleBinding struct {
	// Subject contains a reference to the object or user identities a ClusterPermission binding applies to.
	// Besides the typical subject for a binding, a ManagedServiceAccount can be used as a subject as well.
	// If both subject and subjects exist then only subjects will be used.
	// +optional
	Subject *rbacv1.Subject `json:"subject,omitempty"`

	// Subjects contains an array of references to objects or user identities a ClusterPermission binding applies to.
	// Besides the typical subject for a binding, a ManagedServiceAccount can be used as a subject as well.
	// If both subject and subjects exist then only subjects will be used.
	// +optional
	Subjects []rbacv1.Subject `json:"subjects,omitempty"`

	// Name of the ClusterRoleBinding if a name different than the ClusterPermission name is used
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,4,opt,name=name"`

	// RoleRef contains information that points to the ClusterRole being used
	// +optional
	RoleRef *rbacv1.RoleRef `json:"roleRef,omitempty"`
}

// Role represents the Role that is being created on the managed cluster
type Role struct {
	// Namespace of the Role for that is being created on the managed cluster
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`

	// NamespaceSelector define the general labelSelector which namespace to apply the rules to
	// Note: the namespace must exists on the hub cluster
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Rules holds all the PolicyRules for this Role
	// +required
	Rules []rbacv1.PolicyRule `json:"rules"`
}

// RoleBinding represents the RoleBinding that is being created on the managed cluster
type RoleBinding struct {
	// Subject contains a reference to the object or user identities a ClusterPermission binding applies to.
	// Besides the typical subject for a binding, a ManagedServiceAccount can be used as a subject as well.
	// If both subject and subjects exist then only subjects will be used.
	// +optional
	*rbacv1.Subject `json:"subject,omitempty"`

	// Subjects contains an array of references to objects or user identities a ClusterPermission binding applies to.
	// Besides the typical subject for a binding, a ManagedServiceAccount can be used as a subject as well.
	// If both subject and subjects exist then only subjects will be used.
	// +optional
	Subjects []rbacv1.Subject `json:"subjects,omitempty"`

	// RoleRef contains information that points to the role being used
	// +required
	RoleRef `json:"roleRef"`

	// Namespace of the Role for that is being created on the managed cluster
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`

	// NamespaceSelector define the general labelSelector which namespace to apply the rules to
	// Note: the namespace must exists on the hub cluster
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`

	// Name of the RoleBinding if a name different than the ClusterPermission name is used
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,4,opt,name=name"`
}

// RoleRef contains information that points to the role being used
type RoleRef struct {
	// Kind is the type of resource being referenced
	// +required
	Kind string `json:"kind"`

	// APIGroup is the group for the resource being referenced
	// +optional
	APIGroup string `json:"apiGroup" protobuf:"bytes,1,opt,name=apiGroup"`

	// Name is the name of resource being referenced
	// +optional
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
}

// ClusterPermissionStatus defines the observed state of ClusterPermission
type ClusterPermissionStatus struct {
	// Conditions is the condition list.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced

// ClusterPermission is the Schema for the clusterpermissions API
type ClusterPermission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPermissionSpec   `json:"spec,omitempty"`
	Status ClusterPermissionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterPermissionList contains a list of ClusterPermission
type ClusterPermissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPermission `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterPermission{}, &ClusterPermissionList{})
}
