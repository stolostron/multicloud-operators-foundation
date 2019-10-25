// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorksetLabel is the label set to point to the workset
const WorkSetLabel = "mcm.ibm.com/workset"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkSetList is a list of all the works
type WorkSetList struct {
	metav1.TypeMeta
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta

	// List of Cluster objects.
	Items []WorkSet
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkSet is the work set that will be done on a set of cluster
type WorkSet struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Spec defines the behavior of the work.
	// +optional
	Spec WorkSetSpec
	// Status describes the result of a work
	// +optional
	Status WorkSetStatus
}

// WorkSetSpec is the spec for workset
type WorkSetSpec struct {
	// Selector for works.
	ClusterSelector *metav1.LabelSelector

	// Selector for works.
	Selector *metav1.LabelSelector

	// Template describes the works that will be created.
	Template WorkTemplateSpec
}

// WorkTemplateSpec describes work created from a template
type WorkTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#spec-and-status
	// +optional
	Spec WorkSpec
}

// WorkSetStatus describes the work set status
type WorkSetStatus struct {
	// Status of the work set
	Status WorkStatusType

	// Reason is the reason of the status
	Reason string
}
