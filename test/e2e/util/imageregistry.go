package util

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const imageRegistryTemplate = `{
    "apiVersion": "imageregistry.open-cluster-management.io/v1alpha1",
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
        "registry": "quay.io/image",
        "registries": [
            {
                "mirror": "quay.io/rhacm2",
                "source": "registry.redhat.io/rhacm2"
            },
            {
                "mirror": "quay.io/multicluster-engine",
                "source": "registry.redhat.io/multicluster-engine"
            }
        ]
    }
}`

var imageRegistryGVR = schema.GroupVersionResource{
	Group:    "imageregistry.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "managedclusterimageregistries",
}

func CreateImageRegistry(dynamicClient dynamic.Interface, namespace, name, placement, pullSecret string) error {
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
	err = unstructured.SetNestedField(obj.Object, pullSecret, "spec", "pullSecret", "name")
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

func CreatePullSecret(kubeClient kubernetes.Interface, namespace, name string) error {
	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	_, err := kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), pullSecret, metav1.CreateOptions{})
	return err
}
