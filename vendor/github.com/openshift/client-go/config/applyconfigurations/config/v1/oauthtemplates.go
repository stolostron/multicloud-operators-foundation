// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// OAuthTemplatesApplyConfiguration represents a declarative configuration of the OAuthTemplates type for use
// with apply.
type OAuthTemplatesApplyConfiguration struct {
	Login             *SecretNameReferenceApplyConfiguration `json:"login,omitempty"`
	ProviderSelection *SecretNameReferenceApplyConfiguration `json:"providerSelection,omitempty"`
	Error             *SecretNameReferenceApplyConfiguration `json:"error,omitempty"`
}

// OAuthTemplatesApplyConfiguration constructs a declarative configuration of the OAuthTemplates type for use with
// apply.
func OAuthTemplates() *OAuthTemplatesApplyConfiguration {
	return &OAuthTemplatesApplyConfiguration{}
}

// WithLogin sets the Login field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Login field is set to the value of the last call.
func (b *OAuthTemplatesApplyConfiguration) WithLogin(value *SecretNameReferenceApplyConfiguration) *OAuthTemplatesApplyConfiguration {
	b.Login = value
	return b
}

// WithProviderSelection sets the ProviderSelection field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ProviderSelection field is set to the value of the last call.
func (b *OAuthTemplatesApplyConfiguration) WithProviderSelection(value *SecretNameReferenceApplyConfiguration) *OAuthTemplatesApplyConfiguration {
	b.ProviderSelection = value
	return b
}

// WithError sets the Error field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Error field is set to the value of the last call.
func (b *OAuthTemplatesApplyConfiguration) WithError(value *SecretNameReferenceApplyConfiguration) *OAuthTemplatesApplyConfiguration {
	b.Error = value
	return b
}
