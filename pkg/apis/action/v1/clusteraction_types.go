// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ClusterAction is the action that will be done on a cluster
type ClusterAction struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired behavior of the action.
	// +optional
	Spec ClusterActionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the desired status of the action
	// +optional
	Status ClusterActionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterActionList is a list of all the actions
type ClusterActionList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of ClusterAction objects.
	Items []ClusterAction `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ClusterActionSpec defines the action to be processed on a cluster
type ClusterActionSpec struct {
	// ActionType is the type of the action
	ActionType ActionType `json:"actionType,omitempty" protobuf:"bytes,4,opt,name=actionType"`

	// KubeWorkSpec is the action payload to process
	KubeWork *KubeWorkSpec `json:"kube,omitempty" protobuf:"bytes,6,opt,name=kube"`
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

// WorkStatus returns the current status of the action
type ClusterActionStatus struct {
	// Conditions represents the conditions of this resource on spoke cluster
	// +required
	Conditions []StatusCondition `json:"conditions"`

	// WorkResult references the related result of the work
	Result runtime.RawExtension `json:"result,omitempty" protobuf:"bytes,3,opt,name=result"`
}

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
