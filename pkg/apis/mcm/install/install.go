// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package install

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(mcm.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(v1beta1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1alpha1.SchemeGroupVersion, v1beta1.SchemeGroupVersion))
}
