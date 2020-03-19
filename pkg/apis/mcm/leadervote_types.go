// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LeaderVote keeps the leader election status
type LeaderVote struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the leader vote spec.
	// +optional
	Spec LeaderVoteSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Status defines the status of the current leader
	Status LeaderVoteStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LeaderVoteList is a list of all the leader vote
type LeaderVoteList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Cluster objects.
	Items []LeaderVote `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// LeaderVoteSpec gives the leader vote spec
type LeaderVoteSpec struct {
	// Vote is the number that this server vote for leader
	Vote int32 `json:"vote" protobuf:"varint,1,opt,name=vote"`

	// KubernetesAPIEndpoints represents the endpoints of the API server for this
	// cluster.
	// +optional
	KubernetesAPIEndpoints KubernetesAPIEndpoints `json:"kubernetesApiEndpoints,omitempty" protobuf:"bytes,2,rep,name=kubernetesApiEndpoints"`

	// Identity is the identity of this server
	Identity string `json:"identity" protobuf:"bytes,3,opt,name=identity"`
}

// LeaderVoteStatus gives the status of current leader vote result
type LeaderVoteStatus struct {
	// CurrentLeader shows the current leader identity
	Role string `json:"role" protobuf:"bytes,1,opt,name=role"`
	// ReadyToServer is the flag to show whether this leader is ready to serve
	ReadyToServe bool `json:"readyToServer" protobuf:"bytes,2,opt,name=readyToServer"`
	// LastUpdateTime shows the last leader update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime" protobuf:"bytes,3,opt,name=lastUpdateTime"`
}

// KubernetesAPIEndpoints represents the endpoints for one and only one
// Kubernetes API server.
type KubernetesAPIEndpoints struct {
	// ServerEndpoints specifies the address(es) of the Kubernetes API server’s
	// network identity or identities.
	// +optional
	ServerEndpoints []ServerAddressByClientCIDR `json:"serverEndpoints,omitempty" protobuf:"bytes,1,rep,name=serverEndpoints"`

	// CABundle contains the certificate authority information.
	// +optional
	CABundle []byte `json:"caBundle,omitempty" protobuf:"bytes,2,opt,name=caBundle"`
}

// ServerAddressByClientCIDR helps clients determine the server address that
// they should use, depending on the ClientCIDR that they match.
type ServerAddressByClientCIDR struct {
	// The CIDR with which clients can match their IP to figure out if they should
	// use the corresponding server address.
	// +optional
	ClientCIDR string `json:"clientCIDR,omitempty" protobuf:"bytes,1,opt,name=clientCIDR"`
	// Address of this server, suitable for a client that matches the above CIDR.
	// This can be a hostname, hostname:port, IP or IP:port.
	// +optional
	ServerAddress string `json:"serverAddress,omitempty" protobuf:"bytes,2,opt,name=serverAddress"`
}
