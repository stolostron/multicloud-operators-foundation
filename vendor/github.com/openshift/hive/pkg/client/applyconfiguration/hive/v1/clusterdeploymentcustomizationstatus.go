// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	v1 "k8s.io/api/core/v1"
)

// ClusterDeploymentCustomizationStatusApplyConfiguration represents an declarative configuration of the ClusterDeploymentCustomizationStatus type for use
// with apply.
type ClusterDeploymentCustomizationStatusApplyConfiguration struct {
	ClusterDeploymentRef     *v1.LocalObjectReference `json:"clusterDeploymentRef,omitempty"`
	ClusterPoolRef           *v1.LocalObjectReference `json:"clusterPoolRef,omitempty"`
	LastAppliedConfiguration *string                  `json:"lastAppliedConfiguration,omitempty"`
	Conditions               []conditionsv1.Condition `json:"conditions,omitempty"`
}

// ClusterDeploymentCustomizationStatusApplyConfiguration constructs an declarative configuration of the ClusterDeploymentCustomizationStatus type for use with
// apply.
func ClusterDeploymentCustomizationStatus() *ClusterDeploymentCustomizationStatusApplyConfiguration {
	return &ClusterDeploymentCustomizationStatusApplyConfiguration{}
}

// WithClusterDeploymentRef sets the ClusterDeploymentRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClusterDeploymentRef field is set to the value of the last call.
func (b *ClusterDeploymentCustomizationStatusApplyConfiguration) WithClusterDeploymentRef(value v1.LocalObjectReference) *ClusterDeploymentCustomizationStatusApplyConfiguration {
	b.ClusterDeploymentRef = &value
	return b
}

// WithClusterPoolRef sets the ClusterPoolRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ClusterPoolRef field is set to the value of the last call.
func (b *ClusterDeploymentCustomizationStatusApplyConfiguration) WithClusterPoolRef(value v1.LocalObjectReference) *ClusterDeploymentCustomizationStatusApplyConfiguration {
	b.ClusterPoolRef = &value
	return b
}

// WithLastAppliedConfiguration sets the LastAppliedConfiguration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastAppliedConfiguration field is set to the value of the last call.
func (b *ClusterDeploymentCustomizationStatusApplyConfiguration) WithLastAppliedConfiguration(value string) *ClusterDeploymentCustomizationStatusApplyConfiguration {
	b.LastAppliedConfiguration = &value
	return b
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *ClusterDeploymentCustomizationStatusApplyConfiguration) WithConditions(values ...conditionsv1.Condition) *ClusterDeploymentCustomizationStatusApplyConfiguration {
	for i := range values {
		b.Conditions = append(b.Conditions, values[i])
	}
	return b
}
