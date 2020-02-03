// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta1

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
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Cluster objects.
	Items []ResourceView `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceView is the view of resources on a set of cluster
type ResourceView struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the work.
	// +optional
	Spec ResourceViewSpec `json:"spec,omitempty"`

	// Status describes the result of a work
	// +optional
	Status ResourceViewStatus `json:"status,omitempty"`
}

// ResourceViewSpec is the spec for resource view
type ResourceViewSpec struct {
	// Selector for clusters.
	ClusterSelector *metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// Scope describes the filter of the view.
	Scope ViewFilter `json:"scope,omitempty"`

	// ServerPrint is the flag to set print on server side
	// +optional
	SummaryOnly bool `json:"summaryOnly,omitempty"`

	// Mode is the mode for resource query
	// +optional
	Mode ResourceFilterMode `json:"mode,omitempty"`

	// UpdateIntervalSeconds is the inteval to update view
	// +optional
	UpdateIntervalSeconds int `json:"updateIntervalSeconds,omitempty"`
}

// ViewFilter is the filter of resources
type ViewFilter struct {
	// LabelSelect is a selector that selects a set of resources
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// FieldSelector is a selector that select a set of resources
	// +optional
	FieldSelector string `json:"fieldSelector,omitempty"`

	// APIGroup is the group of resources
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`

	// ResouceType is the resource type of the subject
	// +optional
	Resource string `json:"resource,omitempty"`

	// Name is the name of the subject
	// +optional
	ResourceName string `json:"resourceName,omitempty"`

	// Name is the name of the subject
	// +optional
	NameSpace string `json:"namespace,omitempty"`
}

// ResourceViewStatus describes the status of view
type ResourceViewStatus struct {
	Conditions []ViewCondition `json:"conditions,omitempty"`

	// Works point to the related work result on each cluster
	Results map[string]runtime.RawExtension `json:"results,omitempty"`
}

// ViewCondition contains condition information for a view.
type ViewCondition struct {
	// Type is the type of the cluster condition.
	Type WorkStatusType `json:"type,omitempty"`

	// Status is the status of the condition. One of True, False, Unknown.
	Status v1.ConditionStatus `json:"status,omitempty"`

	// LastUpdateTime is the last time this condition was updated.
	// +optional
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// Reason is a (brief) reason for the condition's last status change.
	// +optional
	Reason string `json:"reason,omitempty"`
}
