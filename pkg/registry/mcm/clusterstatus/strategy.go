// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterstatus

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

type clusterstatusStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = clusterstatusStrategy{api.Scheme, names.SimpleNameGenerator}

func (clusterstatusStrategy) NamespaceScoped() bool {
	return true
}

func ClusterStatusToSelectableFields(clusterstatus *mcm.ClusterStatus) fields.Set {
	return generic.ObjectMetaFieldsSet(&clusterstatus.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	clusterstatus, ok := obj.(*mcm.ClusterStatus)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a clusterstatus")
	}
	return labels.Set(clusterstatus.ObjectMeta.Labels), ClusterStatusToSelectableFields(clusterstatus), clusterstatus.Initializers != nil, nil
}

func MatchClusterStatus(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (clusterstatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

// Validate validates a new clusterstatus.
func (clusterstatusStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (clusterstatusStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for clusterstatus.
func (clusterstatusStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (clusterstatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

// ValidateUpdate is the default update validation for an end user.
func (clusterstatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (clusterstatusStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type clusterstatusStatusStrategy struct {
	clusterstatusStrategy
}

var StatusStrategy = clusterstatusStatusStrategy{Strategy}

func (clusterstatusStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.ClusterStatus)
}
func (clusterstatusStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	clusterstatus := obj.(*mcm.ClusterStatus)
	oldClusterStatus := old.(*mcm.ClusterStatus)
	clusterstatus.Spec = oldClusterStatus.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (clusterstatusStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
