// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// VeleroBackupConfigApplyConfiguration represents an declarative configuration of the VeleroBackupConfig type for use
// with apply.
type VeleroBackupConfigApplyConfiguration struct {
	Enabled   *bool   `json:"enabled,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}

// VeleroBackupConfigApplyConfiguration constructs an declarative configuration of the VeleroBackupConfig type for use with
// apply.
func VeleroBackupConfig() *VeleroBackupConfigApplyConfiguration {
	return &VeleroBackupConfigApplyConfiguration{}
}

// WithEnabled sets the Enabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Enabled field is set to the value of the last call.
func (b *VeleroBackupConfigApplyConfiguration) WithEnabled(value bool) *VeleroBackupConfigApplyConfiguration {
	b.Enabled = &value
	return b
}

// WithNamespace sets the Namespace field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Namespace field is set to the value of the last call.
func (b *VeleroBackupConfigApplyConfiguration) WithNamespace(value string) *VeleroBackupConfigApplyConfiguration {
	b.Namespace = &value
	return b
}