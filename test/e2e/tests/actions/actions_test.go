// +build integration

package actions_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
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
	Resource: "managedclusteractions",
}

var (
	dynamicClient             dynamic.Interface
	realClusters              []*unstructured.Unstructured
	fakeManagedClusterActions []*unstructured.Unstructured
	realManagedClusterActions []*unstructured.Unstructured
	realCluster               *unstructured.Unstructured
	fakeCluster               *unstructured.Unstructured
	actionTemplates           []string
	hasManagedClusters        bool
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

	realClusters, err = common.GetJoinedManagedClusters(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
	hasManagedClusters = len(realClusters) > 0

	// create a fake cluster
	fakeCluster, err = common.CreateManagedCluster(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
	// check fakeCluster ready
	Eventually(func() (interface{}, error) {
		fakeCluster, err := common.GetClusterResource(dynamicClient, common.ManagedClusterGVR, fakeCluster.GetName())
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
	}, eventuallyTimeout, eventuallyInterval).Should(Equal(clusterv1.ManagedClusterConditionJoined))
})

var _ = AfterSuite(func() {
	// delete the namespace created for testing
	err := common.DeleteClusterResource(dynamicClient, common.NamespaceGVR, fakeCluster.GetName())
	Ω(err).ShouldNot(HaveOccurred())

	// delete the resource created
	err = common.DeleteClusterResource(dynamicClient, common.ManagedClusterGVR, fakeCluster.GetName())
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = Describe("Testing ManagedClusterAction", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	BeforeEach(func() {

		actionTemplates = []string{
			template.ManagedClusterActionCreateTemplate,
			template.ManagedClusterActionDeleteTemplate,
			template.ManagedClusterActionUpdateTemplate,
		}

		for _, actionTemplate := range actionTemplates {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(actionTemplate)
			Ω(err).ShouldNot(HaveOccurred())
			err = unstructured.SetNestedField(obj.Object, fakeCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to fake cluster
			obj, err = common.CreateResource(dynamicClient, gvr, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
			// record obj for delete obj
			fakeManagedClusterActions = append(fakeManagedClusterActions, obj)
			// create ManagedClusterAction to real cluster
			if hasManagedClusters {
				// load object from json template
				obj, err := common.LoadResourceFromJSON(actionTemplate)
				Ω(err).ShouldNot(HaveOccurred())
				realCluster = realClusters[0]
				err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
				Ω(err).ShouldNot(HaveOccurred())
				// create ManagedClusterAction to real cluster
				obj, err = common.CreateResource(dynamicClient, gvr, obj)
				Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", gvr.Resource)
				// record obj for delete obj
				realManagedClusterActions = append(realManagedClusterActions, obj)
			}
		}
	})

	Describe("Creating a ManagedClusterAction", func() {
		It("should be created successfully in cluster", func() {
			for _, obj := range fakeManagedClusterActions {
				exists, err := common.HasResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
				Ω(exists).Should(BeTrue())
			}

			if hasManagedClusters {
				for _, obj := range realManagedClusterActions {
					exists, err := common.HasResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
					Ω(err).ShouldNot(HaveOccurred())
					Ω(exists).Should(BeTrue())
				}
			}
		})

		It("should have a valid condition", func() {
			for _, obj := range fakeManagedClusterActions {
				// In fake cluster, the status of ManagedClusterAction should be empty
				Eventually(func() (interface{}, error) {
					ManagedClusterAction, err := common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					// check the ManagedClusterAction status
					condition, err := common.GetConditionFromStatus(ManagedClusterAction)
					if err != nil {
						return "", err
					}

					if condition == nil {
						return "", nil
					}

					return condition["type"], nil
				}, eventuallyTimeout, eventuallyInterval).Should(Equal(""))
			}
			// In real cluster, the status of ManagedClusterAction should have a valid value
			if hasManagedClusters {
				for _, obj := range realManagedClusterActions {
					Eventually(func() (interface{}, error) {
						ManagedClusterAction, err := common.GetResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
						if err != nil {
							return "", err
						}
						// check the ManagedClusterAction status
						condition, err := common.GetConditionFromStatus(ManagedClusterAction)
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
			for _, obj := range fakeManagedClusterActions {
				Eventually(func() (interface{}, error) {
					managedClusterAction, err := common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					err = common.SetStatusType(managedClusterAction, "Completed")
					Ω(err).ShouldNot(HaveOccurred())
					managedClusterAction, err = common.UpdateResourceStatus(dynamicClient, gvr, managedClusterAction)
					Ω(err).ShouldNot(HaveOccurred())
					managedClusterAction, err = common.GetResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
					if err != nil {
						return "", err
					}
					condition, err := common.GetConditionFromStatus(managedClusterAction)
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
			for _, obj := range fakeManagedClusterActions {
				err = common.DeleteResource(dynamicClient, gvr, fakeCluster.GetName(), obj.GetName())
				Ω(err).ShouldNot(HaveOccurred())
			}

			if hasManagedClusters {
				for _, obj := range realManagedClusterActions {
					err = common.DeleteResource(dynamicClient, gvr, realCluster.GetName(), obj.GetName())
					Ω(err).ShouldNot(HaveOccurred())
				}
			}
		})
	})
})
