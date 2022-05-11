//
// Copyright 2022 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package filteredcache

import (
	"context"
	"fmt"
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// NewEnhancedFilteredCacheBuilder implements a customized cache with a filter for specified resources
func NewEnhancedFilteredCacheBuilder(gvkLabelsMap map[schema.GroupVersionKind][]Selector) cache.NewCacheFunc {
	return func(config *rest.Config, opts cache.Options) (cache.Cache, error) {

		// Get the frequency that informers are resynced
		var resync time.Duration
		if opts.Resync != nil {
			resync = *opts.Resync
		}

		// Generate informersmap to contain the gvks and their informers
		informersMap, err := buildInformersMap(config, opts, gvkLabelsMap, resync)
		if err != nil {
			return nil, err
		}

		// Create a default cache for the unspecified resources
		fallback, err := cache.New(config, opts)
		if err != nil {
			klog.Error(err, "Failed to init fallback cache")
			return nil, err
		}

		// Return the customized cache
		return enhancedFilteredCache{config: config, informersMap: informersMap, fallback: fallback, namespace: opts.Namespace, Scheme: opts.Scheme}, nil
	}
}

//buildInformersMap generates informersMap of the specified resource
func buildInformersMap(config *rest.Config, opts cache.Options, gvkLabelsMap map[schema.GroupVersionKind][]Selector, resync time.Duration) (map[schema.GroupVersionKind][]toolscache.SharedIndexInformer, error) {
	// Initialize informersMap
	informersMap := make(map[schema.GroupVersionKind][]toolscache.SharedIndexInformer)

	for gvk, selectors := range gvkLabelsMap {
		for _, selector := range selectors {
			// Get the plural type of the kind as resource
			plural := kindToResource(gvk.Kind)

			fieldSelector := selector.FieldSelector
			labelSelector := selector.LabelSelector
			selectorFunc := func(options *metav1.ListOptions) {
				options.FieldSelector = fieldSelector
				options.LabelSelector = labelSelector
			}

			// Create ListerWatcher with the label by NewFilteredListWatchFromClient
			client, err := getClientForGVK(gvk, config, opts.Scheme)
			if err != nil {
				return nil, err
			}
			listerWatcher := toolscache.NewFilteredListWatchFromClient(client, plural, opts.Namespace, selectorFunc)

			// Build typed runtime object for informer
			objType := &unstructured.Unstructured{}
			objType.GetObjectKind().SetGroupVersionKind(gvk)
			typed, err := opts.Scheme.New(gvk)
			if err != nil {
				return nil, err
			}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objType.UnstructuredContent(), typed); err != nil {
				return nil, err
			}

			// Create new inforemer with the listerwatcher
			informer := toolscache.NewSharedIndexInformer(listerWatcher, typed, resync, toolscache.Indexers{toolscache.NamespaceIndex: toolscache.MetaNamespaceIndexFunc})
			informersMap[gvk] = append(informersMap[gvk], informer)
			// Build list type for the GVK
			gvkList := schema.GroupVersionKind{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind + "List"}
			informersMap[gvkList] = append(informersMap[gvkList], informer)
		}
	}
	return informersMap, nil
}

// enhancedFilteredCache is the customized cache by the specified label
type enhancedFilteredCache struct {
	config       *rest.Config
	informersMap map[schema.GroupVersionKind][]toolscache.SharedIndexInformer
	fallback     cache.Cache
	namespace    string
	Scheme       *runtime.Scheme
}

// Get implements Reader
// If the resource is in the cache, Get function get fetch in from the informer
// Otherwise, resource will be get by the k8s client
func (efc enhancedFilteredCache) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {

	// Get the GVK of the runtime object
	gvk, err := apiutil.GVKForObject(obj, efc.Scheme)
	if err != nil {
		return err
	}

	if informers, ok := efc.informersMap[gvk]; ok {
		// Looking for object from the cache
		existsInCache := false
		for _, informer := range informers {
			if err := efc.getFromStore(informer, key, obj, gvk); err == nil {
				existsInCache = true
				break
			}
		}
		if !existsInCache {
			// If not found the object from cache, then fetch it from k8s apiserver
			if err := efc.getFromClient(ctx, key, obj, gvk); err != nil {
				return err
			}
		}
		return nil
	}

	// Passthrough
	return efc.fallback.Get(ctx, key, obj)
}

