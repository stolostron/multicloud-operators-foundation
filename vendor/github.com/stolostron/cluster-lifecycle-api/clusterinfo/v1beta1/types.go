package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

// ClusterInfoSpec is information about the current status of a managed cluster updated
// by ManagedClusterInfo controller periodically.
type ClusterInfoSpec struct {
	// LoggingCA is the ca data for logging server to authorize apiserver
	// +optional
	LoggingCA []byte `json:"loggingCA,omitempty"`

	// MasterEndpoint shows the apiserver endpoint of managed cluster
	// +optional
	MasterEndpoint string `json:"masterEndpoint,omitempty"`
}

type ClientConfig struct {
	// URL is the URL of apiserver endpoint of the managed cluster.
	// +required
	URL string `json:"url"`

	// CABundle is the ca bundle to connect to apiserver of the managed cluster.
	// System certs are used if it is not set.
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`
}

// ClusterInfoStatus is the information about managed cluster
type ClusterInfoStatus struct {
	// Conditions contains condition information for a managed cluster
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Version is the kube version of managed cluster.
	// +optional
	Version string `json:"version,omitempty"`

	// KubeVendor describes the kubernetes provider of the managed cluster.
	// Deprecated in release 2.3 and will be removed in the future. Use clusterClaim platform.open-cluster-management.io instead.
	// +optional
	KubeVendor KubeVendorType `json:"kubeVendor,omitempty"`

	// CloudVendor describes the cloud provider for the managed cluster.
	// Deprecated in release 2.3 and will be removed in the future. Use clusterClaim product.open-cluster-management.io instead.
	// +optional
	CloudVendor CloudVendorType `json:"cloudVendor,omitempty"`

	// ClusterID is the identifier of managed cluster.
	// Deprecated in release 2.3 and will be removed in the future. Use clusterClaim id.openshift.io instead.
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// DistributionInfo is the information about distribution of managed cluster
	// +optional
	DistributionInfo DistributionInfo `json:"distributionInfo,omitempty"`

	// ConsoleURL shows the url of console in managed cluster.
	// Deprecated in release 2.3 and will be removed in the future. Use clusterClaim consoleurl.cluster.open-cluster-management.io instead.
	// +optional
	ConsoleURL string `json:"consoleURL,omitempty"`

	// NodeList shows a list of the status of nodes
	// +optional
	NodeList []NodeStatus `json:"nodeList,omitempty"`

	// LoggingEndpoint shows the endpoint to connect to logging server of managed cluster
	// +optional
	LoggingEndpoint corev1.EndpointAddress `json:"loggingEndpoint,omitempty"`

	// LoggingPort shows the port to connect to logging server of managed cluster
	// +optional
	LoggingPort corev1.EndpointPort `json:"loggingPort,omitempty"`
}

// NodeStatus presents the name, labels and conditions of node
type NodeStatus struct {
	// Name of node
	// +optional
	Name string `json:"name,omitempty"`

	// Labels of node.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Capacity represents the total resources of a node. only includes CPU and memory.
	// +optional
	Capacity ResourceList `json:"capacity,omitempty"`

	// Conditions is an array of current node conditions. only includes NodeReady.
	// +optional
	Conditions []NodeCondition `json:"conditions,omitempty"`
}

const (
	// CPU, in cores. (500m = .5 cores)
	ResourceCPU clusterv1.ResourceName = "cpu"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceMemory clusterv1.ResourceName = "memory"
)

// ResourceList defines a map for the quantity of different resources, the definition
// matches the ResourceList defined in k8s.io/api/core/v1
type ResourceList map[clusterv1.ResourceName]resource.Quantity

type NodeCondition struct {
	// Type of node condition.
	Type corev1.NodeConditionType `json:"type,omitempty"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty"`
}

// DistributionInfo defines the information about distribution of managed cluster
// +union
type DistributionInfo struct {
	// Type is the distribution type of managed cluster, is OCP currently
	// +unionDiscriminator
	Type DistributionType `json:"type,omitempty"`

	// OCP is the distribution information of OCP managed cluster, is matched when the Type is OCP.
	OCP OCPDistributionInfo `json:"ocp,omitempty"`
}

