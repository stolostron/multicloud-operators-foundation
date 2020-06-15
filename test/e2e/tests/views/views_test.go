// +build integration

package views_test

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
	eventuallyInterval = 2
)

var viewGVR = schema.GroupVersionResource{
	Group:    "view.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterviews",
}

var (
	dynamicClient dynamic.Interface
	realCluster   *unstructured.Unstructured
	fakeCluster   *unstructured.Unstructured
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

	realClusters, err := common.GetJoinedManagedClusters(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(len(realClusters)).ShouldNot(Equal(0))

	realCluster = realClusters[0]

	// create a fake cluster
	fakeCluster, err = common.CreateManagedCluster(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	obj, err := common.LoadResourceFromJSON(template.ManagedClusterViewTemplate)
	Ω(err).ShouldNot(HaveOccurred())
	err = common.DeleteResource(dynamicClient, viewGVR, realCluster.GetName(), obj.GetName())
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = Describe("Testing managedClusterView if agent is ok", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	Context("Creating a managedClusterView", func() {
		It("Should create successfully", func() {
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterViewTemplate)
			Ω(err).ShouldNot(HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create managedClusterView to real cluster
			obj, err = common.CreateResource(dynamicClient, viewGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", viewGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, viewGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterView, err := common.GetResource(dynamicClient, viewGVR, realCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the managedClusterView status
				condition, err := common.GetConditionFromStatus(managedClusterView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("Processing"))
		})
	})
})

var _ = Describe("Testing managedClusterView if agent is lost", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	Context("Creating a managedClusterView", func() {
		It("Should create successfully", func() {
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterViewTemplate)
			Ω(err).ShouldNot(HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, fakeCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create managedClusterView to real cluster
			obj, err = common.CreateResource(dynamicClient, viewGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", viewGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, viewGVR, fakeCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterView, err := common.GetResource(dynamicClient, viewGVR, fakeCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the managedClusterView status
				condition, err := common.GetConditionFromStatus(managedClusterView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal(""))
		})
	})
})