// getFromStore gets the resource from the cache
func (efc enhancedFilteredCache) getFromStore(informer toolscache.SharedIndexInformer, key client.ObjectKey, obj runtime.Object, gvk schema.GroupVersionKind) error {

	// Different key for cluster scope resource and namespaced resource
	var keyString string
	if key.Namespace == "" {
		keyString = key.Name
	} else {
		keyString = key.Namespace + "/" + key.Name
	}

	item, exists, err := informer.GetStore().GetByKey(keyString)
	if err != nil {
		klog.Error("Failed to get item from cache", "error", err)
		return err
	}
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{Group: gvk.Group, Resource: gvk.Kind}, key.String())
	}
	if _, isObj := item.(runtime.Object); !isObj {
		// This should never happen
		return fmt.Errorf("cache contained %T, which is not an Object", item)
	}

	// deep copy to avoid mutating cache
	item = item.(runtime.Object).DeepCopyObject()

	// Copy the value of the item in the cache to the returned value
	objVal := reflect.ValueOf(obj)
	itemVal := reflect.ValueOf(item)
	if !objVal.Type().AssignableTo(objVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", itemVal.Type(), objVal.Type())
	}
	reflect.Indirect(objVal).Set(reflect.Indirect(itemVal))
	obj.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

// getFromClient gets the resource by the k8s client
func (efc enhancedFilteredCache) getFromClient(ctx context.Context, key client.ObjectKey, obj runtime.Object, gvk schema.GroupVersionKind) error {

	// Get resource by the kubeClient
	resource := kindToResource(gvk.Kind)

	client, err := getClientForGVK(gvk, efc.config, efc.Scheme)
	if err != nil {
		return err
	}

	// Different key for cluster scope resource and namespaced resource
	restReq := client.Get()
	if key.Namespace != "" {
		restReq = restReq.Namespace(key.Namespace)
	}

	result, err := restReq.
		Name(key.Name).
		Resource(resource).
		VersionedParams(&metav1.GetOptions{}, metav1.ParameterCodec).
		Do(ctx).
		Get()

	if apierrors.IsNotFound(err) {
		return err
	} else if err != nil {
		klog.Error("Failed to retrieve resource list", "error", err)
		return err
	}

	// Copy the value of the item in the cache to the returned value
	objVal := reflect.ValueOf(obj)
	itemVal := reflect.ValueOf(result)
	if !objVal.Type().AssignableTo(objVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", itemVal.Type(), objVal.Type())
	}
	reflect.Indirect(objVal).Set(reflect.Indirect(itemVal))
	obj.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

// List lists items out of the indexer and writes them to list
func (efc enhancedFilteredCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, efc.Scheme)
	if err != nil {
		return err
	}
	if informers, ok := efc.informersMap[gvk]; ok {
		// Construct filter
		var objList []interface{}

		listOpts := client.ListOptions{}
		listOpts.ApplyOptions(opts)

		// Check the labelSelector
		var labelSel labels.Selector
		if listOpts.LabelSelector != nil {
			labelSel = listOpts.LabelSelector
		}

		// Looking for object from the cache

		if listOpts.FieldSelector != nil {
			// combining multiple indices, GetIndexers, etc
			field, val, requiresExact := requiresExactMatch(listOpts.FieldSelector)
			if !requiresExact {
				return fmt.Errorf("non-exact field matches are not supported by the cache")
			}
			// list all objects by the field selector.  If this is namespaced and we have one, ask for the
			// namespaced index key.  Otherwise, ask for the non-namespaced variant by using the fake "all namespaces"
			// namespace.
			for _, informer := range informers {
				objects, err := informer.GetIndexer().ByIndex(FieldIndexName(field), KeyToNamespacedKey(listOpts.Namespace, val))
				if err != nil {
					return err
				}
				if len(objects) != 0 {
					objList = append(objList, objects...)
				}
			}
		} else if listOpts.Namespace != "" {
			for _, informer := range informers {
				objects, err := informer.GetIndexer().ByIndex(toolscache.NamespaceIndex, listOpts.Namespace)
				if err != nil {
					return err
				}
				if len(objects) != 0 {
					objList = append(objList, objects...)
				}
			}
		} else {
			for _, informer := range informers {
				objects := informer.GetIndexer().List()
				if len(objects) != 0 {
					objList = append(objList, objects...)
				}
			}
		}

		// If not found the object from cache, then fetch the list from k8s apiserver
		if len(objList) == 0 {
			return efc.ListFromClient(ctx, list, gvk, opts...)
		}

		// Check namespace and labelSelector
		runtimeObjList := make([]runtime.Object, 0, len(objList))
		for _, item := range objList {
			obj, isObj := item.(runtime.Object)
			if !isObj {
				return fmt.Errorf("cache contained %T, which is not an Object", obj)
			}
			meta, err := apimeta.Accessor(obj)
			if err != nil {
				return err
			}

			var namespace string

			if efc.namespace != "" {
				if listOpts.Namespace != "" && efc.namespace != listOpts.Namespace {
					return fmt.Errorf("unable to list from namespace : %v because of unknown namespace for the cache", listOpts.Namespace)
				}
				namespace = efc.namespace
			} else if listOpts.Namespace != "" {
				namespace = listOpts.Namespace
			}

			if namespace != "" && namespace != meta.GetNamespace() {
				continue
			}

			if labelSel != nil {
				lbls := labels.Set(meta.GetLabels())
				if !labelSel.Matches(lbls) {
					continue
				}
			}

			outObj := obj.DeepCopyObject()
			outObj.GetObjectKind().SetGroupVersionKind(listToGVK(gvk))
			runtimeObjList = append(runtimeObjList, outObj)
		}
		return apimeta.SetList(list, runtimeObjList)
	}

	// Passthrough
	return efc.fallback.List(ctx, list, opts...)
}

