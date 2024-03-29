// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckpointSpecApplyConfiguration represents an declarative configuration of the CheckpointSpec type for use
// with apply.
type CheckpointSpecApplyConfiguration struct {
	LastBackupChecksum *string                            `json:"lastBackupChecksum,omitempty"`
	LastBackupTime     *v1.Time                           `json:"lastBackupTime,omitempty"`
	LastBackupRef      *BackupReferenceApplyConfiguration `json:"lastBackupRef,omitempty"`
}

// CheckpointSpecApplyConfiguration constructs an declarative configuration of the CheckpointSpec type for use with
// apply.
func CheckpointSpec() *CheckpointSpecApplyConfiguration {
	return &CheckpointSpecApplyConfiguration{}
}

// WithLastBackupChecksum sets the LastBackupChecksum field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastBackupChecksum field is set to the value of the last call.
func (b *CheckpointSpecApplyConfiguration) WithLastBackupChecksum(value string) *CheckpointSpecApplyConfiguration {
	b.LastBackupChecksum = &value
	return b
}

// WithLastBackupTime sets the LastBackupTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastBackupTime field is set to the value of the last call.
func (b *CheckpointSpecApplyConfiguration) WithLastBackupTime(value v1.Time) *CheckpointSpecApplyConfiguration {
	b.LastBackupTime = &value
	return b
}

// WithLastBackupRef sets the LastBackupRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastBackupRef field is set to the value of the last call.
func (b *CheckpointSpecApplyConfiguration) WithLastBackupRef(value *BackupReferenceApplyConfiguration) *CheckpointSpecApplyConfiguration {
	b.LastBackupRef = value
	return b
}
