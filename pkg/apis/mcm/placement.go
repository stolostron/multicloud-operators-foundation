// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// Subject contains a reference to a placed object
type Subject struct {

	// Kind of object being referenced. Values defined by this API group are "Deployable" and "ComplianceResource".
	Kind string
	// APIGroup holds the API group of the referenced subject.
	// +optional
	APIGroup string
	// Name of the object being referenced.
	Name string
}

// PlacementPolicyRef contains information that points to the Placement policy being used
type PlacementPolicyRef struct {
	// Name of the PlacementPolicy instance
	Name string
	// Kind of object being referenced. Values defined by this API group are "Deployable" and "ComplianceResource".
	// +optional
	Kind string
	// APIGroup holds the API group of the referenced subject.
	// +optional
	APIGroup string
	// Name of the object being referenced.
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlacementBinding struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []Subject

	// PlacementPolicyRef references a PlacementPolicy
	PlacementPolicyRef PlacementPolicyRef
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlacementBindingList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta
	// List of Cluster objects.
	Items []PlacementBinding
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlacementPolicy struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta
	// Spec of Node Template
	Spec PlacementPolicySpec
	// keep consistency with fed resource
	Status PlacementPolicyStatus
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
	Type  ResourceType
	Order SelectionOrder
}

type ClusterConditionFilter struct {
	// +optional
	Type v1alpha1.ClusterConditionType
	// +optional
	Status v1.ConditionStatus
}

type PlacementPolicySpec struct {
	// Deprecated since 3.1.2: replaced by ClusterReplicas
	// +optional
	Replicas *int32
	// Deprecated since 3.1.2: replaced by resource hint
	// +optional
	ResourceSelector ResourceHint
	// Deprecated since 3.1.2: replaced by resource hint
	// +optional
	ClustersSelector *metav1.LabelSelector
	////////////////////////////////////////////////////////////////////////////
	// +optional
	ClusterReplicas *int32
	// +optional
	// Target Clusters
	ClusterNames []string
	// +optional
	// Target Cluster is a selector of cluster
	ClusterLabels *metav1.LabelSelector
	// +optional
	ClusterConditions []ClusterConditionFilter
	// +optional
	// Select Resource
	ResourceHint ResourceHint
	// +optional
	// Set ComplianceFilters
	ComplianceNames []string
}

type PlacementPolicyDecision struct {
	ClusterName      string
	ClusterNamespace string
}

type PlacementPolicyStatus struct {
	Decisions []PlacementPolicyDecision
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlacementPolicyList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta
	// List of Cluster objects.
	Items []PlacementPolicy
}
