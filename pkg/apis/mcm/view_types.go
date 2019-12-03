// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ViewLabel is the label set to point to the view
const ViewLabel = "mcm.ibm.com/resourceview"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewList is a list of all the resource view
type ResourceViewList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta

	// List of Cluster objects.
	Items []ResourceView
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceView is the view of resources on a set of cluster
type ResourceView struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Spec defines the behavior of the work.
	// +optional
	Spec ResourceViewSpec

	// Status describes the result of a work
	// +optional
	Status ResourceViewStatus
}

// ResourceViewSpec is the spec for resource view
type ResourceViewSpec struct {
	// Selector for works.
	ClusterSelector *metav1.LabelSelector

	// Scope describes the filter of the view.
	Scope ViewFilter

	// SummaryOnly is the flag to return only summary
	// +optional
	SummaryOnly bool

	// Mode is the mode for resource query
	// +optional
	Mode ResourceFilterMode

	// UpdateIntervalSeconds is the inteval to update view
	// +optional
	UpdateIntervalSeconds int
}

// ViewFilter is the filter of resources
type ViewFilter struct {
	// LabelSelect is a selector that selects a set of resources
	// +optional
	LabelSelector *metav1.LabelSelector

	// FieldSelector is a selector that select a set of resources
	// +optional
	FieldSelector string

	// APIGroup is the group of resources
	// +optional
	APIGroup string

	// ResouceType is the resource type of the subject
	// +optional
	Resource string

	// Name is the name of the subject
	// +optional
	ResourceName string

	// Name is the name of the subject
	// +optional
	NameSpace string
}

// ResourceViewStatus describes the status of view
type ResourceViewStatus struct {
	Conditions []ViewCondition

	// Works point to the related work result on each cluster
	Results map[string]runtime.RawExtension
}

// ViewCondition contains condition information for a view.
type ViewCondition struct {
	// Type is the type of the cluster condition.
	Type WorkStatusType

	// Status is the status of the condition. One of True, False, Unknown.
	Status v1.ConditionStatus

	// LastUpdateTime is the last time this condition was updated.
	// +optional
	LastUpdateTime metav1.Time

	// Reason is a (brief) reason for the condition's last status change.
	// +optional
	Reason string
}
