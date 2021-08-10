package e2e

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/controllers/imageregistry"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Testing ManagedClusterImageRegistry", func() {
	testNamespace := util.RandomName()
	testImageRegistry := util.RandomName()
	testPlacement := util.RandomName()
	testClusterSet := util.RandomName()
	testClusterSetBinding := testClusterSet
	testCluster := util.RandomName()
	selectedLabel := map[string]string{"e2e-placement": "testplacement"}

	ginkgo.BeforeEach(func() {
		// create ns
		err := util.CreateNamespace(testNamespace)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// create clusterSet
		err = util.CreateManagedClusterSet(clusterClient, testClusterSet)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// create clusterSetBinding
		err = util.CreateManagedClusterSetBinding(clusterClient, testNamespace, testClusterSetBinding, testClusterSet)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// create placement
		err = util.CreatePlacement(clusterClient, testNamespace, testPlacement, []string{testClusterSet}, selectedLabel)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// create imageRegistry
		err = util.CreateImageRegistry(dynamicClient, testNamespace, testImageRegistry, testPlacement)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		// delete cluster
		err := util.CleanManagedCluster(clusterClient, testCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// delete imageRegistry
		err = util.DeleteImageRegistry(dynamicClient, testNamespace, testImageRegistry)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// delete placement
		err = util.DeletePlacement(clusterClient, testNamespace, testPlacement)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// delete clusterSetBinding
		err = util.DeleteManagedClusterSetBinding(clusterClient, testNamespace, testClusterSetBinding)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// delete clusterSet
		err = util.DeleteManagedClusterSet(clusterClient, testClusterSet)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// delete ns
		err = util.DeleteNamespace(testNamespace)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("add / remove imageRegistry label successfully", func() {
		// create a cluster is selected by the imageRegistry
		cluster := util.NewManagedCluster(testCluster)
		labels := selectedLabel
		labels["cluster.open-cluster-management.io/clusterset"] = testClusterSet
		cluster.SetLabels(labels)
		err := util.CreateManagedCluster(clusterClient, cluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), testCluster, metav1.GetOptions{})
			if err != nil {
				return err
			}
			labels := cluster.GetLabels()
			if labels == nil {
				return fmt.Errorf("no labels got in cluster")
			}
			if labels[imageregistry.ClusterImageRegistryLabel] != testNamespace+"."+testImageRegistry {
				return fmt.Errorf("the cluster has wrong imageRegistry label %v", labels[imageregistry.ClusterImageRegistryLabel])
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// remove the cluster from the imageRegistry
		err = util.UpdateManagedClusterLabels(clusterClient, testCluster, map[string]string{"e2e-placement": "remove"})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		gomega.Eventually(func() error {
			cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), testCluster, metav1.GetOptions{})
			if err != nil {
				return err
			}
			labels := cluster.GetLabels()
			if labels == nil {
				return nil
			}
			if labels[imageregistry.ClusterImageRegistryLabel] != "" {
				return fmt.Errorf("imageRegistry label %v has not be removed", labels[imageregistry.ClusterImageRegistryLabel])
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

	})
})
