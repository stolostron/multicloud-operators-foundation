package rest

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"
)

type ReloadMapper struct {
	ddm *restmapper.DeferredDiscoveryRESTMapper
}

func NewReloadMapper(ddm *restmapper.DeferredDiscoveryRESTMapper) *ReloadMapper {
	return &ReloadMapper{
		ddm: ddm,
	}
}

func (m *ReloadMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := m.ddm.KindFor(resource)
	if meta.IsNoMatchError(err) {
		m.ddm.Reset()
		return m.ddm.KindFor(resource)
	}
	return gvk, err
}

func (m *ReloadMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	gvks, err := m.ddm.KindsFor(resource)
	if meta.IsNoMatchError(err) {
		m.ddm.Reset()
		return m.ddm.KindsFor(resource)
	}
	return gvks, err
}

func (m *ReloadMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvr, err := m.ddm.ResourceFor(input)
	if meta.IsNoMatchError(err) {
		m.ddm.Reset()
		return m.ddm.ResourceFor(input)
	}
	return gvr, err
}

func (m *ReloadMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	gvrs, err := m.ddm.ResourcesFor(input)
	if err != nil {
		m.ddm.Reset()
		return m.ddm.ResourcesFor(input)
	}
	return gvrs, err
}

func (m *ReloadMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	mapping, err := m.ddm.RESTMapping(gk, versions...)
	if meta.IsNoMatchError(err) {
		m.ddm.Reset()
		return m.ddm.RESTMapping(gk, versions...)
	}
	return mapping, err
}

func (m *ReloadMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	mappings, err := m.ddm.RESTMappings(gk, versions...)
	if meta.IsNoMatchError(err) {
		m.ddm.Reset()
		return m.ddm.RESTMappings(gk, versions...)
	}
	return mappings, err
}

func (m *ReloadMapper) ResourceSingularizer(resource string) (singular string, err error) {
	singlar, err := m.ddm.ResourceSingularizer(resource)
	if meta.IsNoMatchError(err) {
		m.ddm.Reset()
		return m.ddm.ResourceSingularizer(resource)
	}
	return singlar, err
}
