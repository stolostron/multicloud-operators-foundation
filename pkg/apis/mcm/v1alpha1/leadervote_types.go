// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
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
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the leader vote spec.
	// +optional
	Spec LeaderVoteSpec `json:"spec,omitempty"`

	// Status defines the status of the current leader
	Status LeaderVoteStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LeaderVoteList is a list of all the leader vote
type LeaderVoteList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Cluster objects.
	Items []LeaderVote `json:"items"`
}

// LeaderVoteSpec gives the leader vote spec
type LeaderVoteSpec struct {
	// Vote is the number that this server vote for leader
	Vote int32 `json:"vote"`

	// KubernetesAPIEndpoints represents the endpoints of the API server for this
	// cluster.
	// +optional
	KubernetesAPIEndpoints clusterv1alpha1.KubernetesAPIEndpoints `json:"kubernetesApiEndpoints,omitempty"`

	// Identity is the identity of this server
	Identity string `json:"identity"`
}

// LeaderVoteStatus gives the status of current leader vote result
type LeaderVoteStatus struct {
	// CurrentLeader shows the current leader identity
	Role string `json:"role"`
	// ReadyToServer is the flag to show whether this leader is ready to serve
	ReadyToServe bool `json:"readyToServer"`
	// LastUpdateTime shows the last leader update time
	LastUpdateTime metav1.Time `json:"lastUpdateTime"`
}
