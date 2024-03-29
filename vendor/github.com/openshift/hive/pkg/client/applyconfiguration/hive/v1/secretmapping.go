// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// SecretMappingApplyConfiguration represents an declarative configuration of the SecretMapping type for use
// with apply.
type SecretMappingApplyConfiguration struct {
	SourceRef *SecretReferenceApplyConfiguration `json:"sourceRef,omitempty"`
	TargetRef *SecretReferenceApplyConfiguration `json:"targetRef,omitempty"`
}

// SecretMappingApplyConfiguration constructs an declarative configuration of the SecretMapping type for use with
// apply.
func SecretMapping() *SecretMappingApplyConfiguration {
	return &SecretMappingApplyConfiguration{}
}

// WithSourceRef sets the SourceRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SourceRef field is set to the value of the last call.
func (b *SecretMappingApplyConfiguration) WithSourceRef(value *SecretReferenceApplyConfiguration) *SecretMappingApplyConfiguration {
	b.SourceRef = value
	return b
}

// WithTargetRef sets the TargetRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TargetRef field is set to the value of the last call.
func (b *SecretMappingApplyConfiguration) WithTargetRef(value *SecretReferenceApplyConfiguration) *SecretMappingApplyConfiguration {
	b.TargetRef = value
	return b
}
