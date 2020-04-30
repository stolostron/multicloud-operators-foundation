// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterInfoSpec is information about the current status of a cluster updated by cluster controller periodically.
type ClusterInfoSpec struct {
	// MasterIP shows the master IP of managed cluster
	MasterAddresses []v1.EndpointAddress `json:"masterAddresses,omitempty"`
	// ConcoleURL shows the url of icp console in managed cluster
	ConsoleURL string `json:"consoleURL,omitempty"`
	// KlusterletEndpoint shows the endpoint to connect to klusterlet of managed cluster
	KlusterletEndpoint v1.EndpointAddress `json:"klusterletEndpoint,omitempty"`
	// KlusterletPort shows the port to connect to klusterlet of managed cluster
	KlusterletPort v1.EndpointPort `json:"klusterletPort,omitempty"`
	// Version of member cluster
	Version string `json:"version,omitempty"`
	// KlusterletCA is the ca data for klusterlet to authorize apiserver
	KlusterletCA []byte `json:"klusterletCA,omitempty" protobuf:"bytes,10,rep,name=klusterletCA"`
}

// +kubebuilder:object:root=true

// ClusterInfo represents the information of spoke cluster that acm hub needs to know
type ClusterInfo struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the information of the Cluster.
	// +optional
	Spec ClusterInfoSpec `json:"spec,omitempty"`
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
