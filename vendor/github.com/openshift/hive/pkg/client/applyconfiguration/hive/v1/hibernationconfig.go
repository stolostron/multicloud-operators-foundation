// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HibernationConfigApplyConfiguration represents an declarative configuration of the HibernationConfig type for use
// with apply.
type HibernationConfigApplyConfiguration struct {
	ResumeTimeout *v1.Duration `json:"resumeTimeout,omitempty"`
}

// HibernationConfigApplyConfiguration constructs an declarative configuration of the HibernationConfig type for use with
// apply.
func HibernationConfig() *HibernationConfigApplyConfiguration {
	return &HibernationConfigApplyConfiguration{}
}

// WithResumeTimeout sets the ResumeTimeout field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ResumeTimeout field is set to the value of the last call.
func (b *HibernationConfigApplyConfiguration) WithResumeTimeout(value v1.Duration) *HibernationConfigApplyConfiguration {
	b.ResumeTimeout = &value
	return b
}