// ListFromClient implements list resource by k8sClient
func (efc enhancedFilteredCache) ListFromClient(ctx context.Context, list runtime.Object, gvk schema.GroupVersionKind, opts ...client.ListOption) error {

	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)

	// Get labelselector and fieldSelector
	var labelSelector, fieldSelector string
	if listOpts.FieldSelector != nil {
		fieldSelector = listOpts.FieldSelector.String()
	}
	if listOpts.LabelSelector != nil {
		labelSelector = listOpts.LabelSelector.String()
	}

	var namespace string

	if efc.namespace != "" {
		if listOpts.Namespace != "" && efc.namespace != listOpts.Namespace {
			return fmt.Errorf("unable to list from namespace : %v because of unknown namespace for the cache", listOpts.Namespace)
		}
		namespace = efc.namespace
	} else if listOpts.Namespace != "" {
		namespace = listOpts.Namespace
	}

	resource := kindToResource(gvk.Kind[:len(gvk.Kind)-4])

	client, err := getClientForGVK(gvk, efc.config, efc.Scheme)
	if err != nil {
		return err
	}
	result, err := client.
		Get().
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		}, metav1.ParameterCodec).
		Do(ctx).
		Get()

	if err != nil {
		klog.Error("Failed to retrieve resource list: ", err)
		return err
	}

	// Copy the value of the item in the cache to the returned value
	objVal := reflect.ValueOf(list)
	itemVal := reflect.ValueOf(result)
	if !objVal.Type().AssignableTo(objVal.Type()) {
		return fmt.Errorf("cache had type %s, but %s was asked for", itemVal.Type(), objVal.Type())
	}
	reflect.Indirect(objVal).Set(reflect.Indirect(itemVal))
	list.GetObjectKind().SetGroupVersionKind(gvk)

	return nil
}

