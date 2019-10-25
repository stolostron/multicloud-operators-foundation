// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterLabel is the label set to point to the cluster
const ClusterLabel = "mcm.ibm.com/cluster"

// ClusterStatusSpec is information about the current status of a cluster updated by cluster controller periodically.
type ClusterStatusSpec struct {
	// MasterIP shows the master IP of managed cluster
	MasterAddresses []v1.EndpointAddress
	// ConcoleURL shows the url of icp console in managed cluster
	ConsoleURL string
	// Capacity
	Capacity v1.ResourceList
	// Usage
	Usage v1.ResourceList
	// KlusterletEndpoint shows the endpoint to connect to klusterlet of managed cluster
	KlusterletEndpoint v1.EndpointAddress
	// KlusterletPort shows the port to connect to klusterlet of managed cluster
	KlusterletPort v1.EndpointPort
	// Version of Klusterlet
	KlusterletVersion string
	// KlusterletCA is the ca data for klusterlet to authorize apiserver
	KlusterletCA []byte
	// MonitoringScrapeTarget is the scrape target to be used
	MonitoringScrapeTarget string
	// Version is the kubernetes version of the memeber cluster
	Version string
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterStatus about a registered cluster in a federated kubernetes setup.
type ClusterStatus struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Spec defines the behavior of the Cluster.
	// +optional
	Spec ClusterStatusSpec
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterStatusList is a list of all the kubernetes clusters status in the hcm
type ClusterStatusList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta

	// List of Cluster objects.
	Items []ClusterStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodLogOptions is the option for pod
type ClusterRestOptions struct {
	metav1.TypeMeta

	// Path is the URL path to use for the current proxy request
	Path string
}
