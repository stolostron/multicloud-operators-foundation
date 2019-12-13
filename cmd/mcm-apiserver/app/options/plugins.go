// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package options

// This file exists to force the desired plugin implementations to be linked.
// This should probably be part of some configuration fed into the build for a
// given binary target.
import (
	"k8s.io/apiserver/pkg/admission"
	// Admission controllers
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/plugin/pkg/klusterletca"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/plugin/pkg/useridentity"
)

// registerAllAdmissionPlugins registers all admission plugins
func registerAllAdmissionPlugins(plugins *admission.Plugins, caFile *string) {
	useridentity.Register(plugins)
	klusterletca.Register(plugins, caFile)
}
