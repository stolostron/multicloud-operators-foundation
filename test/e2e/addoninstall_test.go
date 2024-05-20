package e2e

import (
	"context"
	"fmt"
	"reflect"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	addonlib "github.com/stolostron/multicloud-operators-foundation/pkg/addon"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
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

	haveAddonInDefaultModeIt := func() {
		ginkgo.It("should have add-on installed in default mode", ginkgo.Offset(1), func() {
			var addon *addonapiv1alpha1.ManagedClusterAddOn
			var err error
			gomega.Eventually(func() error {
				addon, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(context.Background(), addonName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Expect(addon).ShouldNot(gomega.BeNil())
			gomega.Expect(addon.Status.Namespace).To(gomega.Equal("open-cluster-management-agent-addon"))
		})
	}

	ginkgo.Context("cluster is imported in default mode", func() {
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
				apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
				apiconstants.AnnotationKlusterletHostingClusterName: hostingClusterName,
			}
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
				annotations = map[string]string{
					apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
					apiconstants.AnnotationKlusterletHostingClusterName: hostingClusterName,
					addonlib.AnnotationEnableHostedModeAddons:           "true",
				}
			})

			ginkgo.It("should have add-on installed in hosted mode", func() {
				var addon *addonapiv1alpha1.ManagedClusterAddOn
				var err error
				gomega.Eventually(func() error {
					addon, err = addonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(context.Background(), addonName, metav1.GetOptions{})
					return err
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

				gomega.Expect(addon).ShouldNot(gomega.BeNil())
				gomega.Expect(addon.Status.Namespace).To(gomega.Equal(fmt.Sprintf("klusterlet-%s", clusterName)))
			})
		})
	})
})
