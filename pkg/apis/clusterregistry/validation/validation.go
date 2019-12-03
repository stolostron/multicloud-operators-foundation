// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package validation

import (
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ValidateClusterName can be used to check whether the given cluster name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateClusterName = apimachineryvalidation.NameIsDNS1035Label

// ValidateCluster tests if required fields in the cluster are set.
func ValidateCluster(cluster *v1alpha1.Cluster) field.ErrorList {
	fldPath := field.NewPath("metadata")
	allErrs := apimachineryvalidation.ValidateObjectMeta(&cluster.ObjectMeta, true, ValidateClusterName, fldPath)

	return allErrs
}
