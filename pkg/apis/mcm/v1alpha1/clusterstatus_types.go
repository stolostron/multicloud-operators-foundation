// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ResourceNodes is the name of nodes resources
	ResourceNodes = "nodes"
	// ResourcePods is the name of pods resources
	ResourcePods = "pods"
	// ResourcePVS is the name of pv resources
	ResourcePVS = "pvs"
	// ResourceRoles is the name of roles resources
	ResourceRoles = "roles"
	// ResourceClusterRoles is the name of clusterroles resources
	ResourceClusterRoles = "clusterroles"
	// ResourceReleases is the name of release resources
	ResourceReleases = "releases"
	// ResourceWeaveTopology is the name of weave topology data
	ResourceWeaveTopology = "weavetopology"
)

const (
	// UserIdentityAnnotation is identity annotation
	UserIdentityAnnotation = "mcm.ibm.com/user-identity"

	// UserGroupAnnotation is user group annotation
	UserGroupAnnotation = "mcm.ibm.com/user-group"
)

// ClusterStatusSpec is information about the current status of a cluster updated by cluster controller periodically.
type ClusterStatusSpec struct {
	// MasterIP shows the master IP of managed cluster
	MasterAddresses []v1.EndpointAddress `json:"masterAddresses,omitempty"`
	// ConcoleURL shows the url of icp console in managed cluster
	ConsoleURL string `json:"consoleURL,omitempty"`
	// Capacity
	Capacity v1.ResourceList `json:"capacity,omitempty"`
	// Usage
	Usage v1.ResourceList `json:"usage,omitempty"`
	// KlusterletEndpoint shows the endpoint to connect to klusterlet of managed cluster
	KlusterletEndpoint v1.EndpointAddress `json:"klusterletEndpoint,omitempty"`
	// KlusterletPort shows the port to connect to klusterlet of managed cluster
	KlusterletPort v1.EndpointPort `json:"klusterletPort,omitempty"`
	// MonitoringScrapeTarget is the scrape target to be used
	MonitoringScrapeTarget string `json:"monitoringScrapeTarget,omitempty"`
	// Version of Klusterlet
	KlusterletVersion string `json:"klusterletVersion,omitempty"`
	// Version of member cluster
	Version string `json:"version,omitempty"`
	// KlusterletCA is the ca data for klusterlet to authorize apiserver
	KlusterletCA []byte `json:"klusterletCA,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterStatus are namespaced and have unique names in the hcm.
type ClusterStatus struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the Cluster.
	// +optional
	Spec ClusterStatusSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterStatusList is a list of all the kubernetes clusters status in the hcm
type ClusterStatusList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Cluster objects.
	Items []ClusterStatus `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterRestOptions is the option for pod
type ClusterRestOptions struct {
	metav1.TypeMeta `json:",inline"`

	// Path is the URL path to use for the current proxy request to pod.
	// +optional
	Path string `json:"path,omitempty"`
}
