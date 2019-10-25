// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"context"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers"
	printersinternal "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/internalversion"
	printerstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/storage"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/workset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

type REST struct {
	*genericregistry.Store
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &mcm.WorkSet{}
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// NewREST returns a RESTStorage object that will work against worksets.
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mcm.WorkSet{} },
		NewListFunc:              func() runtime.Object { return &mcm.WorkSetList{} },
		PredicateFunc:            workset.MatchWorkSet,
		DefaultQualifiedResource: mcm.Resource("worksets"),

		CreateStrategy:      workset.Strategy,
		UpdateStrategy:      workset.Strategy,
		DeleteStrategy:      workset.Strategy,
		ReturnDeletedObject: true,

		TableConvertor: printerstorage.TableConvertor{TablePrinter: printers.NewTablePrinter().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: workset.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	statusStore := *store
	statusStore.UpdateStrategy = workset.StatusStrategy

	return &REST{store}, &StatusREST{store: &statusStore}
}
