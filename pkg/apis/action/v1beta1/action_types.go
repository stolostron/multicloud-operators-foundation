package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ManagedClusterAction is the action that will be done on a cluster
type ManagedClusterAction struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired behavior of the action.
	// +optional
	Spec ActionSpec `json:"spec,omitempty"`
	// Status describes the desired status of the action
	// +optional
	Status ActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedClusterActionList is a list of all the ManagedClusterActions
type ManagedClusterActionList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ManagedClusterAction objects.
	Items []ManagedClusterAction `json:"items"`
}

// ActionSpec defines the action to be processed on a cluster
type ActionSpec struct {
	// ActionType is the type of the action
	ActionType ActionType `json:"actionType,omitempty"`

	// KubeWorkSpec is the action payload to process
	KubeWork *KubeWorkSpec `json:"kube,omitempty"`
}

// ActionStatus returns the current status of the action
type ActionStatus struct {
	// Conditions represents the conditions of this resource on managed cluster
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Result references the related result of the action
	// +nullable
	// +optional
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	Result runtime.RawExtension `json:"result,omitempty"`
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

// These are valid conditions of a cluster.
const (
	// ConditionActionCompleted means the work is completed.
	ConditionActionCompleted string = "Completed"
)

const (
	ReasonCreateResourceFailed string = "CreateResourceFailed"
	ReasonUpdateResourceFailed string = "UpdateResourceFailed"
	ReasonDeleteResourceFailed string = "DeleteResourceFailed"
	ReasonActionTypeInvalid    string = "ActionTypeInvalid"
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
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	ObjectTemplate runtime.RawExtension `json:"template,omitempty"`
}

const (
	// UserIdentityAnnotation is identity annotation
	UserIdentityAnnotation = "acm.io/user-identity"

	// UserGroupAnnotation is user group annotation
	UserGroupAnnotation = "acm.io/user-group"
)

func init() {
	SchemeBuilder.Register(&ManagedClusterAction{}, &ManagedClusterActionList{})
}
