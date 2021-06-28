package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var viewGVR = schema.GroupVersionResource{
	Group:    "view.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterviews",
}

var _ = ginkgo.Describe("Testing ManagedClusterView if agent is ok", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	ginkgo.Context("Creating a managedClusterView", func() {
		ginkgo.It("Should create successfully", func() {
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterViewTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create managedClusterView to real cluster
			obj, err = util.CreateResource(dynamicClient, viewGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", viewGVR.Resource)

			ginkgo.By("should get successfully")
			exists, err := util.HasResource(dynamicClient, viewGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())

			ginkgo.By("should have a valid condition")
			gomega.Eventually(func() (interface{}, error) {
				managedClusterView, err := util.GetResource(dynamicClient, viewGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the managedClusterView status
				condition, err := util.GetConditionFromStatus(managedClusterView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("Processing"))
		})
	})

})

var _ = ginkgo.Describe("Testing ManagedClusterView if agent is lost", func() {
	var (
		lostManagedCluster = util.RandomName()
		obj                *unstructured.Unstructured
		err                error
	)

	ginkgo.BeforeEach(func() {
		err = util.ImportManagedCluster(clusterClient, lostManagedCluster)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		err = util.CleanManagedCluster(clusterClient, lostManagedCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Creating a managedClusterView", func() {
		ginkgo.It("Should create successfully", func() {
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterViewTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, lostManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			// create managedClusterView to real cluster
			obj, err = util.CreateResource(dynamicClient, viewGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", viewGVR.Resource)

			ginkgo.By("should get successfully")
			exists, err := util.HasResource(dynamicClient, viewGVR, lostManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())

			ginkgo.By("should have a valid condition")
			gomega.Eventually(func() (interface{}, error) {
				managedClusterView, err := util.GetResource(dynamicClient, viewGVR, lostManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the managedClusterView status
				condition, err := util.GetConditionFromStatus(managedClusterView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(""))
		})
	})
})
