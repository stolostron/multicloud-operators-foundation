package v1beta1

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"
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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the desired behavior of the action.
	// +optional
	Spec ActionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the desired status of the action
	// +optional
	Status ActionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +kubebuilder:object:root=true

// ManagedClusterActionList is a list of all the ManagedClusterActions
type ManagedClusterActionList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of ManagedClusterAction objects.
	Items []ManagedClusterAction `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ActionSpec defines the action to be processed on a cluster
type ActionSpec struct {
	// ActionType is the type of the action
	ActionType ActionType `json:"actionType,omitempty" protobuf:"bytes,1,opt,name=actionType"`

	// KubeWorkSpec is the action payload to process
	KubeWork *KubeWorkSpec `json:"kube,omitempty" protobuf:"bytes,2,opt,name=kube"`
}

// ActionStatus returns the current status of the action
type ActionStatus struct {
	// Conditions represents the conditions of this resource on managed cluster
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	Conditions []conditions.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Result references the related result of the action
	// +nullable
	// +optional
	Result runtime.RawExtension `json:"result,omitempty" protobuf:"bytes,2,opt,name=result"`
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
	ConditionActionCompleted conditions.ConditionType = "Completed"
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
	Resource string `json:"resource,omitempty" protobuf:"bytes,1,opt,name=resource"`

	// Name of the object
	Name string `json:"name,omitempty" protobuf:"bytes,2,opt,name=name"`

	// Namespace of the object
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`

	// ObjectTemplate is the template of the object
	ObjectTemplate runtime.RawExtension `json:"template,omitempty" protobuf:"bytes,4,opt,name=template"`
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
