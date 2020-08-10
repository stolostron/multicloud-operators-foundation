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
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=mirroredmanagedclusters
// +kubebuilder:printcolumn:JSONPath=`.spec.hubAcceptsClient`,name="Hub Accepted",type=boolean
// +kubebuilder:printcolumn:JSONPath=`.spec.managedClusterClientConfigs[*].url`,name="Managed Cluster URLs",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.conditions[?(@.type=="ManagedClusterJoined")].status`,name="Joined",type=string
// +kubebuilder:printcolumn:JSONPath=`.status.conditions[?(@.type=="ManagedClusterConditionAvailable")].status`,name="Available",type=string
// +kubebuilder:printcolumn:JSONPath=`.metadata.creationTimestamp`,name="Age",type=date

// MirroredManagedCluster is a mirror of a managed cluster in namespace scope.
// It is ready only and keeps synced with the managed cluster it mirrored.
type MirroredManagedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents a desired configuration for the agent on the managed cluster.
	Spec clusterv1.ManagedClusterSpec `json:"spec"`

	// Status represents the current status of joined managed cluster
	// +optional
	Status clusterv1.ManagedClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MirroredManagedClusterList is a collection of mirrored managed clusters.
type MirroredManagedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of mirrored managed clusters.
	Items []MirroredManagedCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MirroredManagedCluster{}, &MirroredManagedClusterList{})
}
