// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package validation

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateWorkName can be used to check whether the given work name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateWorkName = apimachineryvalidation.NameIsDNS1035Label

// ValidateWork tests if required fields in the work are set.
func ValidateWork(work *mcm.Work) field.ErrorList {
	fldPath := field.NewPath("metadata")
	allErrs := apimachineryvalidation.ValidateObjectMeta(&work.ObjectMeta, true, ValidateWorkName, fldPath)

	return allErrs
}
