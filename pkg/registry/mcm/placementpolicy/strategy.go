// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package placementpolicy

import (
	"context"
	"fmt"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/api"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apistorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

type placementpolicyStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = placementpolicyStrategy{api.Scheme, names.SimpleNameGenerator}

func (placementpolicyStrategy) NamespaceScoped() bool {
	return true
}

func toSelectableFields(placementpolicy *mcm.PlacementPolicy) fields.Set {
	return generic.ObjectMetaFieldsSet(&placementpolicy.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	placementpolicy, ok := obj.(*mcm.PlacementPolicy)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a placementpolicy")
	}
	return labels.Set(placementpolicy.ObjectMeta.Labels), toSelectableFields(placementpolicy), placementpolicy.Initializers != nil, nil
}

func MatchPlacementPolicy(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (placementpolicyStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	placementpolicy := obj.(*mcm.PlacementPolicy)
	placementpolicy.Status = mcm.PlacementPolicyStatus{}
}

// Validate validates a new placementpolicy.
func (placementpolicyStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (placementpolicyStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for placementpolicy.
func (placementpolicyStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (placementpolicyStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	placementpolicy := obj.(*mcm.PlacementPolicy)
	oldPlacementPolicy := old.(*mcm.PlacementPolicy)
	placementpolicy.Status = oldPlacementPolicy.Status
}

// ValidateUpdate is the default update validation for an end user.
func (placementpolicyStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (placementpolicyStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type placementpolicyStatusStrategy struct {
	placementpolicyStrategy
}

var StatusStrategy = placementpolicyStatusStrategy{Strategy}

func (placementpolicyStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.PlacementPolicy)
}
func (placementpolicyStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	placementpolicy := obj.(*mcm.PlacementPolicy)
	oldPlacementPolicy := old.(*mcm.PlacementPolicy)
	placementpolicy.Spec = oldPlacementPolicy.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (placementpolicyStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
