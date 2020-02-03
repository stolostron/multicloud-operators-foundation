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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Work is the work that will be done on a set of cluster
type Work struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the work.
	// +optional
	Spec WorkSpec `json:"spec,omitempty"`
	// Result describes the result of a work
	// +optional
	Status WorkStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkList is a list of all the works
type WorkList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Cluster objects.
	Items []Work `json:"items"`
}

// WorkSpec defines the work to be processes on a set of clusters
type WorkSpec struct {
	// Cluster is a selector of cluster
	Cluster v1.LocalObjectReference `json:"cluster,omitempty"`

	// Type defins the type of the woke to be done
	Type WorkType `json:"type,omitempty"`

	// Scope is the scope of the work to be apply to in a cluster
	Scope ResourceFilter `json:"scope,omitempty"`

	// ActionType is the type of the action
	ActionType ActionType `json:"actionType,omitempty"`

	// HelmWork is the work to process helm operation
	// +optional
	HelmWork *HelmWorkSpec `json:"helm,omitempty"`

	// KubeWorkSpec is the work to process kubernetes operation
	KubeWork *KubeWorkSpec `json:"kube,omitempty"`
}

// WorkStatus returns the status of the work
type WorkStatus struct {
	// Status is the status of the work result
	Type WorkStatusType `json:"type,omitempty"`

	// Reason is the reason of the current status
	Reason string `json:"reason,omitempty"`

	// WorkResult references the related result of the work
	Result runtime.RawExtension `json:"result,omitempty"`

	// LastUpdateTime is the last status update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// WorkType defines the type of cluster status
type WorkType string

// These are types of the work.
const (
	// PolicyWork
	ResourceWorkType WorkType = "Resource"

	// action work
	ActionWorkType WorkType = "Action"
)

// ActionType defines the type of the action
type ActionType string

const (
	// CreateActionType defines create action
	CreateActionType ActionType = "Create"
	// DeleteActionType defines selete action
	DeleteActionType ActionType = "Delete"
	// UpdateActionType defines update action
	UpdateActionType ActionType = "Update"
)

// ResourceFilterMode is the mode to update resource by work
type ResourceFilterMode string

const (
	// PeriodicResourceUpdate is a periodic mode
	PeriodicResourceUpdate ResourceFilterMode = "Periodic"
)

// ResourceFilter is the filter of resources
type ResourceFilter struct {
	// LabelSelect is a selector that selects a set of resources
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// FieldSelector is a selector that select a set of resources
	// +optional
	FieldSelector string `json:"fieldSelector,omitempty"`

	// APIGroup is the api group of the resources
	APIGroup string `json:"apiGroup,omitempty"`

	// ResouceType is the resource type of the subject
	// +optional
	ResourceType string `json:"resourceType,omitempty"`

	// Name is the name of the subject
	// +optional
	Name string `json:"name,omitempty"`

	// Name is the name of the subject
	// +optional
	NameSpace string `json:"namespace,omitempty"`

	// Version is the version of the subject
	// +optional
	Version string `json:"version,omitempty"`

	// ServerPrint is the flag to set print on server side
	// +optional
	ServerPrint bool `json:"serverPrint,omitempty"`

	// Mode is the mode for resource query
	// +optional
	Mode ResourceFilterMode `json:"mode,omitempty"`

	// UpdateIntervalSeconds
	// +optional
	UpdateIntervalSeconds int `json:"updateIntervalSeconds,omitempty"`
}

// HelmWorkSpec is the helm work details
type HelmWorkSpec struct {
	// ReleaseName
	ReleaseName string `json:"releaseName,omitempty"`

	//InSecureSkipVerify skip verification
	InSecureSkipVerify bool `json:"inSecureSkipVerify,omitempty"`

	// ChartName
	ChartName string `json:"chartName,omitempty"`

	// Version
	Version string `json:"version,omitempty"`

	// Chart url
	ChartURL string `json:"chartURL,omitempty"`

	// Namespace
	Namespace string `json:"namespace,omitempty"`

	// Values
	Values []byte `json:"values,omitempty"`

	// ValuesURL url to a file contains value
	ValuesURL string `json:"valuesURL,omitempty"`
}

// WorkStatusType defines the type of work status
type WorkStatusType string

// These are valid conditions of a cluster.
const (
	// WorkCompleted means the work is comleted.
	WorkCompleted WorkStatusType = "Completed"
	// WorkFailed means the work fails to execute
	WorkFailed WorkStatusType = "Failed"
	// WorkProcessing means the work is in process
	WorkProcessing WorkStatusType = "Processing"
)

// KubeWorkSpec is the kubernetes work details
type KubeWorkSpec struct {
	// Resource of the object
	Resource string `json:"resource,omitempty"`

	// Name of the object
	Name string `json:"name,omitempty"`

	// Namespace of the object
	Namespace string `json:"namespace,omitempty"`

	// ObjectTemplate is the template of the object
	ObjectTemplate runtime.RawExtension `json:"template,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResultHelmList is the list of helm release in one cluster
type ResultHelmList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items are the items list of helm release
	Items []HelmRelease `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmRelease is the helm release info
type HelmRelease struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the helm release.
	Spec HelmReleaseSpec `json:"spec,omitempty"`
}

// HelmReleaseSpec is the details of helm release
type HelmReleaseSpec struct {
	// ReleaseName
	ReleaseName string `json:"releaseName,omitempty"`

	// Description
	Description string `json:"description,omitempty"`

	// FirstDeployed
	FirstDeployed metav1.Time `json:"firstDeployed,omitempty"`

	// LastDeployed
	LastDeployed metav1.Time `json:"lastDeployed,omitempty"`

	// Manifest
	Manifest string `json:"manifest,omitempty"`

	// ChartName
	ChartName string `json:"chartName,omitempty"`

	// ChartVersion
	ChartVersion string `json:"chartVersion,omitempty"`

	// Namespace
	Namespace string `json:"namespace,omitempty"`

	// Version
	Version int32 `json:"version,omitempty"`

	// Status
	Status string `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewResultList is a list of all the resource view result
type ResourceViewResultList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Cluster objects.
	Items []ResourceViewResult `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewResult is the view result of resources on a set of cluster
type ResourceViewResult struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the work.
	// +optional
	Data []byte `json:"data,omitempty"`
}
