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
	"time"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MultiNamespacedFilteredCacheBuilder implements a customized cache with a filter for specified resources with multiple namespaces
func MultiNamespacedFilteredCacheBuilder(gvkLabelMap map[schema.GroupVersionKind]Selector, namespaces []string) cache.NewCacheFunc {
	return func(config *rest.Config, opts cache.Options) (cache.Cache, error) {

		filteredCaches := map[string]cache.Cache{}

		for _, ns := range namespaces {
			opts.Namespace = ns
			newFilteredCache := NewFilteredCacheBuilder(gvkLabelMap)
			fc, err := newFilteredCache(config, opts)
			if err != nil {
				return nil, err
			}
			filteredCaches[ns] = fc
		}

		// Return the customized cache
		return &multiNamespacefilteredCache{namespaceToCache: filteredCaches, Scheme: opts.Scheme}, nil
	}
}

type multiNamespacefilteredCache struct {
	namespaceToCache map[string]cache.Cache
	Scheme           *runtime.Scheme
}

// Get implements Reader
// If the resource is in the cache, Get function get fetch in from the informer
// Otherwise, resource will be get by the k8s client
func (c multiNamespacefilteredCache) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	cache, ok := c.namespaceToCache[key.Namespace]
	if !ok {
		return fmt.Errorf("unable to get: %v because of unknown namespace for the cache", key)
	}
	return cache.Get(ctx, key, obj)
}

// List lists items out of the indexer and writes them to list
func (c multiNamespacefilteredCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)
	if listOpts.Namespace != corev1.NamespaceAll {
		cache, ok := c.namespaceToCache[listOpts.Namespace]
		if !ok {
			return fmt.Errorf("unable to get: %v because of unknown namespace for the cache", listOpts.Namespace)
		}
		return cache.List(ctx, list, opts...)
	}

	listAccessor, err := apimeta.ListAccessor(list)
	if err != nil {
		return err
	}

	allItems, err := apimeta.ExtractList(list)
	if err != nil {
		return err
	}
	var resourceVersion string
	for _, cache := range c.namespaceToCache {
		listObj := list.DeepCopyObject().(client.ObjectList)
		err = cache.List(ctx, listObj, opts...)
		if err != nil {
			return err
		}
		items, err := apimeta.ExtractList(listObj)
		if err != nil {
			return err
		}
		accessor, err := apimeta.ListAccessor(listObj)
		if err != nil {
			return fmt.Errorf("object: %T must be a list type", list)
		}
		allItems = append(allItems, items...)
		// The last list call should have the most correct resource version.
		resourceVersion = accessor.GetResourceVersion()
	}
	listAccessor.SetResourceVersion(resourceVersion)

	return apimeta.SetList(list, allItems)
}

// multiNamespaceInformer knows how to handle interacting with the underlying informer across multiple namespaces
type multiNamespaceInformer struct {
	namespaceToInformer map[string]cache.Informer
}

// AddEventHandler adds the handler to each namespaced informer
func (i *multiNamespaceInformer) AddEventHandler(handler toolscache.ResourceEventHandler) {
	for _, informer := range i.namespaceToInformer {
		informer.AddEventHandler(handler)
	}
}

// AddEventHandlerWithResyncPeriod adds the handler with a resync period to each namespaced informer
func (i *multiNamespaceInformer) AddEventHandlerWithResyncPeriod(handler toolscache.ResourceEventHandler, resyncPeriod time.Duration) {
	for _, informer := range i.namespaceToInformer {
		informer.AddEventHandlerWithResyncPeriod(handler, resyncPeriod)
	}
}

// HasSynced checks if each namespaced informer has synced
func (i *multiNamespaceInformer) HasSynced() bool {
	for _, informer := range i.namespaceToInformer {
		if ok := informer.HasSynced(); !ok {
			return ok
		}
	}
	return true
}

// AddIndexers adds the indexer for each namespaced informer
func (i *multiNamespaceInformer) AddIndexers(indexers toolscache.Indexers) error {
	for _, informer := range i.namespaceToInformer {
		err := informer.AddIndexers(indexers)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetInformer fetches or constructs an informer for the given object that corresponds to a single
// API kind and resource.
func (c multiNamespacefilteredCache) GetInformer(ctx context.Context, obj client.Object) (cache.Informer, error) {
	informers := map[string]cache.Informer{}
	for ns, cache := range c.namespaceToCache {
		informer, err := cache.GetInformer(ctx, obj)
		if err != nil {
			return nil, err
		}
		informers[ns] = informer
	}
	return &multiNamespaceInformer{namespaceToInformer: informers}, nil
}

// GetInformerForKind is similar to GetInformer, except that it takes a group-version-kind, instead
// of the underlying object.
func (c multiNamespacefilteredCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (cache.Informer, error) {
	informers := map[string]cache.Informer{}
	for ns, cache := range c.namespaceToCache {
		informer, err := cache.GetInformerForKind(ctx, gvk)
		if err != nil {
			return nil, err
		}
		informers[ns] = informer
	}
	return &multiNamespaceInformer{namespaceToInformer: informers}, nil
}

// Start runs all the informers known to this cache until the given channel is closed.
// It blocks.
func (c multiNamespacefilteredCache) Start(ctx context.Context) error {
	for ns, filteredcache := range c.namespaceToCache {
		go func(ns string, filteredcache cache.Cache) {
			err := filteredcache.Start(ctx)
			if err != nil {
				klog.Error(err, "multinamespace cache failed to start namespaced informer", "namespace", ns)
			}
		}(ns, filteredcache)
	}
	<-ctx.Done()
	return nil
}

// WaitForCacheSync waits for all the caches to sync.  Returns false if it could not sync a cache.
func (c multiNamespacefilteredCache) WaitForCacheSync(ctx context.Context) bool {
	synced := true
	for _, cache := range c.namespaceToCache {
		if s := cache.WaitForCacheSync(ctx); !s {
			synced = s
		}
	}
	return synced
}

// IndexField adds an indexer to the underlying cache, using extraction function to get
// value(s) from the given field. The filtered cache doesn't support the index yet.
func (c multiNamespacefilteredCache) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	for _, cache := range c.namespaceToCache {
		if err := cache.IndexField(ctx, obj, field, extractValue); err != nil {
			return err
		}
	}
	return nil
}
