package util

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const imageRegistryTemplate = `{
    "apiVersion": "cluster.open-cluster-management.io/v1alpha1",
    "kind": "ManagedClusterImageRegistry",
    "metadata": {
        "name": "imageRegistry",
        "namespace": "default"
    },
    "spec": {
        "placementRef": {
            "group": "cluster.open-cluster-management.io",
            "name": "placement",
            "resource": "placements"
        },
        "pullSecret": {
            "name": "pullSecret"
        },
        "registry": "quay.io/image"
    }
}`

var imageRegistryGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "managedclusterimageregistries",
}

func CreateImageRegistry(dynamicClient dynamic.Interface, namespace, name, placement string) error {
	obj, err := LoadResourceFromJSON(imageRegistryTemplate)
	if err != nil {
		return err
	}
	err = unstructured.SetNestedField(obj.Object, namespace, "metadata", "namespace")
	if err != nil {
		return err
	}
	err = unstructured.SetNestedField(obj.Object, name, "metadata", "name")
	if err != nil {
		return err
	}
	err = unstructured.SetNestedField(obj.Object, placement, "spec", "placementRef", "name")
	if err != nil {
		return err
	}

	_, err = CreateResource(dynamicClient, imageRegistryGVR, obj)
	return err
}

func DeleteImageRegistry(dynamicClient dynamic.Interface, namespace, name string) error {
	return DeleteResource(dynamicClient, imageRegistryGVR, namespace, name)
}

func GetImageRegistry(dynamicClient dynamic.Interface, namespace, name string) (*unstructured.Unstructured, error) {
	return GetResource(dynamicClient, imageRegistryGVR, namespace, name)
}
