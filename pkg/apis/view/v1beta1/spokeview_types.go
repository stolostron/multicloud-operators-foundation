// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpokeView is the view of resources on a cluster
type SpokeView struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired configuration of a spokeview
	// +optional
	Spec SpokeViewSpec `json:"spec,omitempty"`
	// Status describes current status of a spokeview
	// +optional
	Status SpokeViewStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpokeViewList is a list of all the spokeview
type SpokeViewList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Cluster objects.
	Items []SpokeView `json:"items"`
}

// SpokeViewSpec defines the desired configuration of a view
type SpokeViewSpec struct {
	// Scope is the scope of the view on a cluster
	Scope SpokeViewScope `json:"scope,omitempty"`
}

// StatusCondition contains condition information for a work.
type StatusCondition struct {
	// Type is the type of the spoke work condition.
	// +required
	Type string `json:"type"`

	// Status is the status of the condition. One of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition changed from one status to another.
	// +required
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason is a (brief) reason for the condition's last status change.
	// +required
	Reason string `json:"reason"`

	// Message is a human-readable message indicating details about the last status change.
	// +required
	Message string `json:"message"`
}

// SpokeViewStatus returns the status of the spoke view
type SpokeViewStatus struct {

	// Conditions represents the conditions of this resource on spoke cluster
	// +required
	Conditions []StatusCondition `json:"conditions"`

	// WorkResult references the related result of the work
	Result runtime.RawExtension `json:"result,omitempty"`
}

// SpokeViewScope represents the scope of resources to be viewed
type SpokeViewScope struct {
	// Group is the api group of the resources
	Group string `json:"apiGroup,omitempty"`

	// Version is the version of the subject
	// +optional
	Version string `json:"version,omitempty"`

	// Kind is the kind of the subject
	// +optional
	Kind string `json:"kind,omitempty"`

	// Resource is the resource type of the subject
	// +optional
	Resource string `json:"resource,omitempty"`

	// Name is the name of the subject
	// +optional
	Name string `json:"name,omitempty"`

	// Name is the name of the subject
	// +optional
	Namespace string `json:"namespace,omitempty"`
}
