package util

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path"

	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	clusterinfov1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers/imageregistry"
	"k8s.io/apimachinery/pkg/util/rand"

	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	apiregistrationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

const kubeConfigFileEnv = "KUBECONFIG"

var ManagedClusterGVR schema.GroupVersionResource = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
}

var ManagedClusterSetGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclustersets",
}
var ClusterInfoGVR = schema.GroupVersionResource{
	Group:    "internal.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterinfos",
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

func NewKubeConfig() (*rest.Config, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func NewKubeClient() (kubernetes.Interface, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(cfg)
}

func NewOCPClient() (openshiftclientset.Interface, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	return openshiftclientset.NewForConfig(cfg)
}

func NewHiveClient() (hiveclient.Interface, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	return hiveclient.NewForConfig(cfg)
}

func NewAPIServiceClient() (*apiregistrationclient.ApiregistrationV1Client, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	return apiregistrationclient.NewForConfig(cfg)
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

func NewDynamicClientWithImpersonate(user string, groups []string) (dynamic.Interface, error) {
	kubeConfigFile, err := getKubeConfigFile()
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	cfg.Impersonate.UserName = user
	cfg.Impersonate.Groups = groups

	return dynamic.NewForConfig(cfg)
}

func NewImageRegistryClient() (imageregistry.Interface, error) {
	kubeClient, err := NewKubeClient()
	if err != nil {
		return nil, err
	}

	return imageregistry.NewClient(kubeClient), nil
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
		list, err = dynamicClient.Resource(gvr).List(context.TODO(), listOptions)
	} else {
		list, err = dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), listOptions)
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
	obj, err := dynamicClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func GetResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	obj, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func HasResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) (bool, error) {
	_, err := dynamicClient.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func HasClusterResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, name string) (bool, error) {
	_, err := dynamicClient.Resource(gvr).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func UpdateClusterResource(dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(gvr).Update(context.TODO(), obj, metav1.UpdateOptions{})
}

func CreateResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(context.TODO(), obj, metav1.CreateOptions{})
}

func DeleteClusterResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, name string) error {
	err := dynamicClient.Resource(gvr).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func DeleteResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string) error {
	err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
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

func CheckNodeList(obj *unstructured.Unstructured) error {
	nodeList, found, err := unstructured.NestedSlice(obj.Object, "status", "nodeList")
	if err != nil || !found {
		return fmt.Errorf("failed to get nodeList. found:%v, err:%v", found, err)
	}

	if len(nodeList) == 0 {
		return fmt.Errorf("expect items in node list")
	}

	return nil
}

// Check if the current cluster is ocp by managedclusterinfo
func IsOCP(obj *unstructured.Unstructured) (bool, error) {
	distributionInfo, found, err := unstructured.NestedMap(obj.Object, "status", "distributionInfo")
	if err != nil || !found {
		return false, fmt.Errorf("failed to get distributionInfo. found:%v, err:%v", found, err)
	}
	distributionType, found, err := unstructured.NestedString(distributionInfo, "type")
	if err != nil || !found {
		return false, fmt.Errorf("failed to get distributionType. found:%v, err:%v", found, err)
	}

	if distributionType == string(clusterinfov1beta1.DistributionTypeOCP) {
		return true, nil
	}
	return false, nil
}

func CheckDistributionInfo(obj *unstructured.Unstructured) error {
	distributionInfo, found, err := unstructured.NestedMap(obj.Object, "status", "distributionInfo")
	if err != nil || !found {
		return fmt.Errorf("failed to get distributionInfo. found:%v, err:%v", found, err)
	}

	distributionType, found, err := unstructured.NestedString(distributionInfo, "type")
	if err != nil || !found {
		return fmt.Errorf("failed to get distributionType. found:%v, err:%v", found, err)
	}

	if distributionType == string(clusterinfov1beta1.DistributionTypeOCP) {
		ocpDistributionInfo, found, err := unstructured.NestedMap(distributionInfo, "ocp")
		if err != nil || !found {
			return fmt.Errorf("failed to get ocpDistributionInfo. found:%v, err:%v", found, err)
		}

		version, found, err := unstructured.NestedString(ocpDistributionInfo, "version")
		if err != nil || !found {
			return fmt.Errorf("failed to get ocp version. found:%v, err:%v", found, err)
		}

		if version == "" {
			return fmt.Errorf("failed to get valid ocp version")
		}
	}

	return nil
}

func CheckClusterID(obj *unstructured.Unstructured) error {
	distributionType, found, err := unstructured.NestedString(obj.Object, "status", "distributionInfo", "type")
	if err != nil || !found {
		return fmt.Errorf("failed to get distributionType. found:%v, err:%v", found, err)
	}

	if distributionType == string(clusterinfov1beta1.DistributionTypeOCP) {
		clusterID, found, err := unstructured.NestedString(obj.Object, "status", "clusterID")
		if err != nil || !found {
			return fmt.Errorf("failed to get ClusterID. found:%v, err:%v", found, err)
		}
		if clusterID == "" {
			return fmt.Errorf("failed to get valid ocp clusterID")
		}
	}
	return nil
}

func RandomName() string {
	return "test-automation-" + rand.String(6)
}
