// +build integration

package clusters_test

import (
	"encoding/base64"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/template"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	eventuallyTimeout  = 60
	eventuallyInterval = 1
	offlineReason      = "Klusterlet failed to update cluster status on time"
)

var gvr = schema.GroupVersionResource{
	Group:    "clusterregistry.k8s.io",
	Version:  "v1alpha1",
	Resource: "clusters",
}

var roleGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "roles",
}

var roleBindingGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "rolebindings",
}

var cjrGVR = schema.GroupVersionResource{
	Group:    "mcm.ibm.com",
	Version:  "v1beta1",
	Resource: "clusterjoinrequests",
}

var (
	dynamicClient dynamic.Interface
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = Describe("Clusters", func() {
	var (
		obj       *unstructured.Unstructured
		err       error
		namespace string
	)

	BeforeEach(func() {
		// create a namespace for testing
		ns, err := common.LoadResourceFromJSON(template.NamespaceTemplate)
		Ω(err).ShouldNot(HaveOccurred())
		ns, err = common.CreateClusterResource(dynamicClient, common.NamespaceGVR, ns)
		Ω(err).ShouldNot(HaveOccurred())
		namespace = ns.GetName()

		obj, err = common.LoadResourceFromJSON(template.ClusterTemplate)
		Ω(err).ShouldNot(HaveOccurred())

		// setup cluster
		err = unstructured.SetNestedField(obj.Object, namespace, "metadata", "namespace")
		Ω(err).ShouldNot(HaveOccurred())
		err = unstructured.SetNestedField(obj.Object, namespace, "metadata", "name")
		Ω(err).ShouldNot(HaveOccurred())

		// create a cluster
		obj, err = common.CreateResource(dynamicClient, gvr, obj)
		Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
	})

	Describe("Creating a cluster", func() {
		It("should be created successfully", func() {
			exists, err := common.HasResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should create role and rolebinding automatically for the cluster as well successfully", func() {
			Eventually(func() (bool, error) {
				exists, err := common.HasResource(dynamicClient, roleGVR, obj.GetNamespace(), obj.GetName())
				if err != nil || !exists {
					return false, err
				}

				return common.HasResource(dynamicClient, roleBindingGVR, obj.GetNamespace(), obj.GetName())
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Updating cluster status", func() {
		var cjr *unstructured.Unstructured

		BeforeEach(func() {
			// create a CSR with a private key
			key, err := common.GeneratePrivateKey()
			Ω(err).ShouldNot(HaveOccurred())

			data, err := common.GenerateCSR(namespace, namespace, key)
			Ω(err).ShouldNot(HaveOccurred())

			// create clusterjoinrequest for the new cluster
			cjr, err = common.LoadResourceFromJSON(template.ClusterJoinRequestTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(cjr.Object, namespace, "spec", "clusterNameSpace")
			Ω(err).ShouldNot(HaveOccurred())
			err = unstructured.SetNestedField(cjr.Object, namespace, "spec", "clusterName")
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(cjr.Object, data, "spec", "csr", "request")
			Ω(err).ShouldNot(HaveOccurred())

			cjr, err = common.CreateClusterResource(dynamicClient, cjrGVR, cjr)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", cjrGVR.Resource)

			// wait until clusterjoinrequest is approved
			var certificate string
			Eventually(func() (string, error) {
				cjr, err = common.GetClusterResource(dynamicClient, cjrGVR, cjr.GetName())
				if err != nil {
					return "", err
				}

				certificate, _, err = unstructured.NestedString(cjr.Object, "status", "csrStatus", "certificate")
				return certificate, err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(BeZero())

			// create a new client with certificate from clusterjoinrequest status
			certificateBytes, err := base64.StdEncoding.DecodeString(certificate)
			Ω(err).ShouldNot(HaveOccurred())

			host, err := common.GetHostFromClientConfig()
			Ω(err).ShouldNot(HaveOccurred())

			newDynamicClient, err := newDynamicClientWithCertAndKey(host, certificateBytes, key)
			Ω(err).ShouldNot(HaveOccurred())

			// wait until role and rolebinding are created
			Eventually(func() (bool, error) {
				exists, err := common.HasResource(dynamicClient, roleGVR, obj.GetNamespace(), obj.GetName())
				if err != nil || !exists {
					return false, err
				}

				return common.HasResource(dynamicClient, roleBindingGVR, obj.GetNamespace(), obj.GetName())
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

			// update cluster status with the new client
			err = setStatusType(obj, "OK")
			Ω(err).ShouldNot(HaveOccurred())

			obj, err = common.UpdateResourceStatus(newDynamicClient, gvr, obj)
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("With heartbeat", func() {
			It("should be marked as Ready by controller successfully", func() {
				// check cluster status
				Eventually(func() (interface{}, error) {
					cluster, err := common.GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
					if err != nil {
						return "", err
					}

					condition, err := getConditionFromStatus(cluster)
					if err != nil {
						return "", err
					}
					if condition == nil {
						return "", nil
					}

					return condition["type"], nil
				}, eventuallyTimeout, eventuallyInterval).Should(Equal("OK"))
			})
		})

		Context("With expired heartbeat", func() {
			BeforeEach(func() {
				// sleep for 1 minite and wait for cluster status is set to offline
				time.Sleep(1 * time.Minute)
			})

			It("should be marked as Offline by controller after 1 minute successfully", func() {
				// check if the cluster status is set correctly by controller
				Eventually(func() (bool, error) {
					cluster, err := common.GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
					if err != nil {
						return false, err
					}
					condition, err := getConditionFromStatus(cluster)
					if err != nil {
						return false, err
					}
					if condition == nil {
						return false, nil
					}

					if condition["type"] != "" {
						return false, nil
					}
					if condition["reason"] != offlineReason {
						return false, nil
					}
					return true, nil
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
			})
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())

			// delete clusterjoinrequest created
			err = common.DeleteClusterResource(dynamicClient, cjrGVR, cjr.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Deny cluster join request", func() {
		var cjr *unstructured.Unstructured
		var data string

		BeforeEach(func() {
			// create a CSR with a private key
			key, err := common.GeneratePrivateKey()
			Ω(err).ShouldNot(HaveOccurred())

			data, err = common.GenerateCSR(namespace, namespace, key)
			Ω(err).ShouldNot(HaveOccurred())

			// create clusterjoinrequest for the new cluster
			cjr, err = common.LoadResourceFromJSON(template.ClusterJoinRequestTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(cjr.Object, namespace, "spec", "clusterNameSpace")
			Ω(err).ShouldNot(HaveOccurred())
			err = unstructured.SetNestedField(cjr.Object, namespace, "spec", "clusterName")
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(cjr.Object, data, "spec", "csr", "request")
			Ω(err).ShouldNot(HaveOccurred())

			cjr, err = common.CreateClusterResource(dynamicClient, cjrGVR, cjr)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", cjrGVR.Resource)

			// wait until clusterjoinrequest is approved
			var certificate string
			Eventually(func() (string, error) {
				cjr, err = common.GetClusterResource(dynamicClient, cjrGVR, cjr.GetName())
				if err != nil {
					return "", err
				}

				certificate, _, err = unstructured.NestedString(cjr.Object, "status", "csrStatus", "certificate")
				return certificate, err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(BeZero())

			// create a new client with certificate from clusterjoinrequest status
			certificateBytes, err := base64.StdEncoding.DecodeString(certificate)
			Ω(err).ShouldNot(HaveOccurred())

			host, err := common.GetHostFromClientConfig()
			Ω(err).ShouldNot(HaveOccurred())

			newDynamicClient, err := newDynamicClientWithCertAndKey(host, certificateBytes, key)
			Ω(err).ShouldNot(HaveOccurred())

			// wait until role and rolebinding are created
			Eventually(func() (bool, error) {
				exists, err := common.HasResource(dynamicClient, roleGVR, obj.GetNamespace(), obj.GetName())
				if err != nil || !exists {
					return false, err
				}

				return common.HasResource(dynamicClient, roleBindingGVR, obj.GetNamespace(), obj.GetName())
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

			// update cluster status with the new client
			err = setStatusType(obj, "OK")
			Ω(err).ShouldNot(HaveOccurred())

			obj, err = common.UpdateResourceStatus(newDynamicClient, gvr, obj)
			Ω(err).ShouldNot(HaveOccurred())

			// marked as Ready by controller successfully
			Eventually(func() (interface{}, error) {
				cluster, err := common.GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
				if err != nil {
					return "", err
				}

				condition, err := getConditionFromStatus(cluster)
				if err != nil {
					return "", err
				}
				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("OK"))

		})

		Context("Deny clusterjoinrequest because of cluster name exist", func() {
			var cjrn *unstructured.Unstructured
			BeforeEach(func() {
				// create new clusterjoinrequest for exist cluster namespace
				cjrn, err = common.LoadResourceFromJSON(template.ClusterJoinRequestTemplate)
				Ω(err).ShouldNot(HaveOccurred())

				err = unstructured.SetNestedField(cjrn.Object, namespace, "spec", "clusterNameSpace")
				Ω(err).ShouldNot(HaveOccurred())
				err = unstructured.SetNestedField(cjrn.Object, namespace+"-DuplicateNamespace", "spec", "clusterName")
				Ω(err).ShouldNot(HaveOccurred())

				err = unstructured.SetNestedField(cjrn.Object, data, "spec", "csr", "request")
				Ω(err).ShouldNot(HaveOccurred())

				cjrn, err = common.CreateClusterResource(dynamicClient, cjrGVR, cjrn)
				Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", cjrGVR.Resource)
			})
			It("should be denied by controller successfully", func() {
				Eventually(func() (string, error) {
					cjrn, err := common.GetClusterResource(dynamicClient, cjrGVR, cjrn.GetName())
					if err != nil {
						return "", err
					}

					phase, _, err := unstructured.NestedString(cjrn.Object, "status", "phase")
					return phase, err
				}, eventuallyTimeout, eventuallyInterval).Should(Equal("Denied"))
			})
			AfterEach(func() {
				// delete clusterjoinrequest created
				err = common.DeleteClusterResource(dynamicClient, cjrGVR, cjrn.GetName())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
		Context("Deny clusterjoinrequest because of cluster namespace exist", func() {
			var cjrns *unstructured.Unstructured
			BeforeEach(func() {
				// create new clusterjoinrequest for exist cluster name
				cjrns, err = common.LoadResourceFromJSON(template.ClusterJoinRequestTemplate)
				Ω(err).ShouldNot(HaveOccurred())

				err = unstructured.SetNestedField(cjrns.Object, namespace+"-DuplicateName", "spec", "clusterNameSpace")
				Ω(err).ShouldNot(HaveOccurred())
				err = unstructured.SetNestedField(cjrns.Object, namespace, "spec", "clusterName")
				Ω(err).ShouldNot(HaveOccurred())

				err = unstructured.SetNestedField(cjrns.Object, data, "spec", "csr", "request")
				Ω(err).ShouldNot(HaveOccurred())

				cjrns, err = common.CreateClusterResource(dynamicClient, cjrGVR, cjrns)
				Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", cjrGVR.Resource)
			})
			It("should be denied by controller successfully", func() {
				Eventually(func() (string, error) {
					cjrns, err := common.GetClusterResource(dynamicClient, cjrGVR, cjrns.GetName())
					if err != nil {
						return "", err
					}
					phase, _, err := unstructured.NestedString(cjrns.Object, "status", "phase")
					return phase, err
				}, eventuallyTimeout, eventuallyInterval).Should(Equal("Denied"))
			})
			AfterEach(func() {
				// delete clusterjoinrequest created
				err = common.DeleteClusterResource(dynamicClient, cjrGVR, cjrns.GetName())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())

			// delete clusterjoinrequest created
			err = common.DeleteClusterResource(dynamicClient, cjrGVR, cjr.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Deleting a cluster", func() {
		BeforeEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should be deleted successfully", func() {
			// check if the resource is deleted eventually
			Eventually(func() (bool, error) {
				return common.HasResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			}, eventuallyTimeout, eventuallyInterval).Should(BeFalse())
		})
	})

	AfterEach(func() {
		// delete the namespace created for testing
		err := common.DeleteClusterResource(dynamicClient, common.NamespaceGVR, namespace)
		Ω(err).ShouldNot(HaveOccurred())
	})
})

func setStatusType(obj *unstructured.Unstructured, statusType string) error {
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return err
	}

	if conditions == nil {
		conditions = make([]interface{}, 0)
	}

	if len(conditions) == 0 {
		conditions = append(conditions, map[string]interface{}{
			"type": statusType,
		})
		err := unstructured.SetNestedField(obj.Object, conditions, "status", "conditions")
		if err != nil {
			return err
		}
	} else {
		condition := conditions[0].(map[string]interface{})
		condition["type"] = statusType
	}

	return nil
}

func getConditionFromStatus(obj *unstructured.Unstructured) (map[string]interface{}, error) {
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

func newDynamicClientWithCertAndKey(host string, cert, key []byte) (dynamic.Interface, error) {
	config := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{"default-cluster": {
			Server:                host,
			InsecureSkipTLSVerify: true,
		}},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{"default-auth": {
			ClientCertificateData: cert,
			ClientKeyData:         key,
		}},
		Contexts: map[string]*clientcmdapi.Context{"default-context": {
			Cluster:   "default-cluster",
			AuthInfo:  "default-auth",
			Namespace: "default",
		}},
		CurrentContext: "default-context",
	}

	clientConfig, err := clientcmd.NewNonInteractiveClientConfig(config, "", &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}
