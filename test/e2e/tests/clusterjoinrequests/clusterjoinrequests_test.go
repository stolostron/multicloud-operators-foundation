// +build integration

package clusterjoinrequests_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/template"

	. "github.com/onsi/gomega"
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
	Resource: "clusterjoinrequests",
}

var (
	dynamicClient dynamic.Interface
	namespace     string
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

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

var _ = Describe("Clusterjoinrequests", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	BeforeEach(func() {
		// load object from json template
		obj, err = common.LoadResourceFromJSON(template.ClusterJoinRequestTemplate)
		Ω(err).ShouldNot(HaveOccurred())

		// setup clusterjoinrequest
		err = unstructured.SetNestedField(obj.Object, namespace, "spec", "clusterNameSpace")
		Ω(err).ShouldNot(HaveOccurred())
		err = unstructured.SetNestedField(obj.Object, namespace, "spec", "clusterName")
		Ω(err).ShouldNot(HaveOccurred())

		data, err := common.GenerateCSR(namespace, namespace, nil)
		Ω(err).ShouldNot(HaveOccurred())

		err = unstructured.SetNestedField(obj.Object, data, "spec", "csr", "request")
		Ω(err).ShouldNot(HaveOccurred())

		// create a resource on api-server
		obj, err = common.CreateClusterResource(dynamicClient, gvr, obj)
		Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
	})

	Describe("Creating a clusterjoinrequest", func() {
		It("should be created successfully", func() {
			exists, err := common.HasClusterResource(dynamicClient, gvr, obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should be approved by controller successfully", func() {
			Eventually(func() (string, error) {
				cjr, err := common.GetClusterResource(dynamicClient, gvr, obj.GetName())
				if err != nil {
					return "", err
				}

				phase, _, err := unstructured.NestedString(cjr.Object, "status", "phase")
				return phase, err
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("Approved"))
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteClusterResource(dynamicClient, gvr, obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Deleting a clusterjoinrequest", func() {
		BeforeEach(func() {
			// delete the resource created
			err = common.DeleteClusterResource(dynamicClient, gvr, obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should be deleted successfully", func() {
			// check if the resource is deleted eventually
			Eventually(func() (bool, error) {
				return common.HasClusterResource(dynamicClient, gvr, obj.GetName())
			}, eventuallyTimeout, eventuallyInterval).Should(BeFalse())
		})
	})
})
