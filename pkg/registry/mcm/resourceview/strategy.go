// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package resourceview

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

type resourceviewStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = resourceviewStrategy{api.Scheme, names.SimpleNameGenerator}

func (resourceviewStrategy) NamespaceScoped() bool {
	return true
}

func toSelectableFields(resourceview *mcm.ResourceView) fields.Set {
	return generic.ObjectMetaFieldsSet(&resourceview.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	resourceview, ok := obj.(*mcm.ResourceView)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a resourceview")
	}
	return labels.Set(resourceview.ObjectMeta.Labels), toSelectableFields(resourceview), resourceview.Initializers != nil, nil
}

func MatchResourceView(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (resourceviewStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	resourceview := obj.(*mcm.ResourceView)
	resourceview.Status = mcm.ResourceViewStatus{}
}

// Validate validates a new resourceview.
func (resourceviewStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	var allErrs field.ErrorList
	resourceview := obj.(*mcm.ResourceView)
	if resourceview.Spec.Scope.Resource == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("ResourceType"), "Resource type can not be null"))
	}
	if !(resourceview.Spec.Mode == "" || resourceview.Spec.Mode == "Periodic") {
		allErrs = append(allErrs, field.Required(field.NewPath("ResourceFilterMode"), "Resource filter mode should be null or Periodic"))
	}
	if resourceview.Spec.Mode == "Periodic" && resourceview.Spec.UpdateIntervalSeconds == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("UpdateIntervalSeconds"), "Please fill updateIntervalSeconds for Periodic mode"))
	}
	return allErrs
}

// Canonicalize normalizes the object after validation.
func (resourceviewStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for resourceview.
func (resourceviewStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (resourceviewStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	resourceview := obj.(*mcm.ResourceView)
	oldResourceView := old.(*mcm.ResourceView)
	resourceview.Status = oldResourceView.Status
}

// ValidateUpdate is the default update validation for an end user.
func (resourceviewStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	var allErrs field.ErrorList
	resourceview := obj.(*mcm.ResourceView)
	if resourceview.Spec.Scope.Resource == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("ResourceType"), "Resource type can not be null"))
	}
	if !(resourceview.Spec.Mode == "" || resourceview.Spec.Mode == "Periodic") {
		allErrs = append(allErrs, field.Required(field.NewPath("ResourceFilterMode"), "Resource filter mode should be null or Periodic"))
	}
	if resourceview.Spec.Mode == "Periodic" && resourceview.Spec.UpdateIntervalSeconds == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("UpdateIntervalSeconds"), "Please fill updateIntervalSeconds for Periodic mode"))
	}
	return allErrs
}

func (resourceviewStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type resourceviewStatusStrategy struct {
	resourceviewStrategy
}

var StatusStrategy = resourceviewStatusStrategy{Strategy}

func (resourceviewStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.ResourceView)
}
func (resourceviewStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	resourceview := obj.(*mcm.ResourceView)
	oldResourceView := old.(*mcm.ResourceView)
	resourceview.Spec = oldResourceView.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (resourceviewStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
