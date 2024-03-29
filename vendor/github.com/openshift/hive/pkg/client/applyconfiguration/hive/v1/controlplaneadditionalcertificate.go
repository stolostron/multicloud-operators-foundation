// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ControlPlaneAdditionalCertificateApplyConfiguration represents an declarative configuration of the ControlPlaneAdditionalCertificate type for use
// with apply.
type ControlPlaneAdditionalCertificateApplyConfiguration struct {
	Name   *string `json:"name,omitempty"`
	Domain *string `json:"domain,omitempty"`
}

// ControlPlaneAdditionalCertificateApplyConfiguration constructs an declarative configuration of the ControlPlaneAdditionalCertificate type for use with
// apply.
func ControlPlaneAdditionalCertificate() *ControlPlaneAdditionalCertificateApplyConfiguration {
	return &ControlPlaneAdditionalCertificateApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *ControlPlaneAdditionalCertificateApplyConfiguration) WithName(value string) *ControlPlaneAdditionalCertificateApplyConfiguration {
	b.Name = &value
	return b
}

// WithDomain sets the Domain field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Domain field is set to the value of the last call.
func (b *ControlPlaneAdditionalCertificateApplyConfiguration) WithDomain(value string) *ControlPlaneAdditionalCertificateApplyConfiguration {
	b.Domain = &value
	return b
}
