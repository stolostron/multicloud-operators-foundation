// +build integration
package worksets_test

import (
	"fmt"

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
	Resource: "worksets",
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
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

	clusters, err = common.GetReadyManagedClusters(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())

	hasManagedClusters = len(clusters) > 0

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
})

var _ = Describe("Worksets", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	BeforeEach(func() {
		// load object from json template
		obj, err = common.LoadResourceFromJSON(template.WorksetsTemplate)
		Ω(err).ShouldNot(HaveOccurred())

		// setup workset
		err = unstructured.SetNestedField(obj.Object, namespace, "metadata", "namespace")
		Ω(err).ShouldNot(HaveOccurred())

		// create a resource on api-server
		obj, err = common.CreateResource(dynamicClient, gvr, obj)
		Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
	})

	Describe("Creating a workset", func() {
		It("should be created successfully", func() {
			exists, err := common.HasResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should be handled by controller successfully", func() {
			// check if controller create works correctly
			labelSelector := fmt.Sprintf("mcm.ibm.com/workset=%s.%s", obj.GetNamespace(), obj.GetName())
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
				Skip("No managed cluster found")
			}

			Eventually(func() (string, error) {
				workset, err := common.GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
				if err != nil {
					return "", err
				}

				// check the workset status
				status, _, err := unstructured.NestedString(workset.Object, "status", "status")
				return status, err
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("Completed"))
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Deleting a workset", func() {
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
})
