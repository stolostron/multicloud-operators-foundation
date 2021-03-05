package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const GroupName = "proxy.open-cluster-management.io"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1beta1"}

var (
	localSchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = localSchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ClusterStatus{},
		&ClusterStatusList{},
		&ClusterStatusProxyOptions{},
	)
	return nil
}
