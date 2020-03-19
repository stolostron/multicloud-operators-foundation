// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewList is a list of all the resource view
type ResourceViewList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Cluster objects.
	Items []ResourceView `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceView is the view of resources on a set of cluster
type ResourceView struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the work.
	// +optional
	Spec ResourceViewSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Status describes the result of a work
	// +optional
	Status ResourceViewStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// ResourceViewSpec is the spec for resource view
type ResourceViewSpec struct {
	// Selector for clusters.
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty" protobuf:"bytes,1,opt,name=clusterSelector"`

	// Scope describes the filter of the view.
	Scope ViewFilter `json:"scope,omitempty" protobuf:"bytes,2,opt,name=scope"`

	// Mode is the mode for resource query
	// +optional
	Mode ResourceFilterMode `json:"mode,omitempty" protobuf:"bytes,4,opt,name=mode"`

	// SummaryOnly is the flag to return only summary
	// +optional
	SummaryOnly bool `json:"summaryOnly,omitempty" protobuf:"bool,3,opt,name=summaryOnly"`

	// UpdateIntervalSeconds is the inteval to update view
	// +optional
	UpdateIntervalSeconds int32 `json:"updateIntervalSeconds,omitempty" protobuf:"varint,5,opt,name=updateIntervalSeconds"`
}

// ViewFilter is the filter of resources
type ViewFilter struct {
	// LabelSelect is a selector that selects a set of resources
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`

	// FieldSelector is a selector that select a set of resources
	// +optional
	FieldSelector string `json:"fieldSelector,omitempty" protobuf:"bytes,2,opt,name=fieldSelector"`

	// APIGroup is the group of resources
	// +optional
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,3,opt,name=apiGroup"`

	// ResouceType is the resource type of the subject
	// +optional
	Resource string `json:"resource,omitempty" protobuf:"bytes,4,opt,name=resource"`

	// Name is the name of the subject
	// +optional
	ResourceName string `json:"resourceName,omitempty" protobuf:"bytes,5,opt,name=resourceName"`

	// Name is the name of the subject
	// +optional
	NameSpace string `json:"namespace,omitempty" protobuf:"bytes,6,opt,name=namespace"`
}

// ResourceViewStatus describes the status of view
type ResourceViewStatus struct {
	Conditions []ViewCondition `json:"conditions,omitempty" protobuf:"bytes,1,rep,name=conditions"`

	// Works point to the related work result on each cluster
	Results map[string]runtime.RawExtension `json:"results,omitempty" protobuf:"bytes,2,rep,name=results"`
}

// ViewCondition contains condition information for a view.
type ViewCondition struct {
	// Type is the type of the cluster condition.
	Type WorkStatusType `json:"type,omitempty" protobuf:"bytes,1,opt,name=results"`

	// Status is the status of the condition. One of True, False, Unknown.
	Status v1.ConditionStatus `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`

	// LastUpdateTime is the last time this condition was updated.
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,3,opt,name=lastUpdateTime"`

	// Reason is a (brief) reason for the condition's last status change.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
}
