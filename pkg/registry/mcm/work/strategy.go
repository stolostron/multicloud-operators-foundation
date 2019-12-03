// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package work

import (
	"context"
	"fmt"
	"net/url"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/validation"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apistorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

type workStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy defines the registry strategy
var Strategy = workStrategy{api.Scheme, names.SimpleNameGenerator}

func (workStrategy) NamespaceScoped() bool {
	return true
}

// WorkToSelectableFields defines field selector
func WorkToSelectableFields(work *mcm.Work) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&work.ObjectMeta, true)
	specificFieldsSet := fields.Set{
		"type": string(work.Spec.Type),
	}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	work, ok := obj.(*mcm.Work)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a work")
	}
	return labels.Set(work.ObjectMeta.Labels), WorkToSelectableFields(work), work.Initializers != nil, nil
}

func validateWork(work *mcm.Work) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validation.ValidateWork(work)...)

	workType := work.Spec.Type

	if work.Spec.Cluster.Name == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("WorkSpec"), "work should have cluster name"))
	}

	if workType == mcm.ResourceWorkType {
		if work.Spec.Scope.ResourceType == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("WorkSpec"), "View work should have resource type"))
		}
	} else {
		if work.Spec.HelmWork == nil && work.Spec.KubeWork == nil {
			allErrs = append(allErrs, field.Required(field.NewPath("WorkSpec"), "Helm work and kube work should not be all nil"))
		}

		if work.Spec.HelmWork != nil && work.Spec.KubeWork != nil {
			allErrs = append(allErrs, field.Required(field.NewPath("WorkSpec"), "Helm and kube work should not be set at the same time"))
		}

		if workType == mcm.ActionWorkType && work.Spec.HelmWork != nil && len(work.Spec.HelmWork.ReleaseName) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("WorkAction"), "ReleaseName should not be nil for helm work"))
		}
	}

	if work.Spec.HelmWork != nil {
		chartURL := work.Spec.HelmWork.ChartURL
		_, err := url.Parse(chartURL)
		if chartURL != "" && err != nil {
			allErrs = append(allErrs, field.Required(field.NewPath("WorkDeployableHelmURL"), "HelmChart URL format is not qualified"))
		}
	}

	return allErrs
}

// MatchWork returns matched predicate
func MatchWork(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (workStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	work := obj.(*mcm.Work)
	work.Status = mcm.WorkStatus{}
}

// Validate validates a new work.
func (workStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	work := obj.(*mcm.Work)
	return validateWork(work)
}

// Canonicalize normalizes the object after validation.
func (workStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for work.
func (workStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (workStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	work := obj.(*mcm.Work)
	oldWork := old.(*mcm.Work)
	work.Status = oldWork.Status
}

// ValidateUpdate is the default update validation for an end user.
func (workStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	work := obj.(*mcm.Work)
	return validateWork(work)
}

func (workStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type workStatusStrategy struct {
	workStrategy
}

// StatusStrategy defines status storage strategy
var StatusStrategy = workStatusStrategy{Strategy}

func (workStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*mcm.Work)
}
func (workStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	work := obj.(*mcm.Work)
	oldWork := old.(*mcm.Work)
	work.Spec = oldWork.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (workStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
