// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"context"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/placementpolicy"
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
	return []string{"pp"}
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &mcm.PlacementPolicy{}
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

// NewREST returns a RESTStorage object that will placementpolicy against placementpolicys.
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mcm.PlacementPolicy{} },
		NewListFunc:              func() runtime.Object { return &mcm.PlacementPolicyList{} },
		PredicateFunc:            placementpolicy.MatchPlacementPolicy,
		DefaultQualifiedResource: mcm.Resource("placementpolicys"),

		CreateStrategy:      placementpolicy.Strategy,
		UpdateStrategy:      placementpolicy.Strategy,
		DeleteStrategy:      placementpolicy.Strategy,
		ReturnDeletedObject: true,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: placementpolicy.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	statusStore := *store
	statusStore.UpdateStrategy = placementpolicy.StatusStrategy

	return &REST{store}, &StatusREST{store: &statusStore}
}
