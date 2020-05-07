// +build integration

package resourceviews_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/template"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	eventuallyTimeout  = 60
	eventuallyInterval = 1
)

var gvr = schema.GroupVersionResource{
	Group:    "mcm.ibm.com",
	Version:  "v1beta1",
	Resource: "resourceviews",
}

var clusterGVR = schema.GroupVersionResource{
	Group:    "clusterregistry.k8s.io",
	Version:  "v1alpha1",
	Resource: "clusters",
}

var workGVR = schema.GroupVersionResource{
	Group:    "mcm.ibm.com",
	Version:  "v1beta1",
	Resource: "works",
}

var (
	dynamicClient      dynamic.Interface
	clusters           []*unstructured.Unstructured
	hasManagedClusters bool
	namespace          string
	cluster1           *unstructured.Unstructured
	cluster2           *unstructured.Unstructured
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

	clusters, err = common.GetReadyManagedClusters(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())

	hasManagedClusters = len(clusters) > 0

	if !hasManagedClusters {
		// create a namespace for testing
		ns, err := common.LoadResourceFromJSON(template.NamespaceTemplate)
		Ω(err).ShouldNot(HaveOccurred())
		ns, err = common.CreateClusterResource(dynamicClient, common.NamespaceGVR, ns)
		Ω(err).ShouldNot(HaveOccurred())
		namespace = ns.GetName()

		//create 2 fake clusters
		cluster1, err = common.CreateCluster(dynamicClient)
		Ω(err).ShouldNot(HaveOccurred())

		cluster2, err = common.CreateCluster(dynamicClient)
		Ω(err).ShouldNot(HaveOccurred())
		//check cluster1 ready
		Eventually(func() (interface{}, error) {
			cluster1, err := common.GetResource(dynamicClient, clusterGVR, cluster1.GetNamespace(), cluster1.GetName())
			if err != nil {
				return "", err
			}

			condition, err := common.GetConditionFromStatus(cluster1)
			if err != nil {
				return "", err
			}
			if condition == nil {
				return "", nil
			}

			return condition["type"], nil
		}, eventuallyTimeout, eventuallyInterval).Should(Equal("OK"))
		//check cluster2 ready
		Eventually(func() (interface{}, error) {
			cluster2, err := common.GetResource(dynamicClient, clusterGVR, cluster2.GetNamespace(), cluster2.GetName())
			if err != nil {
				return "", err
			}

			condition, err := common.GetConditionFromStatus(cluster2)
			if err != nil {
				return "", err
			}
			if condition == nil {
				return "", nil
			}

			return condition["type"], nil
		}, eventuallyTimeout, eventuallyInterval).Should(Equal("OK"))

		clusters, err = common.GetReadyManagedClusters(dynamicClient)
		Ω(err).ShouldNot(HaveOccurred())

	}
	// create a namespace for testing
	ns, err := common.LoadResourceFromJSON(template.NamespaceTemplate)
	Ω(err).ShouldNot(HaveOccurred())

	ns, err = common.CreateClusterResource(dynamicClient, common.NamespaceGVR, ns)
	Ω(err).ShouldNot(HaveOccurred())

	namespace = ns.GetName()
})

var _ = AfterSuite(func() {
	// delete the namespace created for testing
	err := common.DeleteClusterResource(dynamicClient, common.NamespaceGVR, namespace)
	Ω(err).ShouldNot(HaveOccurred())
	if !hasManagedClusters {
		// delete the resource created
		err = common.DeleteResource(dynamicClient, clusterGVR, cluster1.GetNamespace(), cluster1.GetName())
		Ω(err).ShouldNot(HaveOccurred())
		err = common.DeleteResource(dynamicClient, clusterGVR, cluster2.GetNamespace(), cluster2.GetName())
		Ω(err).ShouldNot(HaveOccurred())
	}
})

var _ = Describe("Resourceviews", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	BeforeEach(func() {
		// load object from json template
		obj, err = common.LoadResourceFromJSON(template.ResourceViewTemplate)
		Ω(err).ShouldNot(HaveOccurred())

		// setup resourceview
		err = unstructured.SetNestedField(obj.Object, namespace, "metadata", "namespace")
		Ω(err).ShouldNot(HaveOccurred())

		// create a resource on api-server
		obj, err = common.CreateResource(dynamicClient, gvr, obj)
		Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
	})

	Describe("Creating a resourceview", func() {
		It("should be created successfully", func() {
			exists, err := common.HasResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should be handled by controller successfully", func() {
			// check if controller create works correctly
			labelSelector := fmt.Sprintf("mcm.ibm.com/resourceview=%s.%s", obj.GetNamespace(), obj.GetName())
			Eventually(func() (int, error) {
				works, err := common.ListResource(dynamicClient, workGVR, "", labelSelector)
				if err != nil {
					return -1, err
				}

				return len(works), err
			}, eventuallyTimeout, eventuallyInterval).Should(Equal(len(clusters)))
		})

		It("should be executed on managed cluster successfully", func() {
			if !hasManagedClusters {
				labelSelector := fmt.Sprintf("mcm.ibm.com/resourceview=%s.%s", obj.GetNamespace(), obj.GetName())
				time.Sleep(time.Second * 2)
				works, err := common.ListResource(dynamicClient, workGVR, "", labelSelector)
				Ω(err).ShouldNot(HaveOccurred())

				_, err = common.UpdateWorkStatus(dynamicClient, works[0], "Completed")
				Ω(err).ShouldNot(HaveOccurred())
				_, err = common.UpdateWorkStatus(dynamicClient, works[1], "Completed")
				Ω(err).ShouldNot(HaveOccurred())

			}

			// check the results in status of resourceview
			Eventually(func() (interface{}, error) {
				resourceview, err := common.GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
				if err != nil {
					return "", err
				}

				// check the resourceview status
				condition, err := common.GetConditionFromStatus(resourceview)
				if err != nil {
					return "", err
				}
				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("Completed"))
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Deleting a resourceview", func() {
		BeforeEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should be deleted successfully", func() {
			// check if the resource is deleted eventually
			Eventually(func() (bool, error) {
				exist, err := common.HasResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
				if err != nil || exist {
					return exist, err
				}

				return exist, err
			}, eventuallyTimeout, eventuallyInterval).Should(BeFalse())
		})
	})
})
