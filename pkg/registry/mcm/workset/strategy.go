// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package workset

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

type worksetStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = worksetStrategy{api.Scheme, names.SimpleNameGenerator}

func (worksetStrategy) NamespaceScoped() bool {
	return true
}

func toSelectableFields(workset *mcm.WorkSet) fields.Set {
	return generic.ObjectMetaFieldsSet(&workset.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	workset, ok := obj.(*mcm.WorkSet)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a workset")
	}
	return labels.Set(workset.ObjectMeta.Labels), toSelectableFields(workset), workset.Initializers != nil, nil
}

func MatchWorkSet(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (worksetStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	workset := obj.(*mcm.WorkSet)
	workset.Status = mcm.WorkSetStatus{}
}

// Validate validates a new workset.
func (worksetStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (worksetStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for workset.
func (worksetStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (worksetStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	workset := obj.(*mcm.WorkSet)
	oldWorkSet := old.(*mcm.WorkSet)
	workset.Status = oldWorkSet.Status
}

// ValidateUpdate is the default update validation for an end user.
func (worksetStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
func (worksetStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type worksetStatusStrategy struct {
	worksetStrategy
}

var StatusStrategy = worksetStatusStrategy{Strategy}

func (worksetStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.WorkSet)
}
func (worksetStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	workset := obj.(*mcm.WorkSet)
	oldWorkSet := old.(*mcm.WorkSet)
	workset.Spec = oldWorkSet.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (worksetStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
