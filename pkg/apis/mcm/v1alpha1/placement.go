// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// Subject contains a reference to a placed object
type Subject struct {
	// Kind of object being referenced. Values defined by this API group are "Deployable" and "ComplianceResource".
	Kind string `json:"kind,omitempty"`
	// APIGroup holds the API group of the referenced subject.
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`
	// Name of the object being referenced.
	Name string `json:"name,omitempty"`
}

// PlacementPolicyRef contains information that points to the Placement policy being used
type PlacementPolicyRef struct {
	// Name of the PlacementPolicy instance
	Name string `json:"name,omitempty"`
	// Kind of object being referenced. Values defined by this API group are "Deployable" and "ComplianceResource".
	// +optional
	Kind string `json:"kind,omitempty"`
	// APIGroup holds the API group of the referenced subject.
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`
	// Name of the object being referenced.
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlacementBinding struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []Subject `json:"subjects,omitempty"`

	// PlacementPolicyRef references a PlacementPolicy
	PlacementPolicyRef PlacementPolicyRef `json:"placementRef,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlacementBindingList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of Cluster objects.
	Items []PlacementBinding `json:"items,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlacementPolicy struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec of Node Template
	Spec PlacementPolicySpec `json:"spec,omitempty"`
	// keep consistency with fed resource
	Status PlacementPolicyStatus `json:"status,omitempty"`
}

// NodeState is the type for Nodes
type ResourceType string

// These are valid conditions of a cluster.
const (
	ResourceTypeNone   ResourceType = ""
	ResourceTypeCPU    ResourceType = "cpu"
	ResourceTypeMemory ResourceType = "memory"
)

// NodeState is the type for Nodes
type SelectionOrder string

// These are valid conditions of a cluster.
const (
	SelectionOrderNone SelectionOrder = ""
	SelectionOrderDesc SelectionOrder = "desc"
	SelectionOrderAsce SelectionOrder = "asc"
)

type ResourceHint struct {
	Type  ResourceType   `json:"type,omitempty"`
	Order SelectionOrder `json:"order,omitempty"`
}

type ClusterConditionFilter struct {
	// +optional
	Type v1alpha1.ClusterConditionType `json:"type,omitempty"`
	// +optional
	Status v1.ConditionStatus `json:"status,omitempty"`
}

type PlacementPolicySpec struct {
	// Deprecated since 3.1.2: replaced by ClusterReplicas
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Deprecated since 3.1.2: replaced by resource hint
	// +optional
	ResourceSelector ResourceHint `json:"resourceSelector,omitempty"`
	// Deprecated since 3.1.2: replaced by resource hint
	// +optional
	ClustersSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`
	////////////////////////////////////////////////////////////////////////////
	// +optional
	// number of replicas Application wants to
	ClusterReplicas *int32 `json:"clusterReplicas,omitempty"`
	// +optional
	// Target Clusters
	ClusterNames []string `json:"clusterNames,omitempty"`
	// +optional
	// Target Cluster is a selector of cluster
	ClusterLabels *metav1.LabelSelector `json:"clusterLabels,omitempty"`
	// +optional
	ClusterConditions []ClusterConditionFilter `json:"clusterConditions,omitempty"`
	// +optional
	// Select Resource
	ResourceHint ResourceHint `json:"resourceHint,omitempty"`
	// +optional
	// Set ComplianceFilters
	ComplianceNames []string `json:"compliances,omitempty"`
}

type PlacementPolicyDecision struct {
	ClusterName      string `json:"clusterName,omitempty"`
	ClusterNamespace string `json:"clusterNamespace,omitempty"`
}

type PlacementPolicyStatus struct {
	Decisions []PlacementPolicyDecision `json:"decisions,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlacementPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of Cluster objects.
	Items []PlacementPolicy `json:"items"`
}
