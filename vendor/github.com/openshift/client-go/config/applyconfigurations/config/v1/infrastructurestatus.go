// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	configv1 "github.com/openshift/api/config/v1"
)

// InfrastructureStatusApplyConfiguration represents a declarative configuration of the InfrastructureStatus type for use
// with apply.
type InfrastructureStatusApplyConfiguration struct {
	InfrastructureName     *string                           `json:"infrastructureName,omitempty"`
	Platform               *configv1.PlatformType            `json:"platform,omitempty"`
	PlatformStatus         *PlatformStatusApplyConfiguration `json:"platformStatus,omitempty"`
	EtcdDiscoveryDomain    *string                           `json:"etcdDiscoveryDomain,omitempty"`
	APIServerURL           *string                           `json:"apiServerURL,omitempty"`
	APIServerInternalURL   *string                           `json:"apiServerInternalURI,omitempty"`
	ControlPlaneTopology   *configv1.TopologyMode            `json:"controlPlaneTopology,omitempty"`
	InfrastructureTopology *configv1.TopologyMode            `json:"infrastructureTopology,omitempty"`
	CPUPartitioning        *configv1.CPUPartitioningMode     `json:"cpuPartitioning,omitempty"`
}

// InfrastructureStatusApplyConfiguration constructs a declarative configuration of the InfrastructureStatus type for use with
// apply.
func InfrastructureStatus() *InfrastructureStatusApplyConfiguration {
	return &InfrastructureStatusApplyConfiguration{}
}

// WithInfrastructureName sets the InfrastructureName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InfrastructureName field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithInfrastructureName(value string) *InfrastructureStatusApplyConfiguration {
	b.InfrastructureName = &value
	return b
}

// WithPlatform sets the Platform field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Platform field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithPlatform(value configv1.PlatformType) *InfrastructureStatusApplyConfiguration {
	b.Platform = &value
	return b
}

// WithPlatformStatus sets the PlatformStatus field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PlatformStatus field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithPlatformStatus(value *PlatformStatusApplyConfiguration) *InfrastructureStatusApplyConfiguration {
	b.PlatformStatus = value
	return b
}

// WithEtcdDiscoveryDomain sets the EtcdDiscoveryDomain field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EtcdDiscoveryDomain field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithEtcdDiscoveryDomain(value string) *InfrastructureStatusApplyConfiguration {
	b.EtcdDiscoveryDomain = &value
	return b
}

// WithAPIServerURL sets the APIServerURL field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the APIServerURL field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithAPIServerURL(value string) *InfrastructureStatusApplyConfiguration {
	b.APIServerURL = &value
	return b
}

// WithAPIServerInternalURL sets the APIServerInternalURL field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the APIServerInternalURL field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithAPIServerInternalURL(value string) *InfrastructureStatusApplyConfiguration {
	b.APIServerInternalURL = &value
	return b
}

// WithControlPlaneTopology sets the ControlPlaneTopology field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ControlPlaneTopology field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithControlPlaneTopology(value configv1.TopologyMode) *InfrastructureStatusApplyConfiguration {
	b.ControlPlaneTopology = &value
	return b
}

// WithInfrastructureTopology sets the InfrastructureTopology field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the InfrastructureTopology field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithInfrastructureTopology(value configv1.TopologyMode) *InfrastructureStatusApplyConfiguration {
	b.InfrastructureTopology = &value
	return b
}

// WithCPUPartitioning sets the CPUPartitioning field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CPUPartitioning field is set to the value of the last call.
func (b *InfrastructureStatusApplyConfiguration) WithCPUPartitioning(value configv1.CPUPartitioningMode) *InfrastructureStatusApplyConfiguration {
	b.CPUPartitioning = &value
	return b
}
