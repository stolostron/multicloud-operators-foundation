// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// ClusterDeprovisionPlatformApplyConfiguration represents an declarative configuration of the ClusterDeprovisionPlatform type for use
// with apply.
type ClusterDeprovisionPlatformApplyConfiguration struct {
	AlibabaCloud *AlibabaCloudClusterDeprovisionApplyConfiguration `json:"alibabacloud,omitempty"`
	AWS          *AWSClusterDeprovisionApplyConfiguration          `json:"aws,omitempty"`
	Azure        *AzureClusterDeprovisionApplyConfiguration        `json:"azure,omitempty"`
	GCP          *GCPClusterDeprovisionApplyConfiguration          `json:"gcp,omitempty"`
	OpenStack    *OpenStackClusterDeprovisionApplyConfiguration    `json:"openstack,omitempty"`
	VSphere      *VSphereClusterDeprovisionApplyConfiguration      `json:"vsphere,omitempty"`
	Ovirt        *OvirtClusterDeprovisionApplyConfiguration        `json:"ovirt,omitempty"`
	IBMCloud     *IBMClusterDeprovisionApplyConfiguration          `json:"ibmcloud,omitempty"`
}

// ClusterDeprovisionPlatformApplyConfiguration constructs an declarative configuration of the ClusterDeprovisionPlatform type for use with
// apply.
func ClusterDeprovisionPlatform() *ClusterDeprovisionPlatformApplyConfiguration {
	return &ClusterDeprovisionPlatformApplyConfiguration{}
}

// WithAlibabaCloud sets the AlibabaCloud field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AlibabaCloud field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithAlibabaCloud(value *AlibabaCloudClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.AlibabaCloud = value
	return b
}

// WithAWS sets the AWS field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AWS field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithAWS(value *AWSClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.AWS = value
	return b
}

// WithAzure sets the Azure field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Azure field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithAzure(value *AzureClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.Azure = value
	return b
}

// WithGCP sets the GCP field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the GCP field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithGCP(value *GCPClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.GCP = value
	return b
}

// WithOpenStack sets the OpenStack field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the OpenStack field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithOpenStack(value *OpenStackClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.OpenStack = value
	return b
}

// WithVSphere sets the VSphere field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the VSphere field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithVSphere(value *VSphereClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.VSphere = value
	return b
}

// WithOvirt sets the Ovirt field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ovirt field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithOvirt(value *OvirtClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.Ovirt = value
	return b
}

// WithIBMCloud sets the IBMCloud field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IBMCloud field is set to the value of the last call.
func (b *ClusterDeprovisionPlatformApplyConfiguration) WithIBMCloud(value *IBMClusterDeprovisionApplyConfiguration) *ClusterDeprovisionPlatformApplyConfiguration {
	b.IBMCloud = value
	return b
}
