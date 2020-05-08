// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
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

// SpokeViewStatus returns the status of the spoke view
type SpokeViewStatus struct {
	// Conditions represents the conditions of this resource on spoke cluster
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// WorkResult references the related result of the work
	// +nullable
	// +optional
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

	// UpdateIntervalSeconds is the interval to update view
	// +optional
	UpdateIntervalSeconds int32 `json:"updateIntervalSeconds,omitempty"`
}

// These are valid conditions of a cluster.
const (
	// ConditionViewProcessing means the spoke view is processing.
	ConditionViewProcessing conditionsv1.ConditionType = "Processing"
)

const (
	ReasonResourceNameInvalid string = "ResourceNameInvalid"
	ReasonResourceTypeInvalid string = "ResourceTypeInvalid"
	ReasonResourceGVKInvalid  string = "ResourceGVKInvalid"
	ReasonGetResourceFailed   string = "GetResourceFailed"
)

func init() {
	SchemeBuilder.Register(&SpokeView{}, &SpokeViewList{})
}
