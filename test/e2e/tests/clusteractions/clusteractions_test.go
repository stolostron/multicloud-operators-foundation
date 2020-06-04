// +build integration

package clusteractions_test

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
	Group:    "action.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "clusteractions",
}

var (
	dynamicClient      dynamic.Interface
	realClusters       []*unstructured.Unstructured
	fakeClusterActions []*unstructured.Unstructured
	realClusterActions []*unstructured.Unstructured
	realCluster        *unstructured.Unstructured
	fakeCluster        *unstructured.Unstructured
	actionTemplates    []string
	hasSpokeClusters   bool
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

var _ = Describe("Testing clusterAction", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	BeforeEach(func() {

		actionTemplates = []string{
			template.ClusterActionCreateTemplate,
			template.ClusterActionDeleteTemplate,
			template.ClusterActionUpdateTemplate,
		}

		for _, actionTemplate := range actionTemplates {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(actionTemplate)
			Ω(err).ShouldNot(HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, fakeCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create clusterAction to fake cluster
			obj, err = common.CreateResource(dynamicClient, gvr, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
			// record obj for delete obj
			fakeClusterActions = append(fakeClusterActions, obj)
			// create clusterAction to real cluster
			if hasSpokeClusters {
				// load object from json template
				obj, err := common.LoadResourceFromJSON(actionTemplate)
				Ω(err).ShouldNot(HaveOccurred())
				realCluster = realClusters[0]
				err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
				Ω(err).ShouldNot(HaveOccurred())
				// create clusterAction to real cluster
				obj, err = common.CreateResource(dynamicClient, gvr, obj)
				Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
				// record obj for delete obj
				realClusterActions = append(realClusterActions, obj)
			}
		}
	})

	Describe("Creating a clusterAction", func() {
		It("should be created successfully in cluster", func() {
			for _, obj := range fakeClusterActions {
				exists, err := common.HasResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
				Ω(exists).Should(BeTrue())
			}

			if hasSpokeClusters {
				for _, obj := range realClusterActions {
					exists, err := common.HasResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
					Ω(err).ShouldNot(HaveOccurred())
					Ω(exists).Should(BeTrue())
				}
			}
		})

		It("should have a valid condition", func() {
			for _, obj := range fakeClusterActions {
				// In fake cluster, the status of clusterAction should be empty
				Eventually(func() (interface{}, error) {
					clusterAction, err := common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					// check the clusterAction status
					condition, err := common.GetConditionFromStatus(clusterAction)
					if err != nil {
						return "", err
					}

					if condition == nil {
						return "", nil
					}

					return condition["type"], nil
				}, eventuallyTimeout, eventuallyInterval).Should(Equal(""))
			}
			// In real cluster, the status of clusterAction should have a valid value
			if hasSpokeClusters {
				for _, obj := range realClusterActions {
					Eventually(func() (interface{}, error) {
						clusterAction, err := common.GetResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
						if err != nil {
							return "", err
						}
						// check the clusterAction status
						condition, err := common.GetConditionFromStatus(clusterAction)
						if err != nil {
							return "", err
						}

						if condition == nil {
							return "", nil
						}

						return condition["type"], nil
					}, eventuallyTimeout, eventuallyInterval).Should(Equal("Completed"))
				}
			}
		})

		It("should be updated successfully in cluster", func() {
			for _, obj := range fakeClusterActions {
				Eventually(func() (interface{}, error) {
					clusterAction, err := common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					err = common.SetStatusType(clusterAction, "Completed")
					Ω(err).ShouldNot(HaveOccurred())
					clusterAction, err = common.UpdateResourceStatus(dynamicClient, gvr, clusterAction)
					Ω(err).ShouldNot(HaveOccurred())
					clusterAction, err = common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					condition, err := common.GetConditionFromStatus(clusterAction)
					if err != nil {
						return "", err
					}
					if condition == nil {
						return "", nil
					}

					return condition["type"], nil
				}, eventuallyTimeout, eventuallyInterval).Should(Equal("Completed"))
			}
		})

		AfterEach(func() {
			// delete all resource from fake cluster and real cluster
			for _, obj := range fakeClusterActions {
				err = common.DeleteResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
			}

			if hasSpokeClusters {
				for _, obj := range realClusterActions {
					err = common.DeleteResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
					Ω(err).ShouldNot(HaveOccurred())
				}
			}
		})
	})
})
