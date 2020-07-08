// +build integration

package actions_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/template"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	eventuallyTimeout         = 60
	eventuallyInterval        = 2
	actionDeploymentName      = "nginx-deployment-action"
	actionDeploymentNameSpace = "default"
)

var singleManagedOnHub = true

var actionGVR = schema.GroupVersionResource{
	Group:    "action.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusteractions",
}
var depGVR = schema.GroupVersionResource{
	Group:    "apps",
	Version:  "v1",
	Resource: "deployments",
}

var (
	dynamicClient dynamic.Interface
	realCluster   *unstructured.Unstructured
	fakeCluster   *unstructured.Unstructured
)

var _ = BeforeSuite(func() {
	var err error
	managedOnHub := os.Getenv(common.SingleManagedOnHub)
	if strings.ToLower(managedOnHub) == "false" {
		singleManagedOnHub = false
	}
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
	//delete fake cluster
	err := common.DeleteClusterResource(dynamicClient, common.ManagedClusterGVR, fakeCluster.GetName())
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = Describe("Testing ManagedClusterAction when Agent is ok", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)
	Context("Creating a UpdateManagedClusterAction when resource do not exist", func() {
		It("Should create update managedclusteraction successfully", func() {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterActionUpdateTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = common.CreateResource(dynamicClient, actionGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterAction, err := common.GetResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := common.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("False"))
		})

		It("should delete successfully", func() {
			err := common.DeleteResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Creating a DeleteManagedClusterAction when resource do not exist", func() {
		It("Should create update managedclusteraction successfully", func() {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterActionDeleteTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = common.CreateResource(dynamicClient, actionGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterAction, err := common.GetResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := common.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("False"))
		})

		It("deployment should be deleted successfully in managedcluster", func() {
			if singleManagedOnHub {
				Eventually(func() (interface{}, error) {
					return common.HasResource(dynamicClient, depGVR, actionDeploymentNameSpace, actionDeploymentName)
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(BeTrue())
			}
		})

		It("should delete successfully", func() {
			err := common.DeleteResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
	Context("Creating a CreateManagedClusterAction", func() {
		It("should create successfully", func() {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterActionCreateTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = common.CreateResource(dynamicClient, actionGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterAction, err := common.GetResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := common.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("True"))
		})
		It("deployment should be created successfully in managedcluster", func() {
			if singleManagedOnHub {
				Eventually(func() (interface{}, error) {
					return common.HasResource(dynamicClient, depGVR, actionDeploymentNameSpace, actionDeploymentName)
				}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

			}
		})

		It("should delete successfully", func() {
			err := common.DeleteResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Creating a UpdateManagedClusterAction", func() {
		It("Should create update managedclusteraction successfully", func() {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterActionUpdateTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = common.CreateResource(dynamicClient, actionGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterAction, err := common.GetResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := common.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("True"))
		})

		It("should delete successfully", func() {
			err := common.DeleteResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Creating a DeleteManagedClusterAction", func() {
		It("Should create update managedclusteraction successfully", func() {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterActionDeleteTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, realCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = common.CreateResource(dynamicClient, actionGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				managedClusterAction, err := common.GetResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := common.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal("True"))
		})

		It("deployment should be deleted successfully in managedcluster", func() {

			if singleManagedOnHub {
				Eventually(func() (interface{}, error) {
					return common.HasResource(dynamicClient, depGVR, actionDeploymentNameSpace, actionDeploymentName)
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(BeTrue())
			}
		})

		It("should delete successfully", func() {
			err := common.DeleteResource(dynamicClient, actionGVR, realCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

})

var _ = Describe("Testing ManagedClusterAction when agent is lost", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	Context("Creating a ManagedClusterAction", func() {
		It("Should create successfully", func() {
			// load object from json template
			obj, err = common.LoadResourceFromJSON(template.ManagedClusterActionCreateTemplate)
			Ω(err).ShouldNot(HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, fakeCluster.GetName(), "metadata", "namespace")
			Ω(err).ShouldNot(HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = common.CreateResource(dynamicClient, actionGVR, obj)
			Ω(err).ShouldNot(HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		It("should get successfully", func() {
			exists, err := common.HasResource(dynamicClient, actionGVR, fakeCluster.GetName(), obj.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should not have a valid condition", func() {
			Eventually(func() (bool, error) {
				ManagedClusterAction, err := common.GetResource(dynamicClient, actionGVR, fakeCluster.GetName(), obj.GetName())
				if err != nil {
					return false, err
				}
				// check the ManagedClusterAction status
				condition, err := common.GetConditionFromStatus(ManagedClusterAction)
				if err != nil {
					return false, err
				}

				if condition == nil {
					return true, nil
				}

				return false, nil
			}, eventuallyTimeout, eventuallyInterval).Should(Equal(true))
		})
	})
})
