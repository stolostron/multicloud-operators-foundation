// +build integration

package clusterinfos_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	eventuallyTimeout  = 60
	eventuallyInterval = 2
)

var clusterInofGVR = schema.GroupVersionResource{
	Group:    "internal.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "managedclusterinfos",
}

var (
	dynamicClient dynamic.Interface
	realCluster   *unstructured.Unstructured
)

var _ = BeforeSuite(func() {
	var err error
	dynamicClient, err = common.NewDynamicClient()
	Ω(err).ShouldNot(HaveOccurred())

	realClusters, err := common.GetJoinedManagedClusters(dynamicClient)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(len(realClusters)).ShouldNot(Equal(0))

	realCluster = realClusters[0]
})

var _ = AfterSuite(func() {})

var _ = Describe("Testing ManagedClusterInfo", func() {
	Context("Get ManagedClusterInfo", func() {
		It("should get a ManagedClusterInfo successfully in cluster", func() {
			exists, err := common.HasResource(dynamicClient, clusterInofGVR, realCluster.GetName(), realCluster.GetName())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(exists).Should(BeTrue())
		})

		It("should have a valid condition", func() {
			Eventually(func() (interface{}, error) {
				ManagedClusterInfo, err := common.GetResource(dynamicClient, clusterInofGVR, realCluster.GetName(), realCluster.GetName())
				if err != nil {
					return "", err
				}
				// check the ManagedClusterInfo status
				return common.GetConditionTypeFromStatus(ManagedClusterInfo, clusterv1.ManagedClusterConditionJoined), nil
			}, eventuallyTimeout, eventuallyInterval).Should(BeTrue())
		})
	})
})
