package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageRegistrySpec is the spec of managedClusterImageRegistry.
type ImageRegistrySpec struct {
	// Registry is the address of overridden image registry
	// +kubebuilder:validation:Required
	// +required
	Registry string `json:"registry"`

	// PullSecret is the name of image pull secret which should be in the same namespace with the managedClusterImageRegistry.
	// +kubebuilder:validation:Required
	// +required
	PullSecret corev1.LocalObjectReference `json:"pullSecret"`

	// PlacementRef is the referred Placement name.
	// +kubebuilder:validation:Required
	// +required
	PlacementRef PlacementRef `json:"placementRef"`
}

// PlacementRef is the referred placement
type PlacementRef struct {
	// Group is the api group of the placement. Current group is cluster.open-cluster-management.io.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=cluster.open-cluster-management.io
	// +required
	Group string `json:"group"`

	// Resource is the resource type of the Placement. Current resource is placement or placements.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=placement;placements
	// +required
	Resource string `json:"resource"`

	// Name is the name of the Placement.
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedclusterimageregistries

// ManagedClusterImageRegistry represents the image overridden configuration information.
type ManagedClusterImageRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the information of the ManagedClusterImageRegistry.
	// +required
	Spec ImageRegistrySpec `json:"spec"`

	// Status represents the desired status of the managedClusterImageRegistry.
	// +optional
	Status ImageRegistryStatus `json:"status,omitempty"`
}

type ImageRegistryStatus struct {
	// Conditions contains condition information for a managedClusterImageRegistry
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Condition Types
const (
	// ConditionPlacementAvailable reports whether the placement is available
	ConditionPlacementAvailable string = "PlacementAvailable"

	// ConditionClustersSelected reports whether the clusters are selected
	ConditionClustersSelected string = "ClustersSelected"

	// ConditionClustersUpdated reports whether the clusters are updated
	ConditionClustersUpdated string = "ClustersUpdated"
)

const (
	ConditionReasonClusterSelectedFailure string = "ClusterSelectedFailure"
	ConditionReasonClusterSelected        string = "ClusterSelected"
	ConditionReasonClustersUpdatedFailure string = "ClustersUpdatedFailure"
	ConditionReasonClustersUpdated        string = "ClustersUpdated"
)

const (
	// ClusterImageRegistryLabel value is namespace.managedClusterImageRegistry
	ClusterImageRegistryLabel = "open-cluster-management.io/image-registry"
)

// +kubebuilder:object:root=true

// ManagedClusterImageRegistryList is a list of ManagedClusterImageRegistry objects.
type ManagedClusterImageRegistryList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ManagedClusterInfo objects.
	Items []ManagedClusterImageRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedClusterImageRegistry{}, &ManagedClusterImageRegistryList{})
}
