package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	podNamespace            = "open-cluster-management-agent-addon"
	managedClusterAddOnName = "work-manager"
)

var _ = ginkgo.Describe("Testing Lease", func() {
	ginkgo.Context("Get Lease", func() {
		ginkgo.It("should get/update lease successfully in cluster", func() {
			var firstLeaseTime *metav1.MicroTime

			gomega.Eventually(func() error {
				lease, err := kubeClient.CoordinationV1().Leases(podNamespace).Get(context.Background(), managedClusterAddOnName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				firstLeaseTime = lease.Spec.RenewTime
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
			gomega.Eventually(func() error {
				updatedLease, err := kubeClient.CoordinationV1().Leases(podNamespace).Get(context.Background(), managedClusterAddOnName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				updatedLeaseTime := updatedLease.Spec.RenewTime
				if updatedLeaseTime.Equal(firstLeaseTime) {
					return fmt.Errorf("lease should be updated")
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
			// Ensure the addon status is correct
			gomega.Eventually(func() bool {
				addon, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Get(context.Background(), managedClusterAddOnName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return meta.IsStatusConditionTrue(addon.Status.Conditions, "Available")
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})
	})
})
