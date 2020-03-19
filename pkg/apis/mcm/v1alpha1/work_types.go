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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Work is the work that will be done on a set of cluster
type Work struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the work.
	// +optional
	Spec WorkSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Result describes the result of a work
	// +optional
	Status WorkStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkList is a list of all the works
type WorkList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Cluster objects.
	Items []Work `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// WorkSpec defines the work to be processes on a set of clusters
type WorkSpec struct {
	// Cluster is a selector of cluster
	Cluster v1.LocalObjectReference `json:"cluster,omitempty" protobuf:"bytes,1,opt,name=cluster"`

	// Type defins the type of the woke to be done
	Type WorkType `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`

	// Scope is the scope of the work to be apply to in a cluster
	Scope ResourceFilter `json:"scope,omitempty" protobuf:"bytes,3,opt,name=scope"`

	// ActionType is the type of the action
	ActionType ActionType `json:"actionType,omitempty" protobuf:"bytes,4,opt,name=actionType"`

	// HelmWork is the work to process helm operation
	// +optional
	HelmWork *HelmWorkSpec `json:"helm,omitempty" protobuf:"bytes,5,opt,name=helm"`

	// KubeWorkSpec is the work to process kubernetes operation
	KubeWork *KubeWorkSpec `json:"kube,omitempty" protobuf:"bytes,6,opt,name=kube"`
}

// WorkStatus returns the status of the work
type WorkStatus struct {
	// Status is the status of the work result
	Type WorkStatusType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`

	// Reason is the reason of the current status
	Reason string `json:"reason,omitempty" protobuf:"bytes,2,opt,name=reason"`

	// WorkResult references the related result of the work
	Result runtime.RawExtension `json:"result,omitempty" protobuf:"bytes,3,opt,name=result"`

	// LastUpdateTime is the last status update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,4,opt,name=lastUpdateTime"`
}

// WorkType defines the type of cluster status
type WorkType string

// These are types of the work.
const (
	// resource work
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
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,1,opt,name=labelSelector"`

	// FieldSelector is a selector that select a set of resources
	// +optional
	FieldSelector string `json:"fieldSelector,omitempty" protobuf:"bytes,2,opt,name=fieldSelector"`

	// APIGroup is the api group of the resources
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,3,opt,name=apiGroup"`

	// ResouceType is the resource type of the subject
	// +optional
	ResourceType string `json:"resourceType,omitempty" protobuf:"bytes,4,opt,name=resourceType"`

	// Name is the name of the subject
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,5,opt,name=name"`

	// Name is the name of the subject
	// +optional
	NameSpace string `json:"namespace,omitempty" protobuf:"bytes,6,opt,name=namespace"`

	// Version is the version of the subject
	// +optional
	Version string `json:"version,omitempty" protobuf:"bytes,7,opt,name=version"`

	// ServerPrint is the flag to set print on server side
	// +optional
	ServerPrint bool `json:"serverPrint,omitempty" protobuf:"bytes,8,opt,name=serverPrint"`

	// Mode is the mode for resource query
	// +optional
	Mode ResourceFilterMode `json:"mode,omitempty" protobuf:"bytes,9,opt,name=mode"`

	// UpdateIntervalSeconds
	// +optional
	UpdateIntervalSeconds int32 `json:"updateIntervalSeconds,omitempty" protobuf:"varint,10,opt,name=updateIntervalSeconds"`
}

// HelmWorkSpec is the helm work details
type HelmWorkSpec struct {
	// ReleaseName
	ReleaseName string `json:"releaseName,omitempty" protobuf:"bytes,1,opt,name=releaseName"`

	//InSecureSkipVerify skip verification
	InSecureSkipVerify bool `json:"inSecureSkipVerify,omitempty" protobuf:"bool,2,opt,name=inSecureSkipVerify"`

	// ChartName
	ChartName string `json:"chartName,omitempty" protobuf:"bytes,3,opt,name=chartName"`

	// Version
	Version string `json:"version,omitempty" protobuf:"bytes,4,opt,name=version"`

	// Chart url
	ChartURL string `json:"chartURL,omitempty" protobuf:"bytes,5,opt,name=chartURL"`

	// Namespace
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,6,opt,name=namespace"`

	// Values
	Values []byte `json:"values,omitempty" protobuf:"bytes,7,rep,name=values"`

	// ValuesURL url to a file contains value
	ValuesURL string `json:"valuesURL,omitempty" protobuf:"bytes,8,opt,name=valuesURL"`
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
	Resource string `json:"resource,omitempty" protobuf:"bytes,1,opt,name=resource"`

	// Name of the object
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`

	// Namespace of the object
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`

	// ObjectTemplate is the template of the object
	ObjectTemplate runtime.RawExtension `json:"template,omitempty" protobuf:"bytes,4,opt,name=template"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResultHelmList is the list of helm release in one cluster
type ResultHelmList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Items are the items list of helm release
	Items []HelmRelease `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmRelease is the helm release info
type HelmRelease struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the helm release.
	Spec HelmReleaseSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// HelmReleaseSpec is the details of helm release
type HelmReleaseSpec struct {
	// ReleaseName
	ReleaseName string `json:"releaseName,omitempty" protobuf:"bytes,1,opt,name=releaseName"`

	// Description
	Description string `json:"description,omitempty" protobuf:"bytes,2,opt,name=description"`

	// FirstDeployed
	FirstDeployed metav1.Time `json:"firstDeployed,omitempty" protobuf:"bytes,3,opt,name=firstDeployed"`

	// LastDeployed
	LastDeployed metav1.Time `json:"lastDeployed,omitempty" protobuf:"bytes,4,opt,name=lastDeployed"`

	// Manifest
	Manifest string `json:"manifest,omitempty" protobuf:"bytes,5,opt,name=manifest"`

	// ChartName
	ChartName string `json:"chartName,omitempty" protobuf:"bytes,6,opt,name=chartName"`

	// ChartVersion
	ChartVersion string `json:"chartVersion,omitempty" protobuf:"bytes,7,opt,name=chartVersion"`

	// Namespace
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,8,opt,name=namespace"`

	// Version
	Version int32 `json:"version,omitempty" protobuf:"varint,9,opt,name=version"`

	// Status
	Status string `json:"status,omitempty" protobuf:"bytes,10,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewResultList is a list of all the resource view result
type ResourceViewResultList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Cluster objects.
	Items []ResourceViewResult `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewResult is the view result of resources on a set of cluster
type ResourceViewResult struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the work.
	// +optional
	Data []byte `json:"data,omitempty" protobuf:"bytes,2,rep,name=data"`
}
