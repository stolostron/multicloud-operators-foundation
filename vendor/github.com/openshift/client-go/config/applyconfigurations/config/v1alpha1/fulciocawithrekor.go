// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// FulcioCAWithRekorApplyConfiguration represents a declarative configuration of the FulcioCAWithRekor type for use
// with apply.
type FulcioCAWithRekorApplyConfiguration struct {
	FulcioCAData  []byte                                 `json:"fulcioCAData,omitempty"`
	RekorKeyData  []byte                                 `json:"rekorKeyData,omitempty"`
	FulcioSubject *PolicyFulcioSubjectApplyConfiguration `json:"fulcioSubject,omitempty"`
}

// FulcioCAWithRekorApplyConfiguration constructs a declarative configuration of the FulcioCAWithRekor type for use with
// apply.
func FulcioCAWithRekor() *FulcioCAWithRekorApplyConfiguration {
	return &FulcioCAWithRekorApplyConfiguration{}
}

// WithFulcioCAData adds the given value to the FulcioCAData field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the FulcioCAData field.
func (b *FulcioCAWithRekorApplyConfiguration) WithFulcioCAData(values ...byte) *FulcioCAWithRekorApplyConfiguration {
	for i := range values {
		b.FulcioCAData = append(b.FulcioCAData, values[i])
	}
	return b
}

// WithRekorKeyData adds the given value to the RekorKeyData field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the RekorKeyData field.
func (b *FulcioCAWithRekorApplyConfiguration) WithRekorKeyData(values ...byte) *FulcioCAWithRekorApplyConfiguration {
	for i := range values {
		b.RekorKeyData = append(b.RekorKeyData, values[i])
	}
	return b
}

// WithFulcioSubject sets the FulcioSubject field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FulcioSubject field is set to the value of the last call.
func (b *FulcioCAWithRekorApplyConfiguration) WithFulcioSubject(value *PolicyFulcioSubjectApplyConfiguration) *FulcioCAWithRekorApplyConfiguration {
	b.FulcioSubject = value
	return b
}
