// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"bytes"
	"compress/gzip"
	"context"
	"strings"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers"
	printersinternal "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/internalversion"
	printerstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/storage"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/work"
	mcmstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/storage"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/klog"
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
	obj, err := r.store.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}
	work := obj.(*mcm.Work)

	// For resource view type work, we need to get work result
	if work.Spec.Type == mcm.ResourceWorkType {
		returnResult := &mcm.ResourceViewResult{}
		key, _ := genericregistry.NamespaceKeyFunc(ctx, "", name)
		key = strings.TrimPrefix(key, "/")
		err := r.resultStore.Get(ctx, key, "", returnResult, false)
		if err != nil {
			klog.V(5).Infof("failed to get result of work %s: %v", name, err)
			return work, nil
		}
		resultData, err := mcmstorage.RetriveDataFromResult(returnResult.Data, true)
		if err != nil {
			klog.V(5).Infof("failed to retrieve data from result %s: %v", name, err)
			return work, nil
		}

		work.Status.Result = runtime.RawExtension{Raw: resultData}
	}

	return work, err
}

// Create inserts a new item according to the unique key from the object.
func (r *REST) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	return r.store.Create(ctx, obj, createValidation, options)
}

// Update alters the status subset of an object.
func (r *REST) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, forceAllowCreate, options)
}

// List returns a list of items matching labels and field according to the
// store's PredicateFunc.
func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.store.List(ctx, options)
}

// Delete removes the item from storage.
func (r *REST) Delete(ctx context.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	object, err := r.store.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	work := object.(*mcm.Work)
	if work.Spec.Type == mcm.ResourceWorkType {
		key, _ := genericregistry.NamespaceKeyFunc(ctx, "", name)
		key = strings.TrimPrefix(key, "/")
		r.resultStore.Delete(ctx, key, nil, &storage.Preconditions{})
	}
	return r.store.Delete(ctx, name, options)
}

// DeleteCollection removes all items returned by List with a given ListOptions from storage.
func (r *REST) DeleteCollection(ctx context.Context, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.store.DeleteCollection(ctx, options, listOptions)
}

func (r *REST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1beta1.Table, error) {
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
	return &mcm.Work{}
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	obj, err := r.store.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	updatedObjInfo := objInfo

	work := obj.(*mcm.Work)
	if work.Spec.Type == mcm.ResourceWorkType {
		updateObj, err := objInfo.UpdatedObject(ctx, obj)
		if err != nil {
			return nil, false, err
		}
		updatedWork := updateObj.(*mcm.Work)
		resultData := &mcm.ResourceViewResult{
			ObjectMeta: metav1.ObjectMeta{
				Name:      updatedWork.Name,
				Namespace: updatedWork.Namespace,
				Labels:    updatedWork.Labels,
			},
		}
		if resultData.Labels == nil {
			resultData.Labels = make(map[string]string)
		}
		resultData.Labels[mcm.ClusterLabel] = updatedWork.Spec.Cluster.Name
		// If the status is no nil, put the result into mongo
		if updatedWork.Status.Result.Raw != nil {
			var compressed bytes.Buffer
			w, decerr := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
			if decerr != nil {
				return nil, false, decerr
			}
			_, decerr = w.Write(updatedWork.Status.Result.Raw)
			w.Close()
			if decerr != nil {
				return nil, false, decerr
			}
			resultData.Data = compressed.Bytes()
			returnResult := &mcm.ResourceViewResult{}
			key, _ := genericregistry.NamespaceKeyFunc(ctx, "", name)
			key = strings.TrimPrefix(key, "/")
			err = r.resultStore.GuaranteedUpdate(ctx, key, returnResult, true, &storage.Preconditions{}, func(existing runtime.Object, res storage.ResponseMeta) (runtime.Object, *uint64, error) {
				return resultData, nil, nil
			})

			if err != nil {
				return nil, false, err
			}
			updatedWork.Status.Result.Raw = nil
			updatedObjInfo = rest.DefaultUpdatedObjectInfo(updatedWork)
		}
	}

	return r.store.Update(ctx, name, updatedObjInfo, createValidation, updateValidation, false, options)
}

// NewREST returns a RESTStorage object that will work against works.
func NewREST(optsGetter generic.RESTOptionsGetter, storageOptions *mcmstorage.StorageOptions) (*REST, *StatusREST, error) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mcm.Work{} },
		NewListFunc:              func() runtime.Object { return &mcm.WorkList{} },
		PredicateFunc:            work.MatchWork,
		DefaultQualifiedResource: mcm.Resource("works"),

		CreateStrategy:      work.Strategy,
		UpdateStrategy:      work.Strategy,
		DeleteStrategy:      work.Strategy,
		ReturnDeletedObject: true,

		TableConvertor: printerstorage.TableConvertor{TablePrinter: printers.NewTablePrinter().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: work.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	statusStore := *store
	statusStore.UpdateStrategy = work.StatusStrategy

	resultStore, err := mcmstorage.NewMCMStorage(storageOptions, mcm.Kind("ResourceViewResult"))
	if err != nil {
		return nil, nil, err
	}

	return &REST{store: store, resultStore: resultStore}, &StatusREST{store: &statusStore, resultStore: resultStore}, nil
}