// enhancedFilteredCacheInformer knows how to handle interacting with the underlying informer with multiple internal informers
type enhancedFilteredCacheInformer struct {
	informers []toolscache.SharedIndexInformer
}

// AddEventHandler adds the handler to each internal informer
func (efci *enhancedFilteredCacheInformer) AddEventHandler(handler toolscache.ResourceEventHandler) {
	for _, informer := range efci.informers {
		informer.AddEventHandler(handler)
	}
}

// AddEventHandlerWithResyncPeriod adds the handler with a resync period to each internal informer
func (efci *enhancedFilteredCacheInformer) AddEventHandlerWithResyncPeriod(handler toolscache.ResourceEventHandler, resyncPeriod time.Duration) {
	for _, informer := range efci.informers {
		informer.AddEventHandlerWithResyncPeriod(handler, resyncPeriod)
	}
}

// HasSynced checks if each internal informer has synced
func (efci *enhancedFilteredCacheInformer) HasSynced() bool {
	for _, informer := range efci.informers {
		if ok := informer.HasSynced(); !ok {
			return ok
		}
	}
	return true
}

// AddIndexers adds the indexer for each internal informer
func (efci *enhancedFilteredCacheInformer) AddIndexers(indexers toolscache.Indexers) error {
	for _, informer := range efci.informers {
		err := informer.AddIndexers(indexers)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetInformer fetches or constructs an informer for the given object that corresponds to a single
// API kind and resource.
func (efc enhancedFilteredCache) GetInformer(ctx context.Context, obj client.Object) (cache.Informer, error) {
	gvk, err := apiutil.GVKForObject(obj, efc.Scheme)
	if err != nil {
		return nil, err
	}

	if informers, ok := efc.informersMap[gvk]; ok {
		return &enhancedFilteredCacheInformer{informers: informers}, nil
	}
	// Passthrough
	return efc.fallback.GetInformer(ctx, obj)
}

// GetInformerForKind is similar to GetInformer, except that it takes a group-version-kind, instead
// of the underlying object.
func (efc enhancedFilteredCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cache.Informer, error) {
	if informers, ok := efc.informersMap[gvk]; ok {
		return &enhancedFilteredCacheInformer{informers: informers}, nil
	}
	// Passthrough
	return efc.fallback.GetInformerForKind(ctx, gvk)
}

// Start runs all the informers known to this cache until the given channel is closed.
// It blocks.
func (efc enhancedFilteredCache) Start(ctx context.Context) error {
	klog.Info("Start enhanced filtered cache")
	for _, informers := range efc.informersMap {
		for _, informer := range informers {
			informer := informer
			go informer.Run(ctx.Done())
		}
	}
	return efc.fallback.Start(ctx)
}

// WaitForCacheSync waits for all the caches to sync.  Returns false if it could not sync a cache.
func (efc enhancedFilteredCache) WaitForCacheSync(ctx context.Context) bool {
	// Wait for informer to sync
	waiting := true
	for waiting {
		select {
		case <-ctx.Done():
			waiting = false
		case <-time.After(time.Second):
			for _, informers := range efc.informersMap {
				for _, informer := range informers {
					waiting = !informer.HasSynced() && waiting
				}
			}
		}
	}
	// Wait for fallback cache to sync
	return efc.fallback.WaitForCacheSync(ctx)
}

// IndexField adds an indexer to the underlying cache, using extraction function to get
// value(s) from the given field. The filtered cache doesn't support the index yet.
func (efc enhancedFilteredCache) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	gvk, err := apiutil.GVKForObject(obj, efc.Scheme)
	if err != nil {
		return err
	}

	if informers, ok := efc.informersMap[gvk]; ok {
		for _, informer := range informers {
			if err := indexByField(informer, field, extractValue); err != nil {
				return err
			}
			continue
		}
	}

	return efc.fallback.IndexField(ctx, obj, field, extractValue)
}
