package conditions

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition represents the state of the operator's
// reconciliation functionality.
// +k8s:deepcopy-gen=true
type Condition struct {
	Type ConditionType `json:"type" description:"type of condition ie. Available|Progressing|Degraded."`

	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime" description:"last time the condition transit from one status to another"`
}

// ConditionType is the state of the operator's reconciliation functionality.
type ConditionType string
