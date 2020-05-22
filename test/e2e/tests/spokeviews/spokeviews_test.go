// +build integration

package spokeviews_test

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
	Group:    "view.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "spokeviews",
}

var (
	dynamicClient    dynamic.Interface
	realClusters     []*unstructured.Unstructured
	realCluster      *unstructured.Unstructured
	fakeCluster      *unstructured.Unstructured
	hasSpokeClusters bool
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

	realClusters, err = common.GetJoinedSpokeClusters(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
	hasSpokeClusters = len(realClusters) > 0

	// create a fake cluster
	fakeCluster, err = common.CreateSpokeCluster(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
	// check fakeCluster ready
	Eventually(func() (interface{}, error) {
		fakeCluster, err := common.GetClusterResource(dynamicClient, common.SpokeClusterGVR, fakeCluster.GetName())
		if err != nil {
			return "", err
		}

		condition, err := common.GetConditionFromStatus(fakeCluster)
		if err != nil {
			return "", err
		}
		if condition == nil {
			return "", nil
		}

		return condition["type"], nil
	}, eventuallyTimeout, eventuallyInterval).Should(Equal("SpokeClusterJoined"))
})

var _ = AfterSuite(func() {
	// delete the namespace created for testing
	err := common.DeleteClusterResource(dynamicClient, common.NamespaceGVR, fakeCluster.GetName())
	Ω(err).ShouldNot(HaveOccurred())

	// delete the resource created
	err = common.DeleteClusterResource(dynamicClient, common.SpokeClusterGVR, fakeCluster.GetName())
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = Describe("Testing spokeView", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	BeforeEach(func() {
		// load object from json template
		obj, err = common.LoadResourceFromJSON(template.SpokeViewTemplate)
		Ω(err).ShouldNot(HaveOccurred())
		err = unstructured.SetNestedField(obj.Object, fakeCluster.GetName(), "metadata", "namespace")
		Ω(err).ShouldNot(HaveOccurred())
		// create spokeView to fake cluster
		obj, err = common.CreateResource(dynamicClient, gvr, obj)
		Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
		// create spokeView to real cluster
		if hasSpokeClusters {
			// load object from json template
			obj, err := common.LoadResourceFromJSON(template.SpokeViewTemplate)
			Ω(err).ShouldNot(HaveOccurred())
			realCluster = realClusters[0]
			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create spokeView to real cluster
			obj, err = common.CreateResource(dynamicClient, gvr, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
		}
	})

	Describe("Creating a spokeView", func() {
		It("should be created successfully in cluster", func() {
			exists, err := common.HasResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())

			if hasSpokeClusters {
				exists, err := common.HasResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
				Ω(exists).Should(BeTrue())
			}
		})

		It("should have a valid condition", func() {
			// In fake cluster, the status of spokeView should be empty
			Eventually(func() (interface{}, error) {
				spokeView, err := common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the spokeView status
				condition, err := common.GetConditionFromStatus(spokeView)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal(""))

			// In real cluster, the status of spokeView should have a valid value
			if hasSpokeClusters {
				Eventually(func() (interface{}, error) {
					spokeView, err := common.GetResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					// check the spokeView status
					condition, err := common.GetConditionFromStatus(spokeView)
					if err != nil {
						return "", err
					}

					if condition == nil {
						return "", nil
					}

					return condition["type"], nil
				}, eventuallyTimeout, eventuallyInterval).Should(Equal("Processing"))
			}
		})

		It("should be updated successfully in cluster", func() {
			Eventually(func() (interface{}, error) {
				spokeView, err := common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				err = common.SetStatusType(spokeView, "Processing")
				Ω(err).ShouldNot(HaveOccurred())
				err = unstructured.SetNestedField(spokeView.Object, spokeView.GetResourceVersion(), "metadata", "resourceVersion")
				Ω(err).ShouldNot(HaveOccurred())
				spokeView, err = common.UpdateResourceStatus(dynamicClient, gvr, spokeView)
				Ω(err).ShouldNot(HaveOccurred())
				spokeView, err = common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				condition, err := common.GetConditionFromStatus(spokeView)
				if err != nil {
					return "", err
				}
				if condition == nil {
					return "", nil
				}

				return condition["type"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("Processing"))
		})

		AfterEach(func() {
			// delete all resource from fake cluster and real cluster
			err = common.DeleteResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())

			if hasSpokeClusters {
				err = common.DeleteResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
			}
		})
	})
})
