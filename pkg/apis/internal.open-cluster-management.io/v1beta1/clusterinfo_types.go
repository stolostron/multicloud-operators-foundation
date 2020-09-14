package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ClusterInfoStatus is the information about managed cluster
type ClusterInfoStatus struct {
	// Conditions contains condition information for a managed cluster
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Version is the kube version of managed cluster.
	// +optional
	Version string `json:"version,omitempty"`

	// KubeVendor describes the kubernetes provider of the managed cluster.
	// +optional
	KubeVendor KubeVendorType `json:"kubeVendor,omitempty"`

	// CloudVendor describes the cloud provider for the managed cluster.
	// +optional
	CloudVendor CloudVendorType `json:"cloudVendor,omitempty"`

	// ClusterID is the identifier of managed cluster
	// +optional
	ClusterID string `json:"clusterID,omitempty"`

	// DistributionInfo is the information about distribution of managed cluster
	// +optional
	DistributionInfo DistributionInfo `json:"distributionInfo,omitempty"`

	// ConsoleURL shows the url of console in managed cluster
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

// ResourceName is the name identifying various resources in a ResourceList.
type ResourceName string

const (
	// CPU, in cores. (500m = .5 cores)
	ResourceCPU ResourceName = "cpu"
	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	ResourceMemory ResourceName = "memory"
)

// ResourceList defines a map for the quantity of different resources, the definition
// matches the ResourceList defined in k8s.io/api/core/v1
type ResourceList map[ResourceName]resource.Quantity

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

// OCPDistributionInfo defines the distribution information of OCP managed cluster
type OCPDistributionInfo struct {
	// Version is the distribution version of OCP
	Version string `json:"version,omitempty"`
	// availableUpdates contains the list of update versions that are appropriate for the manage cluster.
	AvailableUpdates []string `json:"availableUpdates,omitempty"`
	// Desiredversion represents the desired version of the ocp cluster
	DesiredVersion string `json:"desiredVersion,omitempty"`
	// UpgradeFailed indicate whether upgrade of the manage cluster is failed
	UpgradeFailed bool `json:"upgradeFailed,omitempty"`
}

// DistributionType is type of distribution
type DistributionType string

// Supported distribution type
const (
	DistributionTypeOCP     DistributionType = "OCP"
	DistributionTypeUnknown DistributionType = "Unknow"
)

// KubeVendorType describe the kubernetes provider of the cluster
type KubeVendorType string

const (
	// KubeVendorOpenShift OpenShift
	KubeVendorOpenShift KubeVendorType = "OpenShift"
	// KubeVendorAKS Azure Kuberentes Service
	KubeVendorAKS KubeVendorType = "AKS"
	// KubeVendorEKS Elastic Kubernetes Service
	KubeVendorEKS KubeVendorType = "EKS"
	// KubeVendorGKE Google Kubernetes Engine
	KubeVendorGKE KubeVendorType = "GKE"
	// KubeVendorICP IBM Cloud Private
	KubeVendorICP KubeVendorType = "ICP"
	// KubeVendorIKS IBM Kubernetes Service
	KubeVendorIKS KubeVendorType = "IKS"
	// KubeVendorOther other (unable to auto detect)
	KubeVendorOther KubeVendorType = "Other"
)

// CloudVendorType describe the cloud provider for the cluster
type CloudVendorType string

const (
	// CloudVendorIBM IBM
	CloudVendorIBM CloudVendorType = "IBM"
	// CloudVendorAWS Amazon
	CloudVendorAWS CloudVendorType = "Amazon"
	// CloudVendorAzure Azure
	CloudVendorAzure CloudVendorType = "Azure"
	// CloudVendorGoogle Google
	CloudVendorGoogle CloudVendorType = "Google"
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

// +kubebuilder:object:root=true
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

// +kubebuilder:object:root=true

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

func init() {
	SchemeBuilder.Register(&ManagedClusterInfo{}, &ManagedClusterInfoList{})
}
