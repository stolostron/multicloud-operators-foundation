package util

import (
	"context"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	clusterv1client "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/client-go/util/retry"
)

const kubeConfigFileEnv = "KUBECONFIG"

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
	clusters, err := dynamicClient.Resource(ManagedClusterGVR).List(context.TODO(), metav1.ListOptions{})
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

func CreateClusterResource(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(gvr).Create(context.TODO(), obj, metav1.CreateOptions{})
}

func CreateResource(dynamicClient dynamic.Interface, gvr schema.GroupVersionResource, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(context.TODO(), obj, metav1.CreateOptions{})
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
		rs, err = dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).UpdateStatus(context.TODO(), obj, metav1.UpdateOptions{})
		return err
	})
	return rs, err
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
	ns, err := LoadResourceFromJSON(NamespaceTemplate)
	if err != nil {
		return nil, err
	}
	ns, err = CreateClusterResource(dynamicClient, NamespaceGVR, ns)
	if err != nil {
		return nil, err
	}
	clusterNamespace := ns.GetName()

	fakeManagedCluster, err := LoadResourceFromJSON(ManagedClusterTemplate)
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

func AcceptManagedCluster(clusterName string) error {
	clusterCfg, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		return err
	}

	hubClient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	clusterClient, err := clusterv1client.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	var (
		csrs      *certificatesv1beta1.CertificateSigningRequestList
		csrClient = hubClient.CertificatesV1beta1().CertificateSigningRequests()
	)
	// Waiting for the CSR for ManagedCluster to exist
	if err := wait.Poll(1*time.Second, 120*time.Second, func() (bool, error) {
		var err error
		csrs, err = csrClient.List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("open-cluster-management.io/cluster-name = %v", clusterName),
		})
		if err != nil {
			return false, err
		}

		if len(csrs.Items) >= 1 {
			return true, nil
		}

		return false, nil
	}); err != nil {
		return err
	}
	// Approving all pending CSRs
	var csr *certificatesv1beta1.CertificateSigningRequest
	for i := range csrs.Items {
		csr = &csrs.Items[i]

		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			csr, err = csrClient.Get(context.TODO(), csr.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if isCSRInTerminalState(&csr.Status) {
				return nil
			}

			csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1beta1.CertificateSigningRequestCondition{
				Type:    certificatesv1beta1.CertificateApproved,
				Reason:  "Approved by E2E",
				Message: "Approved as part of Loopback e2e",
			})
			_, err := csrClient.UpdateApproval(context.TODO(), csr, metav1.UpdateOptions{})
			return err
		}); err != nil {
			return err
		}
	}

	var (
		managedCluster  *clusterv1.ManagedCluster
		managedClusters = clusterClient.ClusterV1().ManagedClusters()
	)
	// Waiting for ManagedCluster to exist
	if err = wait.Poll(1*time.Second, 120*time.Second, func() (bool, error) {
		var err error
		managedCluster, err = managedClusters.Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		return err
	}
	// Accepting ManagedCluster
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var err error
		managedCluster, err = managedClusters.Get(context.TODO(), managedCluster.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		managedCluster.Spec.HubAcceptsClient = true
		managedCluster.Spec.LeaseDurationSeconds = 5
		_, err = managedClusters.Update(context.TODO(), managedCluster, metav1.UpdateOptions{})
		return err
	}); err != nil {
		return err
	}

	return nil
}

func isCSRInTerminalState(status *certificatesv1beta1.CertificateSigningRequestStatus) bool {
	for _, c := range status.Conditions {
		if c.Type == certificatesv1beta1.CertificateApproved {
			return true
		}
		if c.Type == certificatesv1beta1.CertificateDenied {
			return true
		}
	}
	return false
}
