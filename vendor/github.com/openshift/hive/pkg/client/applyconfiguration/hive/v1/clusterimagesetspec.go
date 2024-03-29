// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ClusterImageSetSpecApplyConfiguration represents an declarative configuration of the ClusterImageSetSpec type for use
// with apply.
type ClusterImageSetSpecApplyConfiguration struct {
	ReleaseImage *string `json:"releaseImage,omitempty"`
}

// ClusterImageSetSpecApplyConfiguration constructs an declarative configuration of the ClusterImageSetSpec type for use with
// apply.
func ClusterImageSetSpec() *ClusterImageSetSpecApplyConfiguration {
	return &ClusterImageSetSpecApplyConfiguration{}
}

// WithReleaseImage sets the ReleaseImage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ReleaseImage field is set to the value of the last call.
func (b *ClusterImageSetSpecApplyConfiguration) WithReleaseImage(value string) *ClusterImageSetSpecApplyConfiguration {
	b.ReleaseImage = &value
	return b
}
