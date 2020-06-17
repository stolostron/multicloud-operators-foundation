// +build integration

package cluster_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
var clusterGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
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
	err = common.DeleteClusterResource(dynamicClient, clusterGVR, realCluster.GetName())
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {})

var _ = Describe("Testing ManagedClusterInfo", func() {
	Context("Get ManagedClusterInfo", func() {
		It("clusterinfo shoudl be deleted automitically.", func() {
			Eventually(func() (bool, error) {
				return common.HasResource(dynamicClient, clusterInfoGVR, realCluster.GetName(), realCluster.GetName())
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(BeTrue())
		})
	})
})
