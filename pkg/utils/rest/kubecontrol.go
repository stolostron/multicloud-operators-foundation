package rest

import (
	"context"
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

// KubeControlInterface to call kubernetes api
type KubeControlInterface interface {

	// Impersonate a user
	Impersonate(userID string, userGroups []string) KubeControlInterface

	// Unset impersonate headers
	UnsetImpersonate() KubeControlInterface

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

type KubeControl struct {
	mapper        meta.RESTMapper
	config        *rest.Config
	dynamicClient dynamic.Interface
}

func NewKubeControl(mapper meta.RESTMapper, config *rest.Config) *KubeControl {
	dynamicClient := dynamic.NewForConfigOrDie(config)
	return &KubeControl{
		mapper:        mapper,
		dynamicClient: dynamicClient,
		config:        config,
	}
}

// The resourceOrKindArg should be in the format of {resource}.{version}.{group}, or {resource}.
// If it is {resource}.{group}, it will be mistakenly parsed to {resource: {resource}, version: {group or part of group if there is "." in group} }
// which will trigger the reload of restmapper and introduce performance degradation.
// This is a copy from: https://github.com/kubernetes/cli-runtime/blob/e7b1ca8f27e99b474a841c85a379bf25702dfcb9/pkg/resource/builder.go#L768
// TODO: refactor restmapper to avoid frequent reload.
func MappingFor(mapper meta.RESTMapper, resourceOrKindArg string) (*meta.RESTMapping, error) {
	fullySpecifiedGVR, groupResource := schema.ParseResourceArg(resourceOrKindArg)
	gvk := schema.GroupVersionKind{}
	if fullySpecifiedGVR != nil {
		gvk, _ = mapper.KindFor(*fullySpecifiedGVR)
	}
	if gvk.Empty() {
		gvk, _ = mapper.KindFor(groupResource.WithVersion(""))
	}
	if !gvk.Empty() {
		return mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	}

	fullySpecifiedGVK, groupKind := schema.ParseKindArg(resourceOrKindArg)
	if fullySpecifiedGVK == nil {
		gvk = groupKind.WithVersion("")
		fullySpecifiedGVK = &gvk
	}

	if !fullySpecifiedGVK.Empty() {
		if mapping, err := mapper.RESTMapping(fullySpecifiedGVK.GroupKind(), fullySpecifiedGVK.Version); err == nil {
			return mapping, nil
		}
	}

	mapping, err := mapper.RESTMapping(groupKind, gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("fail to mapping GroupKind %v, GroupKindVersion %v, resource:%s err: %v", groupKind, gvk.Version, resourceOrKindArg, err)
	}

	return mapping, nil
}

func (r *KubeControl) Impersonate(userID string, userGroups []string) KubeControlInterface {
	if userID != "" && userGroups != nil {
		klog.Info("Impersonate user ", r.dynamicClient)
		impersonatedConfig := r.config
		impersonatedConfig.Impersonate.UserName = userID
		impersonatedConfig.Impersonate.Groups = userGroups
		impersonatedClient, err := dynamic.NewForConfig(impersonatedConfig)

		if err != nil {
			klog.Error(err)
			return r
		}
		r.dynamicClient = impersonatedClient
	}
	return r
}

func (r *KubeControl) UnsetImpersonate() KubeControlInterface {
	unsetImpersonatedConfig := r.config
	unsetImpersonatedConfig.Impersonate.UserName = ""
	unsetImpersonatedConfig.Impersonate.Groups = nil
	unsetImpersonatedClient, err := dynamic.NewForConfig(unsetImpersonatedConfig)
	if err != nil {
		klog.Error(err)
		return r
	}
	r.dynamicClient = unsetImpersonatedClient
	return r
}

func (r *KubeControl) Create(
	namespace string, raw runtime.RawExtension, deco func(obj runtime.Object) runtime.Object) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	err := json.Unmarshal(raw.Raw, obj)
	if err != nil {
		return nil, err
	}
	gvk := obj.GroupVersionKind()

	mapping, err := r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	objNamespace := namespace
	if objNamespace == "" {
		objNamespace = obj.GetNamespace()
	}

	if mapping.Scope.Name() == meta.RESTScopeNameNamespace && objNamespace == "" {
		return nil, fmt.Errorf("namespace must be set")
	}

	if deco != nil {
		obj = deco(obj).(*unstructured.Unstructured)
	}
	return r.dynamicClient.Resource(
		mapping.Resource).Namespace(objNamespace).Create(context.TODO(), obj, metav1.CreateOptions{})
}

// Get a resource
func (r *KubeControl) Get(
	gvk *schema.GroupVersionKind, resource, namespace, name string, serverPrint bool) (runtime.Object, error) {
	var mapping *meta.RESTMapping
	var err error

	if resource == "" {
		mapping, err = r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, err
		}
	} else {
		mapping, err = MappingFor(r.mapper, resource)
		if err != nil {
			return nil, err
		}
	}

	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// List resources
func (r *KubeControl) List(
	resource, namespace string, options *metav1.ListOptions, serverPrint bool) (runtime.Object, error) {
	mapping, err := MappingFor(r.mapper, resource)
	if err != nil {
		return nil, err
	}

	helper, err := NewHelper(r.config, mapping, serverPrint)
	if err != nil {
		return nil, err
	}

	return helper.List(namespace, options)
}

func (r *KubeControl) Delete(gvk *schema.GroupVersionKind, resource, namespace, name string) error {
	var mapping *meta.RESTMapping
	var err error

	if resource != "" {
		mapping, err = MappingFor(r.mapper, resource)
		if err != nil {
			return err
		}
	} else {
		mapping, err = r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}
	}

	deletePolicy := metav1.DeletePropagationForeground
	deleteOption := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}
	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Delete(context.TODO(), name, deleteOption)
}

