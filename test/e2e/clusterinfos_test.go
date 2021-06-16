package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var clusterInfoGVR = schema.GroupVersionResource{
	Group:    "internal.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterinfos",
}

var _ = ginkgo.Describe("Testing ManagedClusterInfo", func() {
	ginkgo.Context("Get ManagedClusterInfo", func() {
		ginkgo.It("should get a ManagedClusterInfo successfully in cluster", func() {
			gomega.Eventually(func() bool {
				exists, err := util.HasResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
				if err != nil {
					return false
				}
				return exists
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("should have node list reported successfully in cluster", func() {
			gomega.Eventually(func() error {
				managedClusterInfo, err := util.GetResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
				if err != nil {
					return err
				}
				// check the ManagedClusterInfo status
				return util.CheckNodeList(managedClusterInfo)
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("should have a valid condition", func() {
			gomega.Eventually(func() bool {
				managedClusterInfo, err := util.GetResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
				if err != nil {
					return false
				}
				// check the ManagedClusterInfo status
				return util.GetConditionTypeFromStatus(managedClusterInfo, clusterinfov1beta1.ManagedClusterInfoSynced)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("should have valid distributionInfo", func() {
			gomega.Eventually(func() error {
				managedClusterInfo, err := util.GetResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
				if err != nil {
					return err
				}
				// check the distributionInfo
				return util.CheckDistributionInfo(managedClusterInfo)
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("should have valid ClusterID", func() {
			gomega.Eventually(func() error {
				managedClusterInfo, err := util.GetResource(dynamicClient, clusterInfoGVR, managedClusterName, managedClusterName)
				if err != nil {
					return err
				}
				// check the ClusterID
				return util.CheckClusterID(managedClusterInfo)
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Delete ManagedClusterInfo Automatically after ManagedCluster is deleted", func() {
		var testManagedClusterName = util.RandomName()

		ginkgo.BeforeEach(func() {
			err := util.ImportManagedCluster(clusterClient, testManagedClusterName)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// ManagedClusterInfo should exist
			gomega.Eventually(func() bool {
				existing, err := util.HasResource(dynamicClient, clusterInfoGVR, testManagedClusterName, testManagedClusterName)
				if err != nil {
					return false
				}
				return existing
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			// Delete the managedCluster
			err = util.DeleteClusterResource(dynamicClient, util.ManagedClusterGVR, testManagedClusterName)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("clusterInfo should be deleted automatically.", func() {
			gomega.Eventually(func() bool {
				existing, err := util.HasResource(dynamicClient, clusterInfoGVR, testManagedClusterName, testManagedClusterName)
				if err != nil {
					return false
				}
				return existing
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.BeTrue())
		})
	})
})
