// +build integration

package clusterinfos_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	eventuallyTimeout  = 60
	eventuallyInterval = 2
)

var clusterInfoGVR = schema.GroupVersionResource{
	Group:    "internal.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterinfos",
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

	fakeCluster, err = common.CreateManagedCluster(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred(), "Failed to create fake managedcluster")
})

var _ = AfterSuite(func() {})

var _ = Describe("Testing ManagedClusterInfo", func() {
	Describe("Get ManagedClusterInfo", func() {
		It("should get a ManagedClusterInfo successfully in cluster", func() {
			exists, err := common.HasResource(dynamicClient, clusterInfoGVR, realCluster.GetName(), realCluster.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				ManagedClusterInfo, err := common.GetResource(dynamicClient, clusterInfoGVR, realCluster.GetName(), realCluster.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterInfo status
				return common.GetConditionTypeFromStatus(ManagedClusterInfo, clusterinfov1beta1.ManagedClusterInfoSynced), nil
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		})
	})
	Describe("Delete ManagedCluster", func() {
		BeforeEach(func() {
			//Fake ManagedClusterinfo should exist
			Eventually(func() (bool, error) {
				return common.HasResource(dynamicClient, clusterInfoGVR, fakeCluster.GetName(), fakeCluster.GetName())
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())

			//Delete the fake managedcluster
			err := common.DeleteClusterResource(dynamicClient, common.ManagedClusterGVR, fakeCluster.GetName())
			Ω(err).ShouldNot(HaveOccurred())
		})
		Context("Delete ManagedClusterInfo Automatically", func() {
			It("clusterinfo should be deleted automitically.", func() {
				Eventually(func() (bool, error) {
					return common.HasResource(dynamicClient, clusterInfoGVR, fakeCluster.GetName(), fakeCluster.GetName())
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(BeTrue())
			})
		})
	})
})
