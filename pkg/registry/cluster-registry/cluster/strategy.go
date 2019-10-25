// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package cluster

import (
	"context"
	"fmt"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/clusterregistry/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apistorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type clusterStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var Strategy = clusterStrategy{api.Scheme, names.SimpleNameGenerator}

func (clusterStrategy) NamespaceScoped() bool {
	return true
}

func ClusterToSelectableFields(cluster *clusterregistry.Cluster) fields.Set {
	return generic.ObjectMetaFieldsSet(&cluster.ObjectMeta, true)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, bool, error) {
	cluster, ok := obj.(*clusterregistry.Cluster)
	if !ok {
		return nil, nil, false, fmt.Errorf("given object is not a cluster.")
	}
	return labels.Set(cluster.ObjectMeta.Labels), ClusterToSelectableFields(cluster), cluster.Initializers != nil, nil
}

func MatchCluster(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (clusterStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	cluster := obj.(*clusterregistry.Cluster)
	cluster.Status = clusterregistry.ClusterStatus{}
}

// Validate validates a new cluster.
func (clusterStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	cluster := obj.(*clusterregistry.Cluster)
	return validation.ValidateCluster(cluster)
}

// Canonicalize normalizes the object after validation.
func (clusterStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for cluster.
func (clusterStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (clusterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	cluster := obj.(*clusterregistry.Cluster)
	oldCluster := old.(*clusterregistry.Cluster)
	cluster.Status = oldCluster.Status
}

// ValidateUpdate is the default update validation for an end user.
func (clusterStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	cluster := obj.(*clusterregistry.Cluster)
	return validation.ValidateCluster(cluster)
}
func (clusterStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type clusterStatusStrategy struct {
	clusterStrategy
}

var StatusStrategy = clusterStatusStrategy{Strategy}

func (clusterStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	_ = obj.(*clusterregistry.Cluster)
}
func (clusterStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	cluster := obj.(*clusterregistry.Cluster)
	oldCluster := old.(*clusterregistry.Cluster)
	if len(cluster.Status.Conditions) != 0 && cluster.Status.Conditions[0].Type == clusterregistry.ClusterOK {
		cluster.Status.Conditions[0].LastHeartbeatTime = metav1.Now()
	}
	cluster.Spec = oldCluster.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (clusterStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}
