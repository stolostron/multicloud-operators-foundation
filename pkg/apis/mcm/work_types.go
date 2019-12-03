// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MaxRetries is the max retry time
const MaxRetries = 10

//RetryDelayUnit is the retry delay unit for each retry.
const RetryDelayUnit = 3 * time.Second

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Work is the work that will be done on a set of cluster
type Work struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Spec defines the behavior of the work.
	// +optional
	Spec WorkSpec
	// Result describes the result of a work
	// +optional
	Status WorkStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkList is a list of all the works
type WorkList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta

	// List of Cluster objects.
	Items []Work
}

// WorkSpec defines the work to be processes on a set of clusters
type WorkSpec struct {
	// Cluster is a selector of cluster
	Cluster v1.LocalObjectReference

	// Type defins the type of the woke to be done
	Type WorkType

	// Scope is the scope of the work to be apply to in a cluster
	Scope ResourceFilter

	// ActionType is the type of the action
	ActionType ActionType

	// HelmWork is the work to process helm operation
	// +optional
	HelmWork *HelmWorkSpec

	// KubeWorkSpec is the work to process kubernetes operation
	KubeWork *KubeWorkSpec
}

// WorkStatus returns the status of the work
type WorkStatus struct {
	// Status is the status of the work result
	Type WorkStatusType

	//Retried is the retry count for failed work
	Retried int

	// Reason is the reason of the current status
	Reason string

	// WorkResult references the related result of the work
	Result runtime.RawExtension

	// LastUpdateTime is the last status update time
	LastUpdateTime metav1.Time
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
	// CreateActionType action
	CreateActionType ActionType = "Create"
	// DeleteActionType action
	DeleteActionType ActionType = "Delete"
	// UpdateActionType action
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
	LabelSelector *metav1.LabelSelector

	// FieldSelector is a selector that select a set of resources
	// +optional
	FieldSelector string

	// APIGroup is the api group of the resources
	APIGroup string

	// ResouceType is the resource type of the subject
	// +optional
	ResourceType string

	// Name is the name of the subject
	// +optional
	Name string

	// Name is the name of the subject
	// +optional
	NameSpace string

	// Version is the version of the subject
	// +optional
	Version string

	// ServerPrint is the flag to set print on server side
	// +optional
	ServerPrint bool

	// Mode is the mode for resource query
	// +optional
	Mode ResourceFilterMode

	// UpdateIntervalSeconds
	// +optional
	UpdateIntervalSeconds int
}

// HelmWorkSpec is the helm work details
type HelmWorkSpec struct {
	// ReleaseName
	ReleaseName string

	//InSecureSkipVerify
	InSecureSkipVerify bool

	// ChartName
	ChartName string

	// Version
	Version string

	// Chart url
	ChartURL string

	// Namespace
	Namespace string

	// Values
	Values []byte

	// ValuesURL url to a file contains value
	ValuesURL string
}

// KubeWorkSpec is the kubernetes work details
type KubeWorkSpec struct {
	// Resource of the object
	Resource string

	// Name of the object
	Name string

	// Namespace of the object
	Namespace string

	// ObjectTemplate is the template of the object
	ObjectTemplate runtime.RawExtension
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResultHelmList is the list of helm release in one cluster
type ResultHelmList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta
	// Items are the items list of helm release
	Items []HelmRelease
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HelmRelease is the helm release info
type HelmRelease struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Spec defines the behavior of the helm release.
	Spec HelmReleaseSpec
}

// HelmReleaseSpec is the details of helm release
type HelmReleaseSpec struct {
	// ReleaseName
	ReleaseName string

	// Description
	Description string

	// FirstDeployed
	FirstDeployed metav1.Time

	// LastDeployed
	LastDeployed metav1.Time

	// Manifest
	Manifest string

	// ChartName
	ChartName string

	// ChartVersion
	ChartVersion string

	// Namespace
	Namespace string

	// Version
	Version int32

	// Status
	Status string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewResultList is a list of all the resource view result
type ResourceViewResultList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta

	// List of Cluster objects.
	Items []ResourceViewResult
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceViewResult is the view result of resources on a set of cluster
type ResourceViewResult struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Spec defines the behavior of the work.
	// +optional
	Data []byte
}
