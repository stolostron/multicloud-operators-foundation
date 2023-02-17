package e2e

import (
	"context"
	"fmt"
	"reflect"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	fconstants "github.com/stolostron/multicloud-operators-foundation/pkg/constants"
	"github.com/stolostron/multicloud-operators-foundation/pkg/controllers/addoninstall"
)

var _ = ginkgo.Describe("Testing installation of work-manager add-on", func() {
	var clusterName string
	var annotations map[string]string
	addonName := "work-manager"
	hostingClusterName := "local-cluster"

	ginkgo.JustBeforeEach(func() {
		cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), clusterName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			cluster = &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:        clusterName,
					Annotations: annotations,
				},
				Spec: clusterv1.ManagedClusterSpec{
					HubAcceptsClient: true,
				},
			}
			_, err = clusterClient.ClusterV1().ManagedClusters().Create(context.Background(), cluster, metav1.CreateOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			return
		}

		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		if reflect.DeepEqual(cluster.Annotations, annotations) {
			return
		}

		cluster = cluster.DeepCopy()
		cluster.Annotations = annotations
		_, err = clusterClient.ClusterV1().ManagedClusters().Update(context.Background(), cluster, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	ginkgo.JustAfterEach(func() {
		err := clusterClient.ClusterV1().ManagedClusters().Delete(context.Background(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	haveNoAddonIt := func() {
		ginkgo.It("should have no add-on installed", ginkgo.Offset(1), func() {
			gomega.Consistently(func() bool {
				_, err := addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(context.Background(), addonName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, 30, 2).Should(gomega.BeTrue())
		})
	}

	haveAddonInDefaultModeIt := func() {
		ginkgo.It("should have add-on installed in default mode", ginkgo.Offset(1), func() {
			var addon *addonapiv1alpha1.ManagedClusterAddOn
			var err error
			gomega.Eventually(func() error {
				addon, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(context.Background(), addonName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Expect(addon).ShouldNot(gomega.BeNil())
			gomega.Expect(addon.Annotations).ShouldNot(gomega.HaveKey(fconstants.AnnotationKlusterletHostingClusterName))
			gomega.Expect(addon.Spec.InstallNamespace).To(gomega.Equal(addoninstall.DefaultAddOnInstallNamespace))
		})
	}

	ginkgo.Context("cluster is imported in default mode", func() {
		ginkgo.When("add-on installation is disabled", func() {
			ginkgo.BeforeEach(func() {
				clusterName = fmt.Sprintf("cluster-default-none-%s", rand.String(5))
				annotations = map[string]string{
					constants.DisableAddonAutomaticInstallationAnnotationKey: "true",
				}
			})

			haveNoAddonIt()
		})

		ginkgo.When("default add-on installation is enabled", func() {
			ginkgo.BeforeEach(func() {
				clusterName = fmt.Sprintf("cluster-default-default-%s", rand.String(5))
				annotations = nil
			})

			haveAddonInDefaultModeIt()
		})
	})

	ginkgo.Context("cluster is imported in hosted mode", func() {
		ginkgo.BeforeEach(func() {
			annotations = map[string]string{
				fconstants.AnnotationKlusterletDeployMode:         "Hosted",
				fconstants.AnnotationKlusterletHostingClusterName: hostingClusterName,
			}
		})

		ginkgo.When("add-on installation is disabled", func() {
			ginkgo.BeforeEach(func() {
				clusterName = fmt.Sprintf("cluster-hosted-none-%s", rand.String(5))
				annotations[constants.DisableAddonAutomaticInstallationAnnotationKey] = "true"
			})

			haveNoAddonIt()
		})

		ginkgo.When("default add-on installation is enabled", func() {
			ginkgo.BeforeEach(func() {
				clusterName = fmt.Sprintf("cluster-hosted-default-%s", rand.String(5))
			})

			haveAddonInDefaultModeIt()
		})

		ginkgo.When("hosed add-on installation is enabled", func() {
			ginkgo.BeforeEach(func() {
				clusterName = fmt.Sprintf("cluster-hosted-hosted-%s", rand.String(5))
				annotations[addoninstall.AnnotationEnableHostedModeAddons] = "true"
			})

			ginkgo.It("should have add-on installed in hosted mode", func() {
				var addon *addonapiv1alpha1.ManagedClusterAddOn
				var err error
				gomega.Eventually(func() error {
					addon, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(context.Background(), addonName, metav1.GetOptions{})
					return err
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

				gomega.Expect(addon).ShouldNot(gomega.BeNil())
				gomega.Expect(addon.Annotations).To(gomega.HaveKeyWithValue(addoninstall.AnnotationAddOnHostingClusterName, hostingClusterName))
				gomega.Expect(addon.Spec.InstallNamespace).To(gomega.Equal(fmt.Sprintf("klusterlet-%s", clusterName)))
			})
		})
	})
})
