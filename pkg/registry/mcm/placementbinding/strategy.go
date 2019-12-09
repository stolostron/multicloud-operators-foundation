// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package placementbinding

import (
	"context"
	"fmt"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apistorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

type Strategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var DefaultStrategy = Strategy{api.Scheme, names.SimpleNameGenerator}

func (Strategy) NamespaceScoped() bool {
	return true
}

func toSelectableFields(placementBinding *mcm.PlacementBinding) fields.Set {
	return generic.ObjectMetaFieldsSet(&placementBinding.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	placementBinding, ok := obj.(*mcm.PlacementBinding)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a placementBinding")
	}
	return labels.Set(placementBinding.ObjectMeta.Labels), toSelectableFields(placementBinding), placementBinding.Initializers != nil, nil
}

func MatchPlacementBinding(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (Strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

// Validate validates a new placementBinding.
func (Strategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	pb := obj.(*mcm.PlacementBinding)
	return validatePlacementBinding(pb)
}

// Canonicalize normalizes the object after validation.
func (Strategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for placementBinding.
func (Strategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (Strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

// ValidateUpdate is the default update validation for an end user.
func (Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	pb := obj.(*mcm.PlacementBinding)
	return validatePlacementBinding(pb)
}

func (Strategy) AllowUnconditionalUpdate() bool {
	return true
}

func validatePlacementBinding(pb *mcm.PlacementBinding) field.ErrorList {
	var allErrs field.ErrorList

	if pb.PlacementPolicyRef.APIGroup == "" || pb.PlacementPolicyRef.Kind == "" || pb.PlacementPolicyRef.Name == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("PlacementBindingSpec"), "apiGroup, kind and name in PlacementRef are all required"))
	}

	for _, s := range pb.Subjects {
		if s.APIGroup == "" || s.Kind == "" || s.Name == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("PlacementBindingSpec"), "apiGroup, kind and name in Subjects are all required"))
		}
	}

	return allErrs
}
