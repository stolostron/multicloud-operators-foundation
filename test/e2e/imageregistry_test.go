package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/stolostron/cluster-lifecycle-api/helpers/imageregistry"
	"github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Testing ManagedClusterImageRegistry", func() {
	testNamespace := util.RandomName()
	testImageRegistry := util.RandomName()
	testPlacement := util.RandomName()
	testClusterSet := util.RandomName()
	testClusterSetBinding := testClusterSet
	testCluster := util.RandomName()
	testPullSecret := util.RandomName()
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
		err = util.CreateImageRegistry(dynamicClient, testNamespace, testImageRegistry, testPlacement, testPullSecret)
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

	ginkgo.It("add / remove imageRegistry label and annotation successfully", func() {
		// create a cluster is selected by the imageRegistry
		cluster := util.NewManagedCluster(testCluster)
		labels := selectedLabel
		labels["cluster.open-cluster-management.io/clusterset"] = testClusterSet
		cluster.SetLabels(labels)
		err := util.CreateManagedCluster(clusterClient, cluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// the label and annotation should be added to the cluster
		gomega.Eventually(func() error {
			cluster, err = clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), testCluster, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if len(cluster.GetAnnotations()) == 0 {
				return fmt.Errorf("failed to get annotations")
			}
			if cluster.Annotations[v1alpha1.ClusterImageRegistriesAnnotation] == "" {
				return fmt.Errorf("failed to get registry annotation")
			}

			if len(cluster.GetLabels()) == 0 {
				return fmt.Errorf("failed to get labels")
			}
			if cluster.Labels[v1alpha1.ClusterImageRegistryLabel] == "" {
				return fmt.Errorf("failed to get registry label")
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// should get pull secret successfully from cluster
		gomega.Eventually(func() error {
			pullSecret, err := imageRegistryClient.Cluster(cluster).PullSecret()
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

		// should override the image successfully
		imageName := "registry.redhat.io/multicluster-engine/registration@SHA256abc"
		expectedImageName := "quay.io/multicluster-engine/registration@SHA256abc"
		overrideImageName, err := imageRegistryClient.Cluster(cluster).ImageOverride(imageName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(overrideImageName).Should(gomega.Equal(expectedImageName))
		overrideImageName, err = imageregistry.OverrideImageByAnnotation(cluster.GetAnnotations(), imageName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(overrideImageName).Should(gomega.Equal(expectedImageName))

		// remove the cluster from the placement
		err = util.UpdateManagedClusterLabels(clusterClient, testCluster, map[string]string{"e2e-placement": "remove"})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// the label and annotation should be removed from the cluster
		gomega.Eventually(func() error {
			cluster, err = clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), testCluster, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if len(cluster.GetAnnotations()) != 0 && cluster.Annotations[v1alpha1.ClusterImageRegistriesAnnotation] != "" {
				return fmt.Errorf("should not get annotation %s", v1alpha1.ClusterImageRegistriesAnnotation)
			}

			if len(cluster.GetLabels()) != 0 && cluster.Labels[v1alpha1.ClusterImageRegistryLabel] != "" {
				return fmt.Errorf("should not get label %s", v1alpha1.ClusterImageRegistryLabel)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// should not get a pull secret from cluster
		gomega.Eventually(func() error {
			cluster, err = clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), testCluster, metav1.GetOptions{})
			if err != nil {
				return err
			}
			pullSecret, _ := imageRegistryClient.Cluster(cluster).PullSecret()
			if pullSecret != nil {
				return fmt.Errorf("should not get pull secret. annotation:%v", cluster.GetAnnotations())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// override the image successfully
		imageName = "registry.redhat.io/multicluster-engine/registration@SHA256abc"
		expectedImageName = "registry.redhat.io/multicluster-engine/registration@SHA256abc"
		overrideImageName, err = imageRegistryClient.Cluster(cluster).ImageOverride(imageName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(overrideImageName).Should(gomega.Equal(expectedImageName))
	})
})
