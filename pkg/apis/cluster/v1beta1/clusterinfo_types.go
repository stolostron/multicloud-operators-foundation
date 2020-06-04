// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta1

import (
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterInfoSpec is information about the current status of a spoke cluster updated by clusterinfo controller periodically.
type ClusterInfoSpec struct {
	// KlusterletCA is the ca data for klusterlet to authorize apiserver
	// +optional
	KlusterletCA []byte `json:"klusterletCA,omitempty"`

	// MasterEndpoint shows the apiserver endpoint of spoke cluster
	// +optional
	MasterEndpoint string `json:"masterEndpoint,omitempty"`
}

// ClusterInfoStatus is the information about spoke cluster
type ClusterInfoStatus struct {
	// Conditions contains condition information for a spoke cluster
	// +optional
	Conditions []clusterv1.StatusCondition `json:"conditions,omitempty"`

	// Version is the kube version of spoke cluster.
	// +optional
	Version string `json:"version,omitempty"`

	// DistributionInfo is the information about distribution of spoke cluster
	// +optional
	DistributionInfo DistributionInfo `json:"distributionInfo,omitempty"`

	// ConsoleURL shows the url of console in spoke cluster
	// +optional
	ConsoleURL string `json:"consoleURL,omitempty"`

	// NodeList shows a list of the status of nodes
	// +optional
	NodeList []NodeStatus `json:"nodeList,omitempty"`

	// KlusterletEndpoint shows the endpoint to connect to klusterlet of spoke cluster
	// +optional
	KlusterletEndpoint corev1.EndpointAddress `json:"klusterletEndpoint,omitempty"`

	// KlusterletPort shows the port to connect to klusterlet of spoke cluster
	// +optional
	KlusterletPort corev1.EndpointPort `json:"klusterletPort,omitempty"`
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

// DistributionInfo defines the information about distribution of spoke cluster
// +union
type DistributionInfo struct {
	// Type is the distribution type of spoke cluster, is OCP currently
	// +unionDiscriminator
	Type DistributionType `json:"type,omitempty"`

	// OCP is the distribution information of OCP spoke cluster, is matched when the Type is OCP.
	OCP OCPDistributionInfo `json:"ocp,omitempty"`
}

// OCPDistributionInfo defines the distribution information of OCP spoke cluster
type OCPDistributionInfo struct {
	// Version is the distribution version of OCP
	Version string `json:"version,omitempty"`
}

// DistributionType is type of distribution
type DistributionType string

// Supported distribution type
const (
	DistributionTypeOCP DistributionType = "OCP"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterinfos

// ClusterInfo represents the information of spoke cluster that acm hub needs to know
type ClusterInfo struct {
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

// ClusterInfoList is a list of ClusterInfo objects
type ClusterInfoList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ClusterInfo objects.
	Items []ClusterInfo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterInfo{}, &ClusterInfoList{})
}
