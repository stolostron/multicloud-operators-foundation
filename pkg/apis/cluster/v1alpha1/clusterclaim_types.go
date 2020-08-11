/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedclusterclaims

// ManagedClusterClaim is a user's claim request to a managed cluster.
// It is defined as an ownership claim of a managed cluster in a namespace.
type ManagedClusterClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the managed cluster requested by the claim
	Spec ManagedClusterClaimSpec `json:"spec"`

	// Status represents the current status of the claim
	// +optional
	Status ManagedClusterClaimStatus `json:"status,omitempty"`
}

// ManagedClusterClaimSpec describes the attributes of desired managed cluster
type ManagedClusterClaimSpec struct {
	// A label query over managed clusters to consider for binding. This selector is
	// ignored when ClusterName is set
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	// ClusterName is used to match a concrete managed cluster for this
	// claim. When set to non-empty value Selector is not evaluated
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
}

// ManagedClusterClaimStatus represents the status of ManagedCluster claim
type ManagedClusterClaimStatus struct {
	// ClusterName is the binding reference to the ManagedCluster backing this
	// claim
	// +optional
	ClusterName string `json:"clusterName,omitempty"`
	// +optional
	Conditions []ManagedClusterClaimCondition `json:"conditions,omitempty"`
}

// ManagedClusterClaimCondition represents the current condition of ManagedCluster claim
type ManagedClusterClaimCondition struct {
	// Type is the type of the ManagedClusterClaim condition.
	// +required
	Type string `json:"type"`
	// Status is the status of the condition. One of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus `json:"status"`
	// LastTransitionTime is the last time the condition changed from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Reason is a (brief) reason for the condition's last status change.
	// +required
	Reason string `json:"reason"`
	// Message is a human-readable message indicating details about the last status change.
	// +required
	Message string `json:"message"`
}

// ManagedClusterClaimConditionType defines the condition of ManagedCluster claim.
type ManagedClusterClaimConditionType string

// These are valid conditions of ManagedClusterClaim
const (
	// ManagedClusterClaim is bound to a managed cluster
	ClaimBound ManagedClusterClaimConditionType = "ClaimBound"

	// A mirrored managed cluster is created in the same namespace of claim
	// and it will keep synced with the source managed cluster
	ClusterMirrored ManagedClusterClaimConditionType = "ClusterMirrored"
)

// +kubebuilder:object:root=true

// ManagedClusterClaimList is a collection of ManagedClusterClaims.
type ManagedClusterClaimList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of ManagedClusterClaims.
	Items []ManagedClusterClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedClusterClaim{}, &ManagedClusterClaimList{})
}
