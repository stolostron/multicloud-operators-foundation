// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package mcm

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the group name use in this package
const GroupName = "mcm.ibm.com"

const (
	//OwnersLabel is the label set to point to resource's owners
	OwnersLabel = "mcm.ibm.com/owners"

	// ClusterOwnerAnnotationKey is the key of hub owner
	ClusterOwnerAnnotationKey = "mcm.ibm.com/hub"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
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
		&ClusterRestOptions{},
	)
	return nil
}
