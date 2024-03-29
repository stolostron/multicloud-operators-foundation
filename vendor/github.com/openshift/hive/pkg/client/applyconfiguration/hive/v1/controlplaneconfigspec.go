// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ControlPlaneConfigSpecApplyConfiguration represents an declarative configuration of the ControlPlaneConfigSpec type for use
// with apply.
type ControlPlaneConfigSpecApplyConfiguration struct {
	ServingCertificates *ControlPlaneServingCertificateSpecApplyConfiguration `json:"servingCertificates,omitempty"`
	APIURLOverride      *string                                               `json:"apiURLOverride,omitempty"`
	APIServerIPOverride *string                                               `json:"apiServerIPOverride,omitempty"`
}

// ControlPlaneConfigSpecApplyConfiguration constructs an declarative configuration of the ControlPlaneConfigSpec type for use with
// apply.
func ControlPlaneConfigSpec() *ControlPlaneConfigSpecApplyConfiguration {
	return &ControlPlaneConfigSpecApplyConfiguration{}
}

// WithServingCertificates sets the ServingCertificates field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ServingCertificates field is set to the value of the last call.
func (b *ControlPlaneConfigSpecApplyConfiguration) WithServingCertificates(value *ControlPlaneServingCertificateSpecApplyConfiguration) *ControlPlaneConfigSpecApplyConfiguration {
	b.ServingCertificates = value
	return b
}

// WithAPIURLOverride sets the APIURLOverride field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the APIURLOverride field is set to the value of the last call.
func (b *ControlPlaneConfigSpecApplyConfiguration) WithAPIURLOverride(value string) *ControlPlaneConfigSpecApplyConfiguration {
	b.APIURLOverride = &value
	return b
}

// WithAPIServerIPOverride sets the APIServerIPOverride field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the APIServerIPOverride field is set to the value of the last call.
func (b *ControlPlaneConfigSpecApplyConfiguration) WithAPIServerIPOverride(value string) *ControlPlaneConfigSpecApplyConfiguration {
	b.APIServerIPOverride = &value
	return b
}