func (r *KubeControl) Patch(
	namespace, name string, gvk schema.GroupVersionKind, pt types.PatchType, data []byte) (runtime.Object, error) {
	mapping, err := r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Patch(context.TODO(), name, pt, data, metav1.PatchOptions{})
}

func (r *KubeControl) Replace(namespace string, overwrite bool, raw runtime.RawExtension) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	err := json.Unmarshal(raw.Raw, obj)
	if err != nil {
		return nil, err
	}
	gvk := obj.GroupVersionKind()

	mapping, err := r.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	// Start update here
	return r.dynamicClient.Resource(mapping.Resource).Namespace(namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
}

func (r *KubeControl) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return r.mapper.KindFor(resource)
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

func (r *FakeKubeControl) getKey(namespace, name string) string {
	return fmt.Sprintf("%s.%s", namespace, name)
}

func (r *FakeKubeControl) SetObject(gvk *schema.GroupVersionKind, resource, namespace, name string, object runtime.Object) {
	key := r.getKey(namespace, name)
	r.objectsMap[key] = object
}

func (r *FakeKubeControl) Create(
	namespace string, raw runtime.RawExtension, deco func(obj runtime.Object) runtime.Object) (runtime.Object, error) {
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
func (r *FakeKubeControl) Get(
	gvk *schema.GroupVersionKind, resource, namespace, name string, serverPrint bool) (runtime.Object, error) {
	key := r.getKey(namespace, name)
	return r.objectsMap[key], nil
}

// List resources
func (r *FakeKubeControl) List(resource, namespace string, options *metav1.ListOptions, serverPrint bool) (runtime.Object, error) {
	return &unstructured.UnstructuredList{}, nil
}

func (r *FakeKubeControl) Delete(gvk *schema.GroupVersionKind, resource, namespace, name string) error {
	return nil
}

func (r *FakeKubeControl) Patch(
	namespace, name string, gvk schema.GroupVersionKind, pt types.PatchType, data []byte) (runtime.Object, error) {
	obj := &unstructured.Unstructured{}
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.SetGroupVersionKind(gvk)
	// Start patch here
	return obj, nil
}

func (r *FakeKubeControl) Replace(
	namespace string, overwrite bool, raw runtime.RawExtension) (runtime.Object, error) {
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

func (r *FakeKubeControl) Impersonate(userID string, userGroups []string) KubeControlInterface {
	if userID != "" && userGroups != nil {
		klog.Info("Impersonate user ")
	}
	return r
}

func (r *FakeKubeControl) UnsetImpersonate() KubeControlInterface {
	return r
}
