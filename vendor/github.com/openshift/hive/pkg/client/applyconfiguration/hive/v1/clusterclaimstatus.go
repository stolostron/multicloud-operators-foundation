// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterClaimStatusApplyConfiguration represents an declarative configuration of the ClusterClaimStatus type for use
// with apply.
type ClusterClaimStatusApplyConfiguration struct {
	Conditions []ClusterClaimConditionApplyConfiguration `json:"conditions,omitempty"`
	Lifetime   *metav1.Duration                          `json:"lifetime,omitempty"`
}

// ClusterClaimStatusApplyConfiguration constructs an declarative configuration of the ClusterClaimStatus type for use with
// apply.
func ClusterClaimStatus() *ClusterClaimStatusApplyConfiguration {
	return &ClusterClaimStatusApplyConfiguration{}
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *ClusterClaimStatusApplyConfiguration) WithConditions(values ...*ClusterClaimConditionApplyConfiguration) *ClusterClaimStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}

// WithLifetime sets the Lifetime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Lifetime field is set to the value of the last call.
func (b *ClusterClaimStatusApplyConfiguration) WithLifetime(value metav1.Duration) *ClusterClaimStatusApplyConfiguration {
	b.Lifetime = &value
	return b
}
