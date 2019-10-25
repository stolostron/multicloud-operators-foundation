// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

package rest

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

// Mapper is a struct to define resource mapping
type Mapper struct {
	mapper meta.RESTMapper

	stopCh   <-chan struct{}
	syncLock sync.RWMutex
}

// NewMapper is to create the mapper struct
func NewMapper(discoveryclient discovery.CachedDiscoveryInterface, stopCh <-chan struct{}) *Mapper {
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryclient)

	return &Mapper{
		mapper: mapper,
		stopCh: stopCh,
	}
}

// NewFakeMapper is helper function to create fake mapper
func NewFakeMapper(resources []*restmapper.APIGroupResources) *Mapper {
	return &Mapper{
		mapper: restmapper.NewDiscoveryRESTMapper(resources),
	}
}

// Run start the refresh goroutine
func (p *Mapper) Run() {
	go wait.Until(func() {
		p.syncLock.Lock()
		defer p.syncLock.Unlock()
		deferredMappd := p.mapper.(*restmapper.DeferredDiscoveryRESTMapper)
		deferredMappd.Reset()
	}, 30*time.Second, p.stopCh)
}

// MappingForGVK returns the RESTMapping for a gvk
func (p *Mapper) MappingForGVK(gvk schema.GroupVersionKind) (*meta.RESTMapping, error) {
	p.syncLock.RLock()
	defer p.syncLock.RUnlock()
	mapping, err := p.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("the server doesn't have a resource type %q", gvk.Kind)
	}

	return mapping, nil
}

func (p *Mapper) Mapper() meta.RESTMapper {
	p.syncLock.RLock()
	defer p.syncLock.RUnlock()
	return p.mapper
}

// MappingFor returns the RESTMapping for the Kind given, or the Kind referenced by the resource.
// Prefers a fully specified GroupVersionResource match. If one is not found, we match on a fully
// specified GroupVersionKind, or fallback to a match on GroupKind.
func (p *Mapper) MappingFor(resourceOrKindArg string) (*meta.RESTMapping, error) {
	p.syncLock.RLock()
	defer p.syncLock.RUnlock()
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(resourceOrKindArg)
	gvk := schema.GroupVersionKind{}
	if fullySpecifiedGVR != nil {
		gvk, _ = p.mapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, _ = p.mapper.KindFor(groupResource.WithVersion(""))
	}
	if !gvk.Empty() {
		return p.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}

	fullySpecifiedGVK, groupKind := schema.ParseKindArg(resourceOrKindArg)
	if fullySpecifiedGVK == nil {
		gvk = groupKind.WithVersion("")
		fullySpecifiedGVK = &gvk
	}

	if !fullySpecifiedGVK.Empty() {
		if mapping, err := p.mapper.RESTMapping(fullySpecifiedGVK.GroupKind(), fullySpecifiedGVK.Version); err == nil {
			return mapping, nil
		}
	}

	mapping, err := p.mapper.RESTMapping(groupKind, gvk.Version)
	if err != nil {
		// if we error out here, it is because we could not match a resource or a kind
		// for the given argument. To maintain consistency with previous behavior,
		// announce that a resource type could not be found.
		return nil, fmt.Errorf("the server doesn't have a resource type %q", groupResource.Resource)
	}

	return mapping, nil
}
