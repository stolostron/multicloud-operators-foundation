// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"context"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers"
	printersinternal "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/internalversion"
	printerstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/storage"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/cluster-registry/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type REST struct {
	*genericregistry.Store
}

type StatusREST struct {
	store *genericregistry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &clusterregistry.Cluster{}
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
	forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// NewREST returns a RESTStorage object that will work against clusters.
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &clusterregistry.Cluster{} },
		NewListFunc:              func() runtime.Object { return &clusterregistry.ClusterList{} },
		PredicateFunc:            cluster.MatchCluster,
		DefaultQualifiedResource: clusterregistry.Resource("clusters"),

		CreateStrategy:      cluster.Strategy,
		UpdateStrategy:      cluster.Strategy,
		DeleteStrategy:      cluster.Strategy,
		ReturnDeletedObject: true,

		TableConvertor: printerstorage.TableConvertor{TablePrinter: printers.NewTablePrinter().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: cluster.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err)
	}

	statusStore := *store
	statusStore.UpdateStrategy = cluster.StatusStrategy

	return &REST{store}, &StatusREST{store: &statusStore}
}
