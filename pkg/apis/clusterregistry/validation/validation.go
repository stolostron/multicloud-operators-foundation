// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package validation

import (
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ValidateClusterName can be used to check whether the given cluster name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateClusterName = apimachineryvalidation.NameIsDNS1035Label

// ValidateObjectMeta validates an object's metadata on creation. It expects that name generation has already
// been performed.
// It doesn't return an error for rootscoped resources with namespace, because namespace should already be cleared before.
// TODO: Remove calls to this method scattered in validations of specific resources, e.g., ValidatePodUpdate.
func ValidateObjectMeta(meta *metav1.ObjectMeta, requiresNamespace bool, nameFn apimachineryvalidation.ValidateNameFunc, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateObjectMeta(meta, requiresNamespace, nameFn, fldPath)
	return allErrs
}

// ValidateCluster tests if required fields in the cluster are set.
func ValidateCluster(cluster *v1alpha1.Cluster) field.ErrorList {
	fldPath := field.NewPath("metadata")
	allErrs := ValidateObjectMeta(&cluster.ObjectMeta, true, ValidateClusterName, fldPath)

	return allErrs
}
