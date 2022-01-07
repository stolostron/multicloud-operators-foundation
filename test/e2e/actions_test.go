package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	actionDeploymentName      = "nginx-deployment-action"
	actionDeploymentNameSpace = "default"
)

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

var _ = ginkgo.Describe("Testing ManagedClusterAction when Agent is ok", func() {
	var (
		obj *unstructured.Unstructured
		err error
	)

	ginkgo.Context("Creating a UpdateManagedClusterAction when resource do not exist", func() {
		ginkgo.It("Should create updateManagedClusterAction successfully", func() {
			// load object from json util
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterActionUpdateTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = util.CreateResource(dynamicClient, actionGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		ginkgo.It("should get successfully", func() {
			exists, err := util.HasResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())
		})

		ginkgo.It("should have a valid condition", func() {
			gomega.Eventually(func() (interface{}, error) {
				managedClusterAction, err := util.GetResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := util.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("False"))
		})

		ginkgo.It("should delete successfully", func() {
			err := util.DeleteResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Creating a DeleteManagedClusterAction when resource do not exist", func() {
		ginkgo.It("Should create deleteManagedCusterAction successfully", func() {
			// load object from json util
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterActionDeleteTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = util.CreateResource(dynamicClient, actionGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		ginkgo.It("should get successfully", func() {
			exists, err := util.HasResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())
		})

		ginkgo.It("should have a valid condition", func() {
			gomega.Eventually(func() (interface{}, error) {
				managedClusterAction, err := util.GetResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := util.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("False"))
		})

		ginkgo.It("deployment should be deleted successfully in managedcluster", func() {
			gomega.Eventually(func() (interface{}, error) {
				return util.HasResource(dynamicClient, depGVR, actionDeploymentNameSpace, actionDeploymentName)
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.BeTrue())
		})

		ginkgo.It("should delete successfully", func() {
			err := util.DeleteResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Creating a CreateManagedClusterAction", func() {
		ginkgo.It("should create successfully", func() {
			// load object from json util
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterActionCreateTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = util.CreateResource(dynamicClient, actionGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		ginkgo.It("should get successfully", func() {
			exists, err := util.HasResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())
		})

		ginkgo.It("should have a valid condition", func() {
			gomega.Eventually(func() (interface{}, error) {
				managedClusterAction, err := util.GetResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := util.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("True"))
		})

		ginkgo.It("deployment should be created successfully in managedcluster", func() {
			gomega.Eventually(func() (interface{}, error) {
				return util.HasResource(dynamicClient, depGVR, actionDeploymentNameSpace, actionDeploymentName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("should delete successfully", func() {
			err := util.DeleteResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Creating a UpdateManagedClusterAction", func() {
		ginkgo.It("Should create udateManagedClusterAction successfully", func() {
			// load object from json util
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterActionUpdateTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = util.CreateResource(dynamicClient, actionGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		ginkgo.It("should get successfully", func() {
			exists, err := util.HasResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())
		})

		ginkgo.It("should have a valid condition", func() {
			gomega.Eventually(func() (interface{}, error) {
				managedClusterAction, err := util.GetResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := util.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("True"))
		})

		ginkgo.It("should delete successfully", func() {
			err := util.DeleteResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Creating a DeleteManagedClusterAction", func() {
		ginkgo.It("Should create deleteManagedClusterAction successfully", func() {
			// load object from json util
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterActionDeleteTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, defaultManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// create ManagedClusterAction to real cluster
			obj, err = util.CreateResource(dynamicClient, actionGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", actionGVR.Resource)
		})

		ginkgo.It("should get successfully", func() {
			exists, err := util.HasResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())
		})

		ginkgo.It("should have a valid condition", func() {
			gomega.Eventually(func() (interface{}, error) {
				managedClusterAction, err := util.GetResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterAction status
				condition, err := util.GetConditionFromStatus(managedClusterAction)
				if err != nil {
					return "", err
				}

				if condition == nil {
					return "", nil
				}

				return condition["status"], nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal("True"))
		})

		ginkgo.It("deployment should be deleted successfully in managedcluster", func() {
			gomega.Eventually(func() (interface{}, error) {
				return util.HasResource(dynamicClient, depGVR, actionDeploymentNameSpace, actionDeploymentName)
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.BeTrue())
		})

		ginkgo.It("should delete successfully", func() {
			err := util.DeleteResource(dynamicClient, actionGVR, defaultManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

})

var _ = ginkgo.Describe("Testing ManagedClusterAction when agent is lost", func() {
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

	ginkgo.Context("Creating a ManagedClusterAction", func() {
		ginkgo.It("Should create successfully", func() {
			ginkgo.By("create ManagedClusterAction to fake cluster")
			obj, err = util.LoadResourceFromJSON(util.ManagedClusterActionCreateTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			err = unstructured.SetNestedField(obj.Object, lostManagedCluster, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			obj, err = util.CreateResource(dynamicClient, actionGVR, obj)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", actionGVR.Resource)

			ginkgo.By("should get successfully")
			exists, err := util.HasResource(dynamicClient, actionGVR, lostManagedCluster, obj.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Expect(exists).Should(gomega.BeTrue())

			ginkgo.By("should not have a valid condition")
			gomega.Eventually(func() (bool, error) {
				ManagedClusterAction, err := util.GetResource(dynamicClient, actionGVR, lostManagedCluster, obj.GetName())
				if err != nil {
					return false, err
				}
				// check the ManagedClusterAction status
				condition, err := util.GetConditionFromStatus(ManagedClusterAction)
				if err != nil {
					return false, err
				}

				if condition == nil {
					return true, nil
				}

				return false, nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(true))
		})
	})
})
