package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"

	"k8s.io/apimachinery/pkg/util/rand"
)

var requiredClaimNames = []string{
	"id.k8s.io",
	"kubeversion.open-cluster-management.io",
}

var _ = ginkgo.Describe("Testing ClusterClaim", func() {

	ginkgo.It("should get the required ClusterClaims successfully the managed cluster", func() {
		gomega.Eventually(func() error {
			for _, claimName := range requiredClaimNames {
				_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
				if err != nil {
					return err
				}
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should recreate the ClusterClaim once it is deleted", func() {
		claimName := requiredClaimNames[0]

		// make sure the claim is in place
		gomega.Eventually(func() error {
			_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// delete the claim
		err := clusterClient.ClusterV1alpha1().ClusterClaims().Delete(context.TODO(), claimName, v1.DeleteOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// check if it is recreated
		gomega.Eventually(func() error {
			_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should rollback the change of the ClusterClaim once it is modified", func() {
		claimName := requiredClaimNames[1]

		// get the original claim
		var claim *clusterv1alpha1.ClusterClaim
		var err error
		gomega.Eventually(func() error {
			claim, err = clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		originalValue := claim.Spec.Value

		// modify the the claim value
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			claim, err = clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			if err != nil {
				return err
			}
			claim.Spec.Value = rand.String(6)
			_, err = clusterClient.ClusterV1alpha1().ClusterClaims().Update(context.TODO(), claim, v1.UpdateOptions{})
			return err
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// check if the change is rollback
		gomega.Eventually(func() error {
			claim, err = clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			if err != nil {
				return err
			}
			if originalValue != claim.Spec.Value {
				return fmt.Errorf("the claim is not rollback")
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should not rollback the change of the ClusterClaim, if it's in create only list", func() {
		claimName := requiredClaimNames[0]

		// get the original claim
		var claim *clusterv1alpha1.ClusterClaim
		var err error
		gomega.Eventually(func() error {
			claim, err = clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		originalValue := claim.Spec.Value

		// modify the the claim value
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			claim, err = clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			if err != nil {
				return err
			}
			claim.Spec.Value = rand.String(6)
			_, err = clusterClient.ClusterV1alpha1().ClusterClaims().Update(context.TODO(), claim, v1.UpdateOptions{})
			return err
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// check if the change is rollback
		gomega.Eventually(func() error {
			claim, err = clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), claimName, v1.GetOptions{})
			if err != nil {
				return err
			}
			if originalValue == claim.Spec.Value {
				return fmt.Errorf("the claim is rollback")
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should sync the label to claim", func() {
		var err error
		// add label to managedCluster
		err = util.UpdateManagedClusterLabels(clusterClient, defaultManagedCluster, map[string]string{"test": "test"})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// should get the claim synced from label
		gomega.Eventually(func() error {
			_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), "test", v1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// delete label from managedCluster
		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), defaultManagedCluster, v1.GetOptions{})
			if err != nil {
				return err
			}
			delete(cluster.Labels, "test")
			_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.TODO(), cluster, v1.UpdateOptions{})
			return err
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// should delete the claim synced from label
		gomega.Eventually(func() error {
			_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), "test", v1.GetOptions{})
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed delete claim syned from label %v", err)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should not remove the customized claim", func() {
		var err error

		customizedClaim := &clusterv1alpha1.ClusterClaim{
			ObjectMeta: v1.ObjectMeta{
				Name: rand.String(6),
			},
			Spec: clusterv1alpha1.ClusterClaimSpec{
				Value: rand.String(6),
			},
		}
		_, err = clusterClient.ClusterV1alpha1().ClusterClaims().Create(context.TODO(), customizedClaim, v1.CreateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// should get the customized claim in 1 min
		count := 0
		gomega.Eventually(func() int {
			_, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.TODO(), customizedClaim.Name, v1.GetOptions{})
			if err == nil {
				count = count + 1
			}
			return count
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeNumerically(">=", 30))

		// clean the customized Claim
		err = clusterClient.ClusterV1alpha1().ClusterClaims().Delete(context.TODO(), customizedClaim.Name, v1.DeleteOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

	})
})
