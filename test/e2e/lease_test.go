package e2e

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
)

var (
	podNamespace            = "open-cluster-management-agent"
	managedClusterAddOnName = "work-manager"
)

var _ = ginkgo.Describe("Testing Lease", func() {
	ginkgo.BeforeEach(func() {
		if deployedByACM {
			podNamespace = "open-cluster-management-agent-addon"
		}

		// Create managedClusterAddon apis
		var addon = &addonv1alpha1.ManagedClusterAddOn{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedClusterAddOnName,
				Namespace: defaultManagedCluster,
			},
			Spec: addonv1alpha1.ManagedClusterAddOnSpec{
				InstallNamespace: podNamespace,
			},
		}

		_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Get(context.Background(), managedClusterAddOnName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				_, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Create(context.Background(), addon, metav1.CreateOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			}
		}
	})
	ginkgo.AfterEach(func() {
		err := addonClient.AddonV1alpha1().ManagedClusterAddOns(defaultManagedCluster).Delete(context.Background(), managedClusterAddOnName, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

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
