package common

import (
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"

	"os"
	"os/user"
	"path"
)

const (
	kubeConfigFileEnv = "KUBECONFIG"
)

var clusterGVR schema.GroupVersionResource = schema.GroupVersionResource{
	Group:    "clusterregistry.k8s.io",
	Version:  "v1alpha1",
	Resource: "clusters",
}

var NamespaceGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

func getKubeConfigFile() (string, error) {
	kubeConfigFile := os.Getenv(kubeConfigFileEnv)
	if kubeConfigFile == "" {
		user, err := user.Current()
		if err != nil {
			return "", err
		}
		kubeConfigFile = path.Join(user.HomeDir, ".kube", "config")
	}

	return kubeConfigFile, nil
}

func NewDynamicClient() (dynamic.Interface, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Use kubeconfig file: %s\n", kubeConfigFile)

	clusterCfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}

func GetHostFromClientConfig() (string, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return "", err
	}

	clientCfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return "", err
	}

	return clientCfg.Host, nil
}

func GetReadyManagedClusters(dynamicClient dynamic.Interface) ([]*unstructured.Unstructured, error) {
	clusters, err := dynamicClient.Resource(clusterGVR).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	readyClusters := make([]*unstructured.Unstructured, 0)
	for _, cluster := range clusters.Items {
		// ignore the clusters created by testing
		if strings.HasPrefix(cluster.GetNamespace(), "test-automation-") {
			continue
		}

		conditions, ok, err := unstructured.NestedSlice(cluster.Object, "status", "conditions")
		if err != nil {
			return nil, err
		}

		if !ok || len(conditions) == 0 {
			continue
		}

		condition := conditions[0].(map[string]interface{})
		if t, ok := condition["type"]; ok {
			if t == "OK" {
				readyClusters = append(readyClusters, &cluster)
			}
		}
	}

	return readyClusters, nil
}

func LoadResourceFromJSON(json string) (*unstructured.Unstructured, error) {
	obj := unstructured.Unstructured{}
	err := obj.UnmarshalJSON([]byte(json))
	return &obj, err
}

func ListResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, labelSelector string) ([]*unstructured.Unstructured, error) {
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	var list *unstructured.UnstructuredList
	var err error
	if namespace == "" {
		list, err = dynamicClient.Resource(gvr).List(listOptions)
	} else {
		list, err = dynamicClient.Resource(gvr).Namespace(namespace).List(listOptions)
	}

	if err != nil {
		return nil, err
	}

	resources := make([]*unstructured.Unstructured, 0)
	for _, item := range list.Items {
		resources = append(resources, &item)
	}

	return resources, nil
}

func GetClusterResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, name string) (*unstructured.Unstructured, error) {
	obj, err := dynamicClient.Resource(gvr).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func GetResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func HasResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) (bool, error) {
	_, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func HasClusterResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, name string) (bool, error) {
	_, err := dynamicClient.Resource(gvr).Get(name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func CreateClusterResource(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(gvr).Create(obj, metav1.CreateOptions{})
}

func CreateResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(obj, metav1.CreateOptions{})
}

func UpdateResourceStatus(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	obj = obj.DeepCopy()
	obj.SetResourceVersion("")
	return dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).UpdateStatus(obj, metav1.UpdateOptions{})
}

func DeleteClusterResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, name string) error {
	err := dynamicClient.Resource(gvr).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func DeleteResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) error {
	err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func GeneratePrivateKey() ([]byte, error) {
	return certutil.MakeEllipticPrivateKeyPEM()
}

func GenerateCSR(clusterNamespace, clusterName string, key []byte) (string, error) {
	if key == nil {
		var err error
		key, err = certutil.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return "", err
		}
	}

	subject := &pkix.Name{
		Organization: []string{"hcm:clusters"},
		CommonName:   "hcm:clusters:" + clusterNamespace + ":" + clusterName,
	}

	privateKey, err := certutil.ParsePrivateKeyPEM(key)
	if err != nil {
		return "", err
	}
	data, err := certutil.MakeCSR(privateKey, subject, nil, nil)
	if err != nil {
		return "", err
	}

	csr := base64.StdEncoding.EncodeToString(data)
	return csr, nil
}
