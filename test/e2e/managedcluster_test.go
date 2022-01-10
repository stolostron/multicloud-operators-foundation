package e2e

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	e2eutil "github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Testing ManagedCluster", func() {
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

			gomega.Eventually(func() bool {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), defaultManagedCluster, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				if len(cluster.Spec.ManagedClusterClientConfigs) == 0 {
					return false
				}
				for _, config := range cluster.Spec.ManagedClusterClientConfigs {
					if config.URL != apiserverAddress {
						continue
					}
					if reflect.DeepEqual(config.CABundle, fakeSecret.Data["tls.crt"]) {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

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
			//Only need to test this case in ocp
			if !isOcp {
				return
			}
			serviceAccountCa, err := utils.GetCAFromServiceAccount(context.TODO(), kubeClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				cluster, err := clusterClient.ClusterV1().ManagedClusters().Get(context.Background(), defaultManagedCluster, metav1.GetOptions{})
				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				if len(cluster.Spec.ManagedClusterClientConfigs) == 0 {
					return false
				}
				for _, config := range cluster.Spec.ManagedClusterClientConfigs {
					if reflect.DeepEqual(config.CABundle, serviceAccountCa) {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

	})
})