// OCPVersionRelease represents an OpenShift release image and associated metadata.
// The original definition is from https://github.com/openshift/api/blob/master/config/v1/types_cluster_version.go
type OCPVersionRelease struct {
	// version is a semantic versioning identifying the update version. When this
	// field is part of spec, version is optional if image is specified.
	Version string `json:"version"`

	// image is a container image location that contains the update. When this
	// field is part of spec, image is optional if version is specified and the
	// availableUpdates field contains a matching version.
	Image string `json:"image"`

	// url contains information about this release. This URL is set by
	// the 'url' metadata property on a release or the metadata returned by
	// the update API and should be displayed as a link in user
	// interfaces. The URL field may not be set for test or nightly
	// releases.
	URL string `json:"url,omitempty"`

	// channels is the set of Cincinnati channels to which the release
	// currently belongs.
	Channels []string `json:"channels,omitempty"`
}

// OCPVersionUpdateHistory is a single attempted update to the cluster.
// the original definition is from https://github.com/openshift/api/blob/master/config/v1/types_cluster_version.go
type OCPVersionUpdateHistory struct {
	// state reflects whether the update was fully applied. The Partial state
	// indicates the update is not fully applied, while the Completed state
	// indicates the update was successfully rolled out at least once (all
	// parts of the update successfully applied).
	State string `json:"state"`

	// version is a semantic versioning identifying the update version. If the
	// requested image does not define a version, or if a failure occurs
	// retrieving the image, this value may be empty.
	Version string `json:"version"`

	// image is a container image location that contains the update. This value
	// is always populated.
	Image string `json:"image"`

	// verified indicates whether the provided update was properly verified
	// before it was installed. If this is false the cluster may not be trusted.
	Verified bool `json:"verified"`
}

// OCPDistributionInfo defines the distribution information of OCP managed cluster
type OCPDistributionInfo struct {
	// Version is the current version of the OCP cluster.
	// Deprecated in release 2.3 and will be removed in the future. Use clusterClaim version.openshift.io instead.
	Version string `json:"version,omitempty"`

	// AvailableUpdates contains the list of update versions that are appropriate for the manage cluster.
	// Deprecated in release 2.3 and will be removed in the future. Use VersionAvailableUpdates instead.
	AvailableUpdates []string `json:"availableUpdates,omitempty"`

	// DesiredVersion is the version that the cluster is reconciling towards.
	// Deprecated in release 2.3 and will be removed in the future. User Desired instead.
	DesiredVersion string `json:"desiredVersion,omitempty"`

	// UpgradeFailed indicates whether upgrade of the manage cluster is failed.
	// This is true if the status of Failing condition is True and the version is different with desiredVersion in clusterVersion
	UpgradeFailed bool `json:"upgradeFailed,omitempty"`

	// Channel is an identifier for explicitly requesting that a non-default
	// set of updates be applied to this cluster. The default channel will be
	// contain stable updates that are appropriate for production clusters.
	Channel string `json:"channel,omitempty"`

	// desired is the version that the cluster is reconciling towards.
	// If the cluster is not yet fully initialized desired will be set
	// with the information available, which may be an image or a tag.
	Desired OCPVersionRelease `json:"desired,omitempty"`

	// VersionAvailableUpdates contains the list of updates that are appropriate
	// for this cluster. This list may be empty if no updates are recommended,
	// if the update service is unavailable, or if an invalid channel has
	// been specified.
	VersionAvailableUpdates []OCPVersionRelease `json:"versionAvailableUpdates,omitempty"`

	// VersionHistory contains a list of the most recent versions applied to the cluster.
	// This value may be empty during cluster startup, and then will be updated
	// when a new update is being applied. The newest update is first in the
	// list and it is ordered by recency. Updates in the history have state
	// Completed if the rollout completed - if an update was failing or halfway
	// applied the state will be Partial. Only a limited amount of update history
	// is preserved.
	VersionHistory []OCPVersionUpdateHistory `json:"versionHistory,omitempty"`

	// Controller will sync this field to managedcluster's ManagedClusterClientConfigs
	// +optional
	ManagedClusterClientConfig ClientConfig `json:"managedClusterClientConfig,omitempty"`
}

