// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the group name use in this package
const GroupName = "mcm.ibm.com"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	localSchemeBuilder = &SchemeBuilder
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ClusterStatus{},
		&ClusterStatusList{},
		&Work{},
		&WorkList{},
		&WorkSet{},
		&WorkSetList{},
		&ResourceView{},
		&ResourceViewList{},
		&ResourceViewResult{},
		&ResourceViewResultList{},
		&ClusterJoinRequest{},
		&ClusterJoinRequestList{},
		&PlacementBinding{},
		&PlacementBindingList{},
		&PlacementPolicy{},
		&PlacementPolicyList{},
		&ClusterRestOptions{},
		&metav1.GetOptions{},
		&metav1.ExportOptions{},
		&metav1.ListOptions{},
	)

	return nil
}
