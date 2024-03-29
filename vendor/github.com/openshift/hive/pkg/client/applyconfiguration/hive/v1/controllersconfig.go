// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ControllersConfigApplyConfiguration represents an declarative configuration of the ControllersConfig type for use
// with apply.
type ControllersConfigApplyConfiguration struct {
	Default     *ControllerConfigApplyConfiguration          `json:"default,omitempty"`
	Controllers []SpecificControllerConfigApplyConfiguration `json:"controllers,omitempty"`
}

// ControllersConfigApplyConfiguration constructs an declarative configuration of the ControllersConfig type for use with
// apply.
func ControllersConfig() *ControllersConfigApplyConfiguration {
	return &ControllersConfigApplyConfiguration{}
}

// WithDefault sets the Default field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Default field is set to the value of the last call.
func (b *ControllersConfigApplyConfiguration) WithDefault(value *ControllerConfigApplyConfiguration) *ControllersConfigApplyConfiguration {
	b.Default = value
	return b
}

// WithControllers adds the given value to the Controllers field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Controllers field.
func (b *ControllersConfigApplyConfiguration) WithControllers(values ...*SpecificControllerConfigApplyConfiguration) *ControllersConfigApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithControllers")
		}
		b.Controllers = append(b.Controllers, *values[i])
	}
	return b
}
