// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"context"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/printers"
	printersinternal "github.com/open-cluster-management/multicloud-operators-foundation/pkg/printers/internalversion"
	printerstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/printers/storage"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/resourceview"
	mcmstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

type REST struct {
	store       *genericregistry.Store
	resultStore storage.Interface
}

func (r *REST) New() runtime.Object {
	return r.store.New()
}

// NewList implements rest.Lister.
func (r *REST) NewList() runtime.Object {
	return r.store.NewList()
}

// NamespaceScoped indicates whether the resource is namespaced
func (r *REST) NamespaceScoped() bool {
	return r.store.NamespaceScoped()
}

// GetCreateStrategy implements GenericStore.
func (r *REST) GetCreateStrategy() rest.RESTCreateStrategy {
	return r.store.GetCreateStrategy()
}

// GetUpdateStrategy implements GenericStore.
func (r *REST) GetUpdateStrategy() rest.RESTUpdateStrategy {
	return r.store.GetUpdateStrategy()
}

// GetDeleteStrategy implements GenericStore.
func (r *REST) GetDeleteStrategy() rest.RESTDeleteStrategy {
	return r.store.GetDeleteStrategy()
}

// GetExportStrategy implements GenericStore.
func (r *REST) GetExportStrategy() rest.RESTExportStrategy {
	return r.store.GetExportStrategy()
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	object, err := r.store.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}

	view := object.(*mcm.ResourceView)
	view.Status.Results = map[string]runtime.RawExtension{}

	// Get data from mongo
	resultList := &mcm.ResourceViewResultList{}
	selector, _ := labels.Parse(mcm.ViewLabel + "=" + view.Namespace + "." + view.Name)
	err = r.resultStore.List(ctx, "", "", storage.SelectionPredicate{
		Label: selector,
	}, resultList)
	if err != nil {
		return nil, err
	}

	for _, result := range resultList.Items {
		cluster, ok := result.Labels[mcm.ClusterLabel]
		if !ok {
			continue
		}
		// Decompress data
		resultData, rerr := mcmstorage.RetriveDataFromResult(result.Data, true)
		if rerr != nil {
			continue
		}
		view.Status.Results[cluster] = runtime.RawExtension{Raw: resultData}
	}
	return view, nil
}

// Create inserts a new item according to the unique key from the object.
func (r *REST) Create(
	ctx context.Context, obj runtime.Object,
	createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return r.store.Create(ctx, obj, createValidation, options)
}

// Update alters the status subset of an object.
func (r *REST) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
}

// List returns a list of items matching labels and field according to the
// store's PredicateFunc.
func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.store.List(ctx, options)
}

// Delete removes the item from storage.
func (r *REST) Delete(
	ctx context.Context,
	name string,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.store.Delete(ctx, name, deleteValidation, options)
}

// DeleteCollection removes all items returned by List with a given ListOptions from storage.
func (r *REST) DeleteCollection(
	ctx context.Context,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions,
	listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.store.DeleteCollection(ctx, deleteValidation, options, listOptions)
}

func (r *REST) ConvertToTable(
	ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
	return r.store.ConvertToTable(ctx, object, tableOptions)
}

// Export implements the rest.Exporter interface
func (r *REST) Export(ctx context.Context, name string, opts metav1.ExportOptions) (runtime.Object, error) {
	return r.store.Export(ctx, name, opts)
}

// Watch makes a matcher for the given label and field, and calls
// WatchPredicate. If possible, you should customize PredicateFunc to produce
// a matcher that matches by key. SelectionPredicate does this for you
// automatically.
func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return r.store.Watch(ctx, options)
}

type StatusREST struct {
	store       *genericregistry.Store
	resultStore storage.Interface
}

func (r *StatusREST) New() runtime.Object {
	return &mcm.ResourceView{}
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	object, err := r.store.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}

	view := object.(*mcm.ResourceView)
	view.Status.Results = map[string]runtime.RawExtension{}

	// Get data from mongo
	resultList := &mcm.ResourceViewResultList{}
	selector, _ := labels.Parse(mcm.ViewLabel + "=" + view.Namespace + "." + view.Name)
	err = r.resultStore.List(ctx, "", "", storage.SelectionPredicate{
		Label: selector,
	}, resultList)
	if err != nil {
		for _, result := range resultList.Items {
			cluster, ok := result.Labels[mcm.ClusterLabel]
			if !ok {
				continue
			}
			// Decompress data
			resultData, rerr := mcmstorage.RetriveDataFromResult(result.Data, true)
			if rerr != nil {
				continue
			}
			view.Status.Results[cluster] = runtime.RawExtension{Raw: resultData}
		}
	}
	return view, nil
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(
	ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// NewREST returns a RESTStorage object that will work against resourceviews.
func NewREST(optsGetter generic.RESTOptionsGetter, storageOptions *mcmstorage.Options) (*REST, *StatusREST, error) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mcm.ResourceView{} },
		NewListFunc:              func() runtime.Object { return &mcm.ResourceViewList{} },
		PredicateFunc:            resourceview.MatchResourceView,
		DefaultQualifiedResource: mcm.Resource("resourceviews"),

		CreateStrategy:      resourceview.Strategy,
		UpdateStrategy:      resourceview.Strategy,
		DeleteStrategy:      resourceview.Strategy,
		ReturnDeletedObject: true,

		TableConvertor: printerstorage.TableConvertor{
			TablePrinter: printers.NewTablePrinter().With(printersinternal.AddHandlers)},
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: resourceview.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err)
	}

	statusStore := *store
	statusStore.UpdateStrategy = resourceview.StatusStrategy

	resultStore, err := mcmstorage.NewMCMStorage(storageOptions, mcm.Kind("ResourceViewResult"))
	if err != nil {
		return nil, nil, err
	}

	return &REST{store: store, resultStore: resultStore}, &StatusREST{store: &statusStore, resultStore: resultStore}, nil
}
