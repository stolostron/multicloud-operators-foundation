// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta2

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Work is the work that will be done on a set of cluster
type Work struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the work.
	// +optional
	Spec WorkSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Result describes the result of a work
	// +optional
	Status WorkStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkList is a list of all the works
type WorkList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Cluster objects.
	Items []Work `json:"items" protobuf:"bytes,2,opt,name=items"`
}

// WorkSpec defines the work to be processes on a set of clusters
type WorkSpec struct {
	// Cluster is a selector of cluster
	Cluster v1.LocalObjectReference `json:"cluster,omitempty" protobuf:"bytes,1,opt,name=cluster"`

	// Type defins the type of the woke to be done
	Type WorkType `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
}

// WorkStatus returns the status of the work
type WorkStatus struct {
	// Reason is the reason of the current status
	Reason string `json:"reason,omitempty" protobuf:"bytes,1,opt,name=reason"`
}

// WorkType defines the type of cluster status
type WorkType string
