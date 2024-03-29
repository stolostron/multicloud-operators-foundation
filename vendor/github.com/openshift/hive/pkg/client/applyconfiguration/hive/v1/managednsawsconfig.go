// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "k8s.io/api/core/v1"
)

// ManageDNSAWSConfigApplyConfiguration represents an declarative configuration of the ManageDNSAWSConfig type for use
// with apply.
type ManageDNSAWSConfigApplyConfiguration struct {
	CredentialsSecretRef *v1.LocalObjectReference `json:"credentialsSecretRef,omitempty"`
	Region               *string                  `json:"region,omitempty"`
}

// ManageDNSAWSConfigApplyConfiguration constructs an declarative configuration of the ManageDNSAWSConfig type for use with
// apply.
func ManageDNSAWSConfig() *ManageDNSAWSConfigApplyConfiguration {
	return &ManageDNSAWSConfigApplyConfiguration{}
}

// WithCredentialsSecretRef sets the CredentialsSecretRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the CredentialsSecretRef field is set to the value of the last call.
func (b *ManageDNSAWSConfigApplyConfiguration) WithCredentialsSecretRef(value v1.LocalObjectReference) *ManageDNSAWSConfigApplyConfiguration {
	b.CredentialsSecretRef = &value
	return b
}

// WithRegion sets the Region field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Region field is set to the value of the last call.
func (b *ManageDNSAWSConfigApplyConfiguration) WithRegion(value string) *ManageDNSAWSConfigApplyConfiguration {
	b.Region = &value
	return b
}
