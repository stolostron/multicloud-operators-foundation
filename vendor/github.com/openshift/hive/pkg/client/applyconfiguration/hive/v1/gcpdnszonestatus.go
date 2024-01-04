// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// GCPDNSZoneStatusApplyConfiguration represents an declarative configuration of the GCPDNSZoneStatus type for use
// with apply.
type GCPDNSZoneStatusApplyConfiguration struct {
	ZoneName *string `json:"zoneName,omitempty"`
}

// GCPDNSZoneStatusApplyConfiguration constructs an declarative configuration of the GCPDNSZoneStatus type for use with
// apply.
func GCPDNSZoneStatus() *GCPDNSZoneStatusApplyConfiguration {
	return &GCPDNSZoneStatusApplyConfiguration{}
}

// WithZoneName sets the ZoneName field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ZoneName field is set to the value of the last call.
func (b *GCPDNSZoneStatusApplyConfiguration) WithZoneName(value string) *GCPDNSZoneStatusApplyConfiguration {
	b.ZoneName = &value
	return b
}