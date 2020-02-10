// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"context"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/placementbinding"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

type REST struct {
	*genericregistry.Store
}

// ShortNames implements the ShortNamesProvider interface. Returns a list of short names for a resource.
func (r *REST) ShortNames() []string {
	return []string{"pb"}
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &mcm.PlacementBinding{}
}

func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(
	ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// NewREST returns a RESTStorage object that will placementbinding against placementbindings.
func NewREST(optsGetter generic.RESTOptionsGetter) *REST {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mcm.PlacementBinding{} },
		NewListFunc:              func() runtime.Object { return &mcm.PlacementBindingList{} },
		PredicateFunc:            placementbinding.MatchPlacementBinding,
		DefaultQualifiedResource: mcm.Resource("placementbindings"),

		CreateStrategy:      placementbinding.DefaultStrategy,
		UpdateStrategy:      placementbinding.DefaultStrategy,
		DeleteStrategy:      placementbinding.DefaultStrategy,
		ReturnDeletedObject: true,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: placementbinding.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err)
	}

	return &REST{store}
}
