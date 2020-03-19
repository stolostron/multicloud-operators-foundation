// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Subject contains a reference to a placed object
type Subject struct {
	// Kind of object being referenced. Values defined by this API group are "Deployable" and "ComplianceResource".
	Kind string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	// APIGroup holds the API group of the referenced subject.
	// +optional
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,2,opt,name=apiGroup"`
	// Name of the object being referenced.
	Name string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
}

// PlacementPolicyRef contains information that points to the Placement policy being used
type PlacementPolicyRef struct {
	// Name of the PlacementPolicy instance
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Kind of object being referenced. Values defined by this API group are "Deployable" and "ComplianceResource".
	// +optional
	Kind string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	// APIGroup holds the API group of the referenced subject.
	// +optional
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,3,opt,name=apiGroup"`
	// Name of the object being referenced.
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlacementBinding struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []Subject `json:"subjects,omitempty" protobuf:"bytes,2,rep,name=subjects"`

	// PlacementPolicyRef references a PlacementPolicy
	PlacementPolicyRef PlacementPolicyRef `json:"placementRef,omitempty" protobuf:"bytes,3,opt,name=placementRef"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlacementBindingList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// List of Cluster objects.
	Items []PlacementBinding `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlacementPolicy struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec of Node Template
	Spec PlacementPolicySpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// keep consistency with fed resource
	Status PlacementPolicyStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
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
	Type  ResourceType   `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`
	Order SelectionOrder `json:"order,omitempty" protobuf:"bytes,2,opt,name=order"`
}

// ClusterConditionType marks the kind of cluster condition being reported.
type ClusterConditionType string

type ClusterConditionFilter struct {
	// +optional
	Type ClusterConditionType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`
	// +optional
	Status v1.ConditionStatus `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}

type PlacementPolicySpec struct {
	// Deprecated since 3.1.2: replaced by ClusterReplicas
	// +optional
	Replicas *int32 `json:"replicas,omitempty" protobuf:"bytes,1,opt,name=replicas"`
	// Deprecated since 3.1.2: replaced by resource hint
	// +optional
	ResourceSelector ResourceHint `json:"resourceSelector,omitempty" protobuf:"bytes,2,opt,name=resourceSelector"`
	// Deprecated since 3.1.2: replaced by resource hint
	// +optional
	ClustersSelector *metav1.LabelSelector `json:"clusterSelector,omitempty" protobuf:"bytes,3,opt,name=clusterSelector"`
	////////////////////////////////////////////////////////////////////////////
	// +optional
	// number of replicas Application wants to
	ClusterReplicas *int32 `json:"clusterReplicas,omitempty" protobuf:"bytes,4,opt,name=clusterReplicas"`
	// +optional
	// Target Clusters
	ClusterNames []string `json:"clusterNames,omitempty" protobuf:"bytes,5,rep,name=clusterNames"`
	// +optional
	// Target Cluster is a selector of cluster
	ClusterLabels *metav1.LabelSelector `json:"clusterLabels,omitempty" protobuf:"bytes,6,opt,name=clusterLabels"`
	// +optional
	ClusterConditions []ClusterConditionFilter `json:"clusterConditions,omitempty" protobuf:"bytes,7,rep,name=clusterConditions"`
	// +optional
	// Select Resource
	ResourceHint ResourceHint `json:"resourceHint,omitempty" protobuf:"bytes,8,opt,name=resourceHint"`
	// +optional
	// Set ComplianceFilters
	ComplianceNames []string `json:"compliances,omitempty" protobuf:"bytes,9,rep,name=compliances"`
}

type PlacementPolicyDecision struct {
	ClusterName      string `json:"clusterName,omitempty" protobuf:"bytes,1,opt,name=clusterName"`
	ClusterNamespace string `json:"clusterNamespace,omitempty" protobuf:"bytes,2,opt,name=clusterNamespace"`
}

type PlacementPolicyStatus struct {
	Decisions []PlacementPolicyDecision `json:"decisions,omitempty" protobuf:"bytes,1,rep,name=decisions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PlacementPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// List of Cluster objects.
	Items []PlacementPolicy `json:"items" protobuf:"bytes,2,rep,name=items"`
}
