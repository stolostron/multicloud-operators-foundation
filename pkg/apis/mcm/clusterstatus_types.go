// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterLabel is the label set to point to the cluster
const ClusterLabel = "mcm.ibm.com/cluster"

// ClusterStatusSpec is information about the current status of a cluster updated by cluster controller periodically.
type ClusterStatusSpec struct {
	// MasterIP shows the master IP of managed cluster
	MasterAddresses []v1.EndpointAddress `json:"masterAddresses,omitempty" protobuf:"bytes,1,rep,name=masterAddresses"`
	// ConcoleURL shows the url of icp console in managed cluster
	ConsoleURL string `json:"consoleURL,omitempty" protobuf:"bytes,2,opt,name=consoleURL"`
	// Capacity
	Capacity v1.ResourceList `json:"capacity,omitempty" protobuf:"bytes,3,opt,name=capacity"`
	// Usage
	Usage v1.ResourceList `json:"usage,omitempty" protobuf:"bytes,4,opt,name=usage"`
	// KlusterletEndpoint shows the endpoint to connect to klusterlet of managed cluster
	KlusterletEndpoint v1.EndpointAddress `json:"klusterletEndpoint,omitempty" protobuf:"bytes,5,opt,name=klusterletEndpoint"`
	// KlusterletPort shows the port to connect to klusterlet of managed cluster
	KlusterletPort v1.EndpointPort `json:"klusterletPort,omitempty" protobuf:"bytes,6,opt,name=klusterletPort"`
	// MonitoringScrapeTarget is the scrape target to be used
	MonitoringScrapeTarget string `json:"monitoringScrapeTarget,omitempty" protobuf:"bytes,7,opt,name=monitoringScrapeTarget"`
	// Version of Klusterlet
	KlusterletVersion string `json:"klusterletVersion,omitempty" protobuf:"bytes,8,opt,name=klusterletVersion"`
	// Version is the kubernetes version of the member cluster
	Version string `json:"version,omitempty" protobuf:"bytes,9,opt,name=version"`
	// KlusterletCA is the ca data for klusterlet to authorize apiserver
	KlusterletCA []byte `json:"klusterletCA,omitempty" protobuf:"bytes,10,rep,name=klusterletCA"`
	// Version of Endpoint
	EndpointVersion string `json:"endpointVersion,omitempty" protobuf:"bytes,11,opt,name=endpointVersion"`
	// Version of Endpoint Operator
	EndpointOperatorVersion string `json:"endpointOperatorVersion,omitempty" protobuf:"bytes,12,opt,name=endpointOperatorVersion"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterStatus are namespaced and have unique names in the hcm.
type ClusterStatus struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the Cluster.
	// +optional
	Spec ClusterStatusSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterStatusList is a list of all the kubernetes clusters status in the hcm
type ClusterStatusList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Cluster objects.
	Items []ClusterStatus `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterRestOptions is the option for pod
type ClusterRestOptions struct {
	metav1.TypeMeta `json:",inline"`

	// Path is the URL path to use for the current proxy request to pod.
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
}
