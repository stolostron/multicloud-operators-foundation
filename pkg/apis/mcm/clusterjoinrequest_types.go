// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterJoinRequest is the request from spoke cluster to join Hub
type ClusterJoinRequest struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta
	// Spec defines the request information to join Hub
	Spec ClusterJoinRequestSpec
	// Derived information about the request.
	// +optional
	Status ClusterJoinRequestStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterJoinRequestList is the request list from spoke cluster to join Hub
type ClusterJoinRequestList struct {
	metav1.TypeMeta
	// +optional
	metav1.ListMeta
	Items []ClusterJoinRequest
}

type ClusterJoinRequestSpec struct {
	// The name of the spoke cluster
	ClusterName string
	// The namespace for the spoke cluster
	ClusterNamespace string
	// Base64-encoded PKCS#10 CSR data for certificate
	Request []byte
}

type ClusterJoinRequestPhase string

// These are the possible phase for a cluster join request.
const (
	JoinPhaseApproved ClusterJoinRequestPhase = "Succeeded"
	JoinPhaseDenied   ClusterJoinRequestPhase = "Failed"
	JoinPhasePending  ClusterJoinRequestPhase = "Pending"
)

type ClusterJoinRequestType string

// These are the possible conditions for a cluster join request.
const (
	JoinTypeApproved ClusterJoinRequestType = "Approved"
	JoinTypeDenied   ClusterJoinRequestType = "Denied"
)

type ClusterJoinRequestStatus struct {
	// Conditions applied to the request, such as approval or denial.
	// +optional
	Conditions []CLusterJoinRequestConditions
	// If request was approved, the controller will place the issued certificate here.
	// +optional
	Certificate []byte
	// The request phase, currently Approved, Denied or Pending.
	// +optional
	Phase ClusterJoinRequestPhase
}

type CLusterJoinRequestConditions struct {
	// request approval state, currently Approved or Denied.
	Type ClusterJoinRequestType
	// brief reason for the request state
	// +optional
	Reason string
	// human readable message with details about the request state
	// +optional
	Message string
	// timestamp for the last update to this condition
	// +optional
	LastUpdateTime metav1.Time
}
