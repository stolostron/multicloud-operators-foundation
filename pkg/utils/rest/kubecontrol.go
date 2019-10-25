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
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Interface to call kubernetes api
type KubeControlInterface interface {
	// Create creates an object
	Create(namespace string, raw runtime.RawExtension, deco func(obj runtime.Object) runtime.Object) (runtime.Object, error)

	// Delete a resource
	Delete(gvk *schema.GroupVersionKind, resource, namespace, name string) error

	// Get a resource
	Get(gvk *schema.GroupVersionKind, resource, namespace, name string, serverPrint bool) (runtime.Object, error)

	// List resources
	List(resource, namespace string, options *metav1.ListOptions, serverPrint bool) (runtime.Object, error)

	// Update resource
	Replace(namespace string, overwrite bool, obj runtime.RawExtension) (runtime.Object, error)

	// Patch resource
	Patch(namespace, name string, gvk schema.GroupVersionKind, pt types.PatchType, data []byte) (runtime.Object, error)

	// KindFor returns kind of a resource
	KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error)
}

type RestKubeControl struct {
	mapper        *Mapper
	config        *rest.Config
	dynamicClient dynamic.Interface
}

func NewRestKubeControl(mapper *Mapper, config *rest.Config) *RestKubeControl {
	dynamicClient := dynamic.NewForConfigOrDie(config)
	return &RestKubeControl{
		mapper:        mapper,
		dynamicClient: dynamicClient,
		config:        config,
	}
}

func (r *RestKubeControl) Create(namespace string, raw runtime.RawExtension, deco func(obj runtime.Object) runtime.Object) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	err := json.Unmarshal(raw.Raw, obj)
	if err != nil {
		return nil, err
	}
	gvk := obj.GroupVersionKind()

	mapping, err := r.mapper.MappingForGVK(gvk)
	if err != nil {
		return nil, err
	}

	if deco != nil {
		obj = deco(obj).(*unstructured.Unstructured)
	}

	objNamespace := namespace
	if objNamespace == "" {
		objNamespace = obj.GetNamespace()
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace && objNamespace == "" {
		return nil, fmt.Errorf("namespace must be set")
	}

	return r.dynamicClient.Resource(
		mapping.Resource).Namespace(objNamespace).Create(obj, metav1.CreateOptions{})
}

// Get a resource
func (r *RestKubeControl) Get(gvk *schema.GroupVersionKind, resource, namespace, name string, serverPrint bool) (runtime.Object, error) {
	var mapping *meta.RESTMapping
	var err error

	if resource == "" {
		mapping, err = r.mapper.MappingForGVK(*gvk)
		if err != nil {
			return nil, err
		}
	} else {
		mapping, err = r.mapper.MappingFor(resource)
		if err != nil {
			return nil, err
		}
	}

	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Get(name, metav1.GetOptions{})
}

// List resources
func (r *RestKubeControl) List(resource, namespace string, options *metav1.ListOptions, serverPrint bool) (runtime.Object, error) {
	mapping, err := r.mapper.MappingFor(resource)
	if err != nil {
		return nil, err
	}

	helper, err := NewHelper(r.config, mapping, serverPrint)
	if err != nil {
		return nil, err
	}

	return helper.List(namespace, options)
}

func (r *RestKubeControl) Delete(gvk *schema.GroupVersionKind, resource, namespace, name string) error {
	var mapping *meta.RESTMapping
	var err error

	if resource != "" {
		mapping, err = r.mapper.MappingFor(resource)
		if err != nil {
			return err
		}
	} else {
		mapping, err = r.mapper.MappingForGVK(*gvk)
		if err != nil {
			return err
		}
	}

	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (r *RestKubeControl) Patch(namespace, name string, gvk schema.GroupVersionKind, pt types.PatchType, data []byte) (runtime.Object, error) {
	mapping, err := r.mapper.MappingForGVK(gvk)
	if err != nil {
		return nil, err
	}

	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Patch(name, pt, data, metav1.UpdateOptions{})
}

func (r *RestKubeControl) Replace(namespace string, overwrite bool, raw runtime.RawExtension) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	err := json.Unmarshal(raw.Raw, obj)
	if err != nil {
		return nil, err
	}
	gvk := obj.GroupVersionKind()

	mapping, err := r.mapper.MappingForGVK(gvk)
	if err != nil {
		return nil, err
	}

	// Start update here
	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Update(obj, metav1.UpdateOptions{})
}

