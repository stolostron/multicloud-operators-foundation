// +build integration
package works_test

import (
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

	// create a namespace for testing if no managed cluster found
	if !hasManagedClusters {
		ns, err := common.LoadResourceFromJSON(template.NamespaceTemplate)
		Ω(err).ShouldNot(HaveOccurred())

		ns, err = common.CreateClusterResource(dynamicClient, common.NamespaceGVR, ns)
		Ω(err).ShouldNot(HaveOccurred())

		namespace = ns.GetName()
	}
})

var _ = AfterSuite(func() {
	// delete the namespace created for testing if necessary
	if namespace != "" {
		err := common.DeleteClusterResource(dynamicClient, common.NamespaceGVR, namespace)
		Ω(err).ShouldNot(HaveOccurred())
	}
})

var _ = Describe("Works", func() {
	var (
		obj          *unstructured.Unstructured
		jsonTemplate string
		err          error
	)

	JustBeforeEach(func() {
		// load object from json template
		obj, err = common.LoadResourceFromJSON(jsonTemplate)
		Ω(err).ShouldNot(HaveOccurred())

		// setup work
		if hasManagedClusters {
			obj.SetNamespace(clusters[0].GetNamespace())
			unstructured.SetNestedField(obj.Object, clusters[0].GetName(), "spec", "cluster", "name")
		} else {
			err = unstructured.SetNestedField(obj.Object, namespace, "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
		}

		// create a resource on api-server
		obj, err = common.CreateResource(dynamicClient, gvr, obj)
		Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
	})

	Describe("Creating a work", func() {
		createSpecsForWorks := func() {
			It("should be created successfully", func() {
				exists, err := common.HasResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
				Ω(exists).Should(BeTrue())
			})

			It("should be executed on managed cluster successfully", func() {
				if !hasManagedClusters {
					Skip("No managed cluster found")
				}

				Eventually(func() (string, error) {
					work, err := common.GetResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
					if err != nil {
						return "", err
					}

					// check work status
					_, ok, err := unstructured.NestedString(work.Object, "status", "lastUpdateTime")
					if err != nil {
						return "", err
					}
					if !ok {
						return "", nil
					}

					_, ok, err = unstructured.NestedMap(work.Object, "status", "result")
					if err != nil {
						return "", err
					}
					if !ok {
						return "", nil
					}

					t, _, err := unstructured.NestedString(work.Object, "status", "type")
					return t, err
				}, eventuallyTimeout, eventuallyInterval).Should(Equal("Completed"))
			})
		}

		Context("When it's an action work", func() {
			BeforeEach(func() {
				jsonTemplate = template.ActionWorkTemplate
			})

			createSpecsForWorks()
		})

		Context("When it's a resource work", func() {
			BeforeEach(func() {
				jsonTemplate = template.ResourceWorkTemplate
			})

			createSpecsForWorks()
		})

		AfterEach(func() {
			// delete the resource created
			err = common.DeleteResource(dynamicClient, gvr, obj.GetNamespace(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Deleting a work", func() {
		BeforeEach(func() {
			jsonTemplate = template.ResourceWorkTemplate
		})

		JustBeforeEach(func() {
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
