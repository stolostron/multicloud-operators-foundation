// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	configv1alpha1 "github.com/openshift/api/config/v1alpha1"
)

// StorageApplyConfiguration represents a declarative configuration of the Storage type for use
// with apply.
type StorageApplyConfiguration struct {
	Type             *configv1alpha1.StorageType               `json:"type,omitempty"`
	PersistentVolume *PersistentVolumeConfigApplyConfiguration `json:"persistentVolume,omitempty"`
}

// StorageApplyConfiguration constructs a declarative configuration of the Storage type for use with
// apply.
func Storage() *StorageApplyConfiguration {
	return &StorageApplyConfiguration{}
}

// WithType sets the Type field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Type field is set to the value of the last call.
func (b *StorageApplyConfiguration) WithType(value configv1alpha1.StorageType) *StorageApplyConfiguration {
	b.Type = &value
	return b
}

// WithPersistentVolume sets the PersistentVolume field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PersistentVolume field is set to the value of the last call.
func (b *StorageApplyConfiguration) WithPersistentVolume(value *PersistentVolumeConfigApplyConfiguration) *StorageApplyConfiguration {
	b.PersistentVolume = value
	return b
}
