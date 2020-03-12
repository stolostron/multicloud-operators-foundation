// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterjoinrequest

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

type clusterjoinrequestStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = clusterjoinrequestStrategy{api.Scheme, names.SimpleNameGenerator}

func (clusterjoinrequestStrategy) NamespaceScoped() bool {
	return false
}

func toSelectableFields(clusterjoinrequest *mcm.ClusterJoinRequest) fields.Set {
	return generic.ObjectMetaFieldsSet(&clusterjoinrequest.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	clusterjoinrequest, ok := obj.(*mcm.ClusterJoinRequest)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a clusterjoinrequest")
	}
	return labels.Set(clusterjoinrequest.ObjectMeta.Labels), toSelectableFields(clusterjoinrequest), nil
}

func MatchClusterJoinRequest(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (clusterjoinrequestStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	clusterjoinrequest := obj.(*mcm.ClusterJoinRequest)
	clusterjoinrequest.Status = mcm.ClusterJoinRequestStatus{}
}

// Validate validates a new clusterjoinrequest.
func (clusterjoinrequestStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

// Canonicalize normalizes the object after validation.
func (clusterjoinrequestStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for clusterjoinrequest.
func (clusterjoinrequestStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (clusterjoinrequestStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	clusterjoinrequest := obj.(*mcm.ClusterJoinRequest)
	oldClusterJoinRequest := old.(*mcm.ClusterJoinRequest)
	clusterjoinrequest.Status = oldClusterJoinRequest.Status
}

// ValidateUpdate is the default update validation for an end user.
func (clusterjoinrequestStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (clusterjoinrequestStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type clusterjoinrequestStatusStrategy struct {
	clusterjoinrequestStrategy
}

var StatusStrategy = clusterjoinrequestStatusStrategy{Strategy}

func (clusterjoinrequestStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.ClusterJoinRequest)
}
func (clusterjoinrequestStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	clusterjoinrequest := obj.(*mcm.ClusterJoinRequest)
	oldClusterJoinRequest := old.(*mcm.ClusterJoinRequest)
	clusterjoinrequest.Spec = oldClusterJoinRequest.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (clusterjoinrequestStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
