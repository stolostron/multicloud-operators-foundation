// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "github.com/openshift/hive/pkg/client/applyconfiguration/hive/v1"
	corev1 "k8s.io/api/core/v1"
)

// FakeClusterInstallSpecApplyConfiguration represents an declarative configuration of the FakeClusterInstallSpec type for use
// with apply.
type FakeClusterInstallSpecApplyConfiguration struct {
	ImageSetRef          *v1.ClusterImageSetReferenceApplyConfiguration `json:"imageSetRef,omitempty"`
	ClusterDeploymentRef *corev1.LocalObjectReference                   `json:"clusterDeploymentRef,omitempty"`
	ClusterMetadata      *v1.ClusterMetadataApplyConfiguration          `json:"clusterMetadata,omitempty"`
}

// FakeClusterInstallSpecApplyConfiguration constructs an declarative configuration of the FakeClusterInstallSpec type for use with
// apply.
func FakeClusterInstallSpec() *FakeClusterInstallSpecApplyConfiguration {
	return &FakeClusterInstallSpecApplyConfiguration{}
}

// WithImageSetRef sets the ImageSetRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ImageSetRef field is set to the value of the last call.
func (b *FakeClusterInstallSpecApplyConfiguration) WithImageSetRef(value *v1.ClusterImageSetReferenceApplyConfiguration) *FakeClusterInstallSpecApplyConfiguration {
	b.ImageSetRef = value
	return b
}

// WithClusterDeploymentRef sets the ClusterDeploymentRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClusterDeploymentRef field is set to the value of the last call.
func (b *FakeClusterInstallSpecApplyConfiguration) WithClusterDeploymentRef(value corev1.LocalObjectReference) *FakeClusterInstallSpecApplyConfiguration {
	b.ClusterDeploymentRef = &value
	return b
}

// WithClusterMetadata sets the ClusterMetadata field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClusterMetadata field is set to the value of the last call.
func (b *FakeClusterInstallSpecApplyConfiguration) WithClusterMetadata(value *v1.ClusterMetadataApplyConfiguration) *FakeClusterInstallSpecApplyConfiguration {
	b.ClusterMetadata = value
	return b
}
