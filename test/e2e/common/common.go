package common

import (
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/template"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/client-go/util/retry"
)

const (
	kubeConfigFileEnv  = "KUBECONFIG"
	SingleManagedOnHub = true
)

var ManagedClusterGVR schema.GroupVersionResource = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
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

func GetJoinedManagedClusters(dynamicClient dynamic.Interface) ([]*unstructured.Unstructured, error) {
	clusters, err := dynamicClient.Resource(ManagedClusterGVR).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	readyClusters := make([]*unstructured.Unstructured, 0)
	for _, cluster := range clusters.Items {
		conditions, ok, err := unstructured.NestedSlice(cluster.Object, "status", "conditions")
		if err != nil {
			return nil, err
		}

		if !ok || len(conditions) == 0 {
			continue
		}

		if strings.HasPrefix(cluster.GetName(), "test-automation") {
			continue
		}

		for _, condition := range conditions {
			if t, ok := condition.(map[string]interface{})["type"]; ok {
				if t == clusterv1.ManagedClusterConditionJoined {
					readyClusters = append(readyClusters, cluster.DeepCopy())
					break
				}
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
		resources = append(resources, item.DeepCopy())
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
	var rs *unstructured.Unstructured

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		oldObj, err := GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
		if err != nil {
			return err
		}
		obj.SetResourceVersion(oldObj.GetResourceVersion())
		rs, err = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).UpdateStatus(obj, metav1.UpdateOptions{})
		return err
	})
	return rs, err
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
	return keyutil.MakeEllipticPrivateKeyPEM()
}

func GenerateCSR(clusterNamespace, clusterName string, key []byte) (string, error) {
	if key == nil {
		var err error
		key, err = keyutil.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return "", err
		}
	}

	subject := &pkix.Name{
		Organization: []string{"hcm:clusters"},
		CommonName:   "hcm:clusters:" + clusterNamespace + ":" + clusterName,
	}

	privateKey, err := keyutil.ParsePrivateKeyPEM(key)
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

func SetStatusType(obj *unstructured.Unstructured, statusType string) error {
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return err
	}

	if conditions == nil {
		conditions = make([]interface{}, 0)
	}

	if len(conditions) == 0 {
		conditions = append(conditions, map[string]interface{}{
			"status": "True",
			"type":   statusType,
		})
		err := unstructured.SetNestedField(obj.Object, conditions, "status", "conditions")
		if err != nil {
			return err
		}
	} else {
		condition := conditions[0].(map[string]interface{})
		condition["status"] = "True"
		condition["type"] = statusType
	}

	return nil
}

func GetConditionFromStatus(obj *unstructured.Unstructured) (map[string]interface{}, error) {
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return nil, err
	}

	if conditions == nil {
		return nil, nil
	}

	condition, _ := conditions[0].(map[string]interface{})
	return condition, nil
}

func GetConditionTypeFromStatus(obj *unstructured.Unstructured, typeName string) bool {
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return false
	}

	if conditions == nil {
		return false
	}
	for _, condition := range conditions {
		conditionValue, _ := condition.(map[string]interface{})
		if conditionValue["type"] == typeName {
			return true
		}
	}
	return false
}

func CreateManagedCluster(dynamicClient dynamic.Interface) (*unstructured.Unstructured, error) {
	// create a namespace for testing
	ns, err := LoadResourceFromJSON(template.NamespaceTemplate)
	if err != nil {
		return nil, err
	}
	ns, err = CreateClusterResource(dynamicClient, NamespaceGVR, ns)
	if err != nil {
		return nil, err
	}
	clusterNamespace := ns.GetName()

	fakeManagedCluster, err := LoadResourceFromJSON(template.ManagedClusterTemplate)
	if err != nil {
		return nil, err
	}

	// setup fakeManagedCluster
	err = unstructured.SetNestedField(fakeManagedCluster.Object, clusterNamespace, "metadata", "name")
	if err != nil {
		return nil, err
	}

	// create a fakeManagedCluster
	fakeManagedCluster, err = CreateClusterResource(dynamicClient, ManagedClusterGVR, fakeManagedCluster)
	if err != nil {
		return nil, err
	}

	exists, err := HasResource(dynamicClient, ManagedClusterGVR, fakeManagedCluster.GetNamespace(), fakeManagedCluster.GetName())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("failed to get managedcluster: %v", err)
	}

	// update fakeManagedCluster status with the new client
	err = SetStatusType(fakeManagedCluster, clusterv1.ManagedClusterConditionJoined)
	if err != nil {
		return nil, err
	}

	fakeManagedCluster, err = UpdateResourceStatus(dynamicClient, ManagedClusterGVR, fakeManagedCluster)
	if err != nil {
		return nil, err
	}

	return fakeManagedCluster, nil
}
