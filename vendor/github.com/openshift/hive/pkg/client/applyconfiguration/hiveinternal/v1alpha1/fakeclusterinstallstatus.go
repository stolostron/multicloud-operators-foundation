// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "github.com/openshift/hive/pkg/client/applyconfiguration/hive/v1"
)

// FakeClusterInstallStatusApplyConfiguration represents an declarative configuration of the FakeClusterInstallStatus type for use
// with apply.
type FakeClusterInstallStatusApplyConfiguration struct {
	Conditions []v1.ClusterInstallConditionApplyConfiguration `json:"conditions,omitempty"`
}

// FakeClusterInstallStatusApplyConfiguration constructs an declarative configuration of the FakeClusterInstallStatus type for use with
// apply.
func FakeClusterInstallStatus() *FakeClusterInstallStatusApplyConfiguration {
	return &FakeClusterInstallStatusApplyConfiguration{}
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *FakeClusterInstallStatusApplyConfiguration) WithConditions(values ...*v1.ClusterInstallConditionApplyConfiguration) *FakeClusterInstallStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}
