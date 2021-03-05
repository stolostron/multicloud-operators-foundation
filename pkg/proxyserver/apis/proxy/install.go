package proxy

import (
	proxyv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

func Install(scheme *runtime.Scheme) {
	utilruntime.Must(proxyv1beta1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(proxyv1beta1.SchemeGroupVersion))
}
