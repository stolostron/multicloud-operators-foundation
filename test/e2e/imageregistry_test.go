package e2e

import (
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
)

var _ = ginkgo.Describe("Testing ManagedClusterImageRegistry", func() {
	testNamespace := util.RandomName()
	testImageRegistry := util.RandomName()
	testPlacement := util.RandomName()
	testClusterSet := util.RandomName()
	testClusterSetBinding := testClusterSet
	testCluster := util.RandomName()
	testPullSecret := util.RandomName()
	testRegistry := "quay.io/test/"
	selectedLabel := map[string]string{"e2e-placement": "testplacement"}

	ginkgo.BeforeEach(func() {
		// create ns
		err := util.CreateNamespace(testNamespace)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// create pullSecret
		err = util.CreatePullSecret(kubeClient, testNamespace, testPullSecret)
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
		err = util.CreateImageRegistry(dynamicClient, testNamespace, testImageRegistry, testPlacement, testPullSecret, testRegistry)
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
			pullSecret, err := imageRegistryClient.Cluster(testCluster).PullSecret()
			if err != nil {
				return err
			}
			if pullSecret == nil {
				return fmt.Errorf("failed to get pullSecret of imageRegistry %v in cluster %v", testImageRegistry, testCluster)
			}
			if pullSecret.Name != testPullSecret {
				return fmt.Errorf("expected pullSecret %v, but got %v", testPullSecret, pullSecret.Name)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			registry, err := imageRegistryClient.Cluster(testCluster).Registry()
			if err != nil {
				return err
			}
			if registry == "" {
				return fmt.Errorf("failed to get registry of imageRegistry %v in cluster %v", testImageRegistry, testCluster)
			}
			if registry != testRegistry {
				return fmt.Errorf("expected registry %v, but got %v", testRegistry, registry)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// remove the cluster from the imageRegistry
		err = util.UpdateManagedClusterLabels(clusterClient, testCluster, map[string]string{"e2e-placement": "remove"})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		gomega.Eventually(func() error {
			pullSecret, err := imageRegistryClient.Cluster(testCluster).PullSecret()
			if err != nil {
				return err
			}

			if pullSecret != nil {
				return fmt.Errorf("expected nil pullSecret, but got %v", pullSecret.Name)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			registry, err := imageRegistryClient.Cluster(testCluster).Registry()
			if err != nil {
				return err
			}

			if registry != "" {
				return fmt.Errorf("expected null registry, but got %v", registry)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

	})
})
