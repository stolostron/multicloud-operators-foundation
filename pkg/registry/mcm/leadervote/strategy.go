// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package leadervote

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

type leadervoteStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = leadervoteStrategy{api.Scheme, names.SimpleNameGenerator}

func (leadervoteStrategy) NamespaceScoped() bool {
	return false
}

func ToSelectableFields(leadervote *mcm.LeaderVote) fields.Set {
	return generic.ObjectMetaFieldsSet(&leadervote.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	leadervote, ok := obj.(*mcm.LeaderVote)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a leadervote")
	}
	return labels.Set(leadervote.ObjectMeta.Labels), ToSelectableFields(leadervote), nil
}

func MatchLeaderVote(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (leadervoteStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	leadervote := obj.(*mcm.LeaderVote)
	leadervote.Status = mcm.LeaderVoteStatus{}
}

// Validate validates a new leadervote.
func (leadervoteStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (leadervoteStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for leadervote.
func (leadervoteStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (leadervoteStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	leadervote := obj.(*mcm.LeaderVote)
	oldLeaderVote := old.(*mcm.LeaderVote)
	leadervote.Status = oldLeaderVote.Status
}

// ValidateUpdate is the default update validation for an end user.
func (leadervoteStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (leadervoteStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type leadervoteStatusStrategy struct {
	leadervoteStrategy
}

var StatusStrategy = leadervoteStatusStrategy{Strategy}

func (leadervoteStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.LeaderVote)
}
func (leadervoteStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	leadervote := obj.(*mcm.LeaderVote)
	oldLeaderVote := old.(*mcm.LeaderVote)
	leadervote.Spec = oldLeaderVote.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (leadervoteStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
