package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"github.com/stolostron/multicloud-operators-foundation/pkg/constants"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
)

var _ = ginkgo.Describe("Testing managed cluster deletion", func() {
	var userName = rand.String(6)
	var clusterName = "integration-" + userName

	ginkgo.It("Can not delete a cluster when it is hosting a hypershift cluster", func() {
		cluster := util.NewManagedCluster(clusterName)
		cluster.SetLabels(map[string]string{
			constants.LabelFeatureHypershiftAddon: "available",
		})

		ginkgo.By(fmt.Sprintf("create a managedCluster %s", clusterName))
		gomega.Eventually(func() error {
			err := util.CreateManagedCluster(clusterClient, cluster)
			if err != nil {
				return err
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("Update the hosting cluster %s claim, has hosted cluster", clusterName))
		gomega.Eventually(func() error {
			c, err := clusterClient.ClusterV1().ManagedClusters().Get(
				context.TODO(), clusterName, metav1.GetOptions{})
			if err != nil {
				ginkgo.By(fmt.Sprintf("get cluster status error: %v", err))
				return err
			}

			c.Status = clusterv1.ManagedClusterStatus{
				ClusterClaims: []clusterv1.ManagedClusterClaim{
					{
						Name:  constants.ClusterClaimHostedClusterCountZero,
						Value: "false",
					},
				},
				Conditions: []metav1.Condition{
					{
						Type:               clusterv1.ManagedClusterConditionAvailable,
						Status:             metav1.ConditionTrue,
						Reason:             "available",
						LastTransitionTime: metav1.Time{Time: time.Now()},
					},
				},
			}
			_, err = clusterClient.ClusterV1().ManagedClusters().UpdateStatus(
				context.TODO(), c, metav1.UpdateOptions{})
			if err != nil {
				ginkgo.By(fmt.Sprintf("update cluster status error: %v", err))
				return err
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The hosting cluster %s can not be deleted", clusterName))
		err := clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).Should(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("Update the hosting cluster %s claim to no hosted cluster", clusterName))
		gomega.Eventually(func() error {
			c, err := clusterClient.ClusterV1().ManagedClusters().Get(
				context.TODO(), clusterName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			c.Status.ClusterClaims = nil
			_, err = clusterClient.ClusterV1().ManagedClusters().UpdateStatus(
				context.TODO(), c, metav1.UpdateOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The cluster %s can be deleted now", clusterName))
		err = clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing cluster deployment deletion", func() {
	var namespace = rand.String(6)
	var clusterName = "integration-" + namespace

	ginkgo.BeforeEach(func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err := kubeClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		err := kubeClient.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.It("Can not delete a cluster deployment when it is hosting a hypershift cluster", func() {

		clusterDeployment := &hivev1.ClusterDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
		}
		ginkgo.By(fmt.Sprintf("create a clusterDeployment %s", clusterName))
		gomega.Eventually(func() error {
			_, err := hiveClient.HiveV1().ClusterDeployments(namespace).Create(
				context.TODO(), clusterDeployment, metav1.CreateOptions{})
			if err != nil {
				ginkgo.By(fmt.Sprintf("create cluster deployment err %v", err))
				return err
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		cluster := util.NewManagedCluster(clusterName)
		cluster.SetLabels(map[string]string{
			constants.LabelFeatureHypershiftAddon: "available",
		})

		ginkgo.By(fmt.Sprintf("create a managedCluster %s", clusterName))
		gomega.Eventually(func() error {
			err := util.CreateManagedCluster(clusterClient, cluster)
			if err != nil {
				return err
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("Update the hosting cluster %s claim, has hosted cluster", clusterName))
		gomega.Eventually(func() error {
			c, err := clusterClient.ClusterV1().ManagedClusters().Get(
				context.TODO(), clusterName, metav1.GetOptions{})
			if err != nil {
				ginkgo.By(fmt.Sprintf("get cluster status error: %v", err))
				return err
			}

			c.Status = clusterv1.ManagedClusterStatus{
				ClusterClaims: []clusterv1.ManagedClusterClaim{
					{
						Name:  constants.ClusterClaimHostedClusterCountZero,
						Value: "false",
					},
				},
				Conditions: []metav1.Condition{
					{
						Type:               clusterv1.ManagedClusterConditionAvailable,
						Status:             metav1.ConditionTrue,
						Reason:             "available",
						LastTransitionTime: metav1.Time{Time: time.Now()},
					},
				},
			}
			_, err = clusterClient.ClusterV1().ManagedClusters().UpdateStatus(
				context.TODO(), c, metav1.UpdateOptions{})
			if err != nil {
				ginkgo.By(fmt.Sprintf("update cluster status error: %v", err))
				return err
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The cluster deployment %s can not be deleted", clusterName))
		err := hiveClient.HiveV1().ClusterDeployments(namespace).Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).Should(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The hosting cluster %s can not be deleted", clusterName))
		err = clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).Should(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("Update the hosting cluster %s claim to no hosted cluster", clusterName))
		gomega.Eventually(func() error {
			c, err := clusterClient.ClusterV1().ManagedClusters().Get(
				context.TODO(), clusterName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			c.Status.ClusterClaims = nil
			_, err = clusterClient.ClusterV1().ManagedClusters().UpdateStatus(
				context.TODO(), c, metav1.UpdateOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The cluster deployment %s can be deleted now", clusterName))
		err = hiveClient.HiveV1().ClusterDeployments(namespace).Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The cluster %s can be deleted now", clusterName))
		err = clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
})