func (r *RestKubeControl) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return r.mapper.Mapper().KindFor(resource)
}

func GeneratePatch(object runtime.Object, raw, originalRaw runtime.RawExtension) ([]byte, error) {
	// Check if the two are the same type
	var modified, original, current []byte
	var err error

	current, err = json.Marshal(object)
	if err != nil {
		return nil, err
	}

	if raw.Object != nil {
		modified, err = json.Marshal(raw.Object)
		if err != nil {
			return nil, err
		}
	} else {
		modified = raw.Raw
	}

	if originalRaw.Object != nil {
		original, err = json.Marshal(originalRaw.Object)
		if err != nil {
			return nil, err
		}
	} else {
		original = originalRaw.Raw
	}

	preconditions := []mergepatch.PreconditionFunc{mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"), mergepatch.RequireMetadataKeyUnchanged("name")}
	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current, preconditions...)
	if err != nil {
		if mergepatch.IsPreconditionFailed(err) {
			klog.V(5).Infof("%s", "At least one of apiVersion, kind and name was changed")
		} else {
			klog.V(5).Infof("failed to create merge patch: %v", err)
		}

		return nil, err
	}

	return patch, nil
}

// Fake interface for testing
type FakeKubeControl struct {
	objectsMap map[string]runtime.Object
}

func NewFakeKubeControl() *FakeKubeControl {
	return &FakeKubeControl{
		objectsMap: map[string]runtime.Object{},
	}
}

func (r *FakeKubeControl) getKey(gvk *schema.GroupVersionKind, resource, namespace, name string) string {
	return fmt.Sprintf("%s.%s", namespace, name)
}

func (r *FakeKubeControl) SetObject(gvk *schema.GroupVersionKind, resource, namespace, name string, object runtime.Object) {
	key := r.getKey(gvk, resource, namespace, name)
	r.objectsMap[key] = object
}

func (r *FakeKubeControl) Create(namespace string, raw runtime.RawExtension, deco func(obj runtime.Object) runtime.Object) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	if raw.Object != nil {
		return raw.Object, nil
	}

	err := json.Unmarshal(raw.Raw, obj)
	if err != nil {
		return nil, err
	}

	// Start patch here
	return deco(obj), nil
}

// Get a resource
func (r *FakeKubeControl) Get(gvk *schema.GroupVersionKind, resource, namespace, name string, serverPrint bool) (runtime.Object, error) {
	key := r.getKey(gvk, resource, namespace, name)
	return r.objectsMap[key], nil
}

// List resources
func (r *FakeKubeControl) List(resource, namespace string, options *metav1.ListOptions, serverPrint bool) (runtime.Object, error) {
	return &unstructured.UnstructuredList{}, nil
}

func (r *FakeKubeControl) Delete(gvk *schema.GroupVersionKind, resource, namespace, name string) error {
	return nil
}

func (r *FakeKubeControl) Patch(namespace, name string, gvk schema.GroupVersionKind, pt types.PatchType, data []byte) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.SetGroupVersionKind(gvk)
	// Start patch here
	return obj, nil
}

func (r *FakeKubeControl) Replace(namespace string, overwrite bool, raw runtime.RawExtension) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	if raw.Object != nil {
		return raw.Object, nil
	}

	err := json.Unmarshal(raw.Raw, obj)
	if err != nil {
		return nil, err
	}

	// Start patch here
	return obj, nil
}

func (r *FakeKubeControl) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}
