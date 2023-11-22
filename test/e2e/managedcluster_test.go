package e2e

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strconv"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	e2eutil "github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Testing ManagedCluster", func() {
	var testCluster string
	var adminClusterClusterRoleName string
	var viewClusterClusterRoleName string

	ginkgo.BeforeEach(func() {
		testCluster = util.RandomName()
		adminClusterClusterRoleName = utils.GenerateClusterRoleName(testCluster, "admin")
		viewClusterClusterRoleName = utils.GenerateClusterRoleName(testCluster, "view")
		cluster := util.NewManagedCluster(testCluster)
		err := util.CreateManagedCluster(clusterClient, cluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.AfterEach(func() {
		// delete cluster
		err := util.CleanManagedCluster(clusterClient, testCluster)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Get ManagedCluster cpu worker capacity", func() {
		ginkgo.It("should get a cpu_worker successfully in status of managedcluster", func() {
			gomega.Eventually(func() error {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), defaultManagedCluster, metav1.GetOptions{})
				if err != nil {
					return err
				}

				capacity := cluster.Status.Capacity
				if _, ok := capacity["core_worker"]; !ok {
					return fmt.Errorf("Expect core_worker to be set, but got %v", capacity)
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Testing Clusterca sync", func() {
		ginkgo.It("Get CA from apiserver", func() {
			//Only need to test this case in ocp
			if !isOcp {
				return
			}
			//Create a fake secret for apiserver
			fakesecretName := "fake-server-secret"
			fakeSecret, err := e2eutil.CreateFakeTlsSecret(kubeClient, fakesecretName, utils.OpenshiftConfigNamespace)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			//get apiserveraddress
			apiserverAddress, err := utils.GetKubeAPIServerAddress(context.TODO(), ocpClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			//add serving secret in apiserver
			url, err := url.Parse(apiserverAddress)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			apiserver, err := ocpClient.ConfigV1().APIServers().Get(context.TODO(), utils.ApiserverConfigName, metav1.GetOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			newApiserver := apiserver.DeepCopy()
			newApiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{
				{
					Names: []string{
						url.Hostname(),
					},
					ServingCertificate: configv1.SecretNameReference{
						Name: fakesecretName,
					},
				},
			}

			newApiserver, err = ocpClient.ConfigV1().APIServers().Update(context.TODO(), newApiserver, metav1.UpdateOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), defaultManagedCluster, metav1.GetOptions{})
				if err != nil {
					return err
				}

				if len(cluster.Spec.ManagedClusterClientConfigs) == 0 {
					return fmt.Errorf("cluster.Spec.ManagedClusterClientConfigs should not be 0")
				}
				for _, config := range cluster.Spec.ManagedClusterClientConfigs {
					if config.URL != apiserverAddress {
						continue
					}
					if reflect.DeepEqual(config.CABundle, fakeSecret.Data["tls.crt"]) {
						return nil
					}
				}
				return fmt.Errorf("cannot found config")
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//rollback apiserver and delete secret
			newApiserver.Spec.ServingCerts.NamedCertificates = []configv1.APIServerNamedServingCert{}
			_, err = ocpClient.ConfigV1().APIServers().Update(context.TODO(), newApiserver, metav1.UpdateOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			err = kubeClient.CoreV1().Secrets(utils.OpenshiftConfigNamespace).Delete(context.TODO(), fakesecretName, metav1.DeleteOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("Get CA from configmap", func() {
			//Only need to test this case in ocp
			if !isOcp {
				return
			}
			configmapCa, err := utils.GetCAFromConfigMap(context.TODO(), kubeClient)
			if err != nil {
				if errors.IsNotFound(err) {
					_, err = e2eutil.CreateFakeRootCaConfigMap(kubeClient, utils.CrtConfigmapName, utils.ConfigmapNamespace)
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				} else {
					gomega.Expect(err).ToNot(gomega.HaveOccurred())
				}
			}
			configmapCa, err = utils.GetCAFromConfigMap(context.TODO(), kubeClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), defaultManagedCluster, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				if len(cluster.Spec.ManagedClusterClientConfigs) == 0 {
					return false
				}

				for _, config := range cluster.Spec.ManagedClusterClientConfigs {
					if reflect.DeepEqual(config.CABundle, configmapCa) {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//delete configmap
			err = kubeClient.CoreV1().ConfigMaps(utils.ConfigmapNamespace).Delete(context.TODO(), utils.CrtConfigmapName, metav1.DeleteOptions{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})

		ginkgo.It("Get CA from service account", func() {
			// Only need to test this case in ocp (kubernetes version < 1.24)
			if !isOcp {
				ginkgo.By("hub is not ocp, skip running this test")
				return
			}

			if hubVersionInfo == nil {
				ginkgo.By("hub version is nil, skip running this test")
				return
			}
			major, err := strconv.Atoi(hubVersionInfo.Major)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			minor, err := strconv.Atoi(hubVersionInfo.Minor)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// check if the hub version is 1.24 or above
			if major != 1 || minor >= 24 {
				ginkgo.By(fmt.Sprintf("hub version is %s, skip running this test", hubVersionInfo.String()))
				return
			}
			serviceAccountCa, err := utils.GetCAFromServiceAccount(context.TODO(), kubeClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), defaultManagedCluster, metav1.GetOptions{})
				if err != nil {
					return err
				}

				if len(cluster.Spec.ManagedClusterClientConfigs) == 0 {
					return fmt.Errorf("cluster.Spec.ManagedClusterClientConfigs should not be 0")
				}
				for _, config := range cluster.Spec.ManagedClusterClientConfigs {
					if reflect.DeepEqual(config.CABundle, serviceAccountCa) {
						return nil
					}
				}
				return fmt.Errorf("cannot found config")
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

	})

	ginkgo.Context("Check clusterrole sync", func() {

		ginkgo.It("Check if clusterrole admin/view created", func() {
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			ginkgo.By("clusterrole admin/view should be deleted after managedcluster deleted")
			err := util.CleanManagedCluster(clusterClient, testCluster)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Eventually(func() bool {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
			gomega.Eventually(func() bool {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				return errors.IsNotFound(err)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Check if admin/view clusterrole could be recreated after delete it", func() {
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
			err := kubeClient.RbacV1().ClusterRoles().Delete(context.Background(), adminClusterClusterRoleName, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = kubeClient.RbacV1().ClusterRoles().Delete(context.Background(), viewClusterClusterRoleName, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("Check if admin clusterrole could be reconcile after update it", func() {
			gomega.Eventually(func() error {
				adminClusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				updatedAdminClusterRole := adminClusterRole.DeepCopy()
				updatedAdminClusterRole.Rules = []v1.PolicyRule{}
				updatedAdminClusterRole, err = kubeClient.RbacV1().ClusterRoles().Update(context.Background(), updatedAdminClusterRole, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
				gomega.Eventually(func() error {
					updatedAdminClusterRole, err = kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if len(updatedAdminClusterRole.Rules) == len(adminClusterRole.Rules) {
						return nil
					}
					return fmt.Errorf("The admin clusterrole should be reconciled. updatedAdminClusterRole: %v,adminClusterRole: %v", updatedAdminClusterRole, adminClusterRole)
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
		ginkgo.It("Check if view clusterrole could be reconcile after update it", func() {
			gomega.Eventually(func() error {
				viewClusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				updatedViewClusterRole := viewClusterRole.DeepCopy()
				updatedViewClusterRole.Rules = []v1.PolicyRule{}
				updatedViewClusterRole, err = kubeClient.RbacV1().ClusterRoles().Update(context.Background(), updatedViewClusterRole, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
				gomega.Eventually(func() error {
					updatedViewClusterRole, err = kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if len(updatedViewClusterRole.Rules) == len(viewClusterRole.Rules) {
						return nil
					}
					return fmt.Errorf("The admin clusterrole should be reconciled. updatedViewClusterRole: %v,viewClusterRole: %v", updatedViewClusterRole, viewClusterRole)
				}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClusterClusterRoleName, metav1.GetOptions{})
				return err
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})

	})
})