// DistributionType is type of distribution
type DistributionType string

// Supported distribution type
const (
	DistributionTypeOCP     DistributionType = "OCP"
	DistributionTypeUnknown DistributionType = "Unknown"
)

// KubeVendorType describe the kubernetes provider of the cluster
type KubeVendorType string

const (
	// KubeVendorOpenShift OpenShift
	KubeVendorOpenShift KubeVendorType = "OpenShift"
	// KubeVendorAKS Azure Kubernetes Service
	KubeVendorAKS KubeVendorType = "AKS"
	// KubeVendorEKS Elastic Kubernetes Service
	KubeVendorEKS KubeVendorType = "EKS"
	// KubeVendorGKE Google Kubernetes Engine
	KubeVendorGKE KubeVendorType = "GKE"
	// KubeVendorICP IBM Cloud Private
	KubeVendorICP KubeVendorType = "ICP"
	// KubeVendorIKS IBM Kubernetes Service
	KubeVendorIKS KubeVendorType = "IKS"
	// KubeVendorOSD OpenShiftDedicated
	KubeVendorOSD KubeVendorType = "OpenShiftDedicated"
	// KubeVendorOther other (unable to auto detect)
	KubeVendorOther KubeVendorType = "Other"
)

// CloudVendorType describe the cloud provider for the cluster
type CloudVendorType string

const (
	// CloudVendorIBM IBM
	CloudVendorIBM CloudVendorType = "IBM"
	// CloudVendorIBMZ IBM s360x
	CloudVendorIBMZ CloudVendorType = "IBMZPlatform"
	// CloudVendorIBMP IBM Power
	CloudVendorIBMP CloudVendorType = "IBMPowerPlatform"
	// CloudVendorAWS Amazon
	CloudVendorAWS CloudVendorType = "Amazon"
	// CloudVendorAzure Azure
	CloudVendorAzure CloudVendorType = "Azure"
	// CloudVendorGoogle Google
	CloudVendorGoogle CloudVendorType = "Google"
	// CloudVendorVSphere vSphere
	CloudVendorVSphere CloudVendorType = "VSphere"
	// CloudVendorOpenStack OpenStack
	CloudVendorOpenStack CloudVendorType = "OpenStack"
	// CloudVendorRHV RHV
	CloudVendorRHV CloudVendorType = "RHV"
	// CloudVendorAlibabaCloud AlibabaCloud
	CloudVendorAlibabaCloud = "AlibabaCloud"
	// CloudVendorBareMetal BareMetal
	CloudVendorBareMetal = "BareMetal"
	// CloudVendorOther other (unable to auto detect)
	CloudVendorOther CloudVendorType = "Other"
)

const (
	// ManagedClusterInfoSynced means the info on managed cluster is synced.
	ManagedClusterInfoSynced string = "ManagedClusterInfoSynced"
)

const (
	ReasonManagedClusterInfoSynced       string = "ManagedClusterInfoSynced"
	ReasonManagedClusterInfoSyncedFailed string = "ReasonManagedClusterInfoSyncedFailed"
)

const (
	LabelCloudVendor = "cloud"
	LabelKubeVendor  = "vendor"
	LabelClusterID   = "clusterID"
	LabelManagedBy   = "managed-by"
	AutoDetect       = "auto-detect"
	// OCPVersion is the full version of OCP cluster, like 4.11.3
	OCPVersion = "openshiftVersion"
	// OCPVersionMajor is the major version of OCP cluster, like 4
	OCPVersionMajor = "openshiftVersion-major"
	// OCPVersionMajorMinor is the version of OCP cluster without patch, like 4.11
	OCPVersionMajorMinor = "openshiftVersion-major-minor"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=managedclusterinfos

// ManagedClusterInfo represents the information of managed cluster that acm hub needs to know
type ManagedClusterInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the information of the Cluster.
	// +optional
	Spec ClusterInfoSpec `json:"spec,omitempty"`

	// Status represents the desired status of the Cluster
	// +optional
	Status ClusterInfoStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ManagedClusterInfoList is a list of ManagedClusterInfo objects
type ManagedClusterInfoList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ManagedClusterInfo objects.
	Items []ManagedClusterInfo `json:"items"`
}
