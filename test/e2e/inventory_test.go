package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveinternalv1alpha1 "github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
)

var bmaGVR = schema.GroupVersionResource{
	Group:    "inventory.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "baremetalassets",
}

var _ = ginkgo.Describe("Testing BareMetalAsset", func() {
	var testNamespace string
	var testName string
	ginkgo.BeforeEach(func() {
		testName = "mycluster"
		suffix := rand.String(6)
		testNamespace = fmt.Sprintf("bma-ns-%v", suffix)
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		// create ns
		_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		// create ClusterDeployment
		clusterDeployment := &hivev1.ClusterDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: hivev1.ClusterDeploymentSpec{
				BaseDomain:  "hive.example.com",
				ClusterName: testName,
				Platform:    hivev1.Platform{},
				Provisioning: &hivev1.Provisioning{
					InstallConfigSecretRef: &corev1.LocalObjectReference{
						Name: "secret-ref",
					},
				},
				PullSecretRef: &corev1.LocalObjectReference{
					Name: "pull-ref",
				},
			},
		}
		_, err = hiveClient.HiveV1().ClusterDeployments(testNamespace).Create(context.Background(), clusterDeployment, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterSync := &hiveinternalv1alpha1.ClusterSync{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}
		_, err = hiveClient.HiveinternalV1alpha1().ClusterSyncs(testNamespace).Create(context.Background(), clusterSync, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.AfterEach(func() {
		//clean up clusterset
		err := kubeClient.CoreV1().Namespaces().Delete(context.Background(), testNamespace, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Create bma", func() {
		ginkgo.It("BMA should be auto created successfully", func() {
			// Create bma at first
			bma, err := util.LoadResourceFromJSON(util.BMATemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(bma.Object, testNamespace, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(bma.Object, testName, "metadata", "name")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(bma.Object, testNamespace, "spec", "clusterDeployment", "namespace")
			_, err = util.CreateResource(dynamicClient, bmaGVR, bma)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", bmaGVR.Resource)

			// create secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: testNamespace,
				},
			}
			_, err = kubeClient.CoreV1().Secrets(testNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			// ensure syncset is created
			var syncSet *hivev1.SyncSet
			gomega.Eventually(func() bool {
				syncSet, err = hiveClient.HiveV1().SyncSets(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
				if err != nil {
					return false
				}
				return true
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			// manually update clustersync status
			clusterSync, err := hiveClient.HiveinternalV1alpha1().ClusterSyncs(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			clusterSync.Status.SyncSets = []hiveinternalv1alpha1.SyncStatus{
				{
					Name:               testName,
					ObservedGeneration: syncSet.Generation,
					LastTransitionTime: metav1.Now(),
					Result:             hiveinternalv1alpha1.SuccessSyncSetResult,
				},
			}
			_, err = hiveClient.HiveinternalV1alpha1().ClusterSyncs(testNamespace).UpdateStatus(context.Background(), clusterSync, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			// ensure conditions of bma are correct
			gomega.Eventually(func() error {
				bma, err := dynamicClient.Resource(bmaGVR).Namespace(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				status := bma.Object["status"]
				conditions := status.(map[string]interface{})["conditions"]
				for _, condition := range conditions.([]interface{}) {
					conditionStatus := condition.(map[string]interface{})["status"]
					if conditionStatus.(string) != "True" {
						fmt.Printf("conditon fail: %v\n", conditions)
						clusterSync, err := hiveClient.HiveinternalV1alpha1().ClusterSyncs(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
						fmt.Printf("clusterSync: %v,%v\n", clusterSync, err)
						syncSet, err := hiveClient.HiveV1().SyncSets(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
						fmt.Printf("syncSet: %v,%v\n", syncSet, err)
						return fmt.Errorf("contidion %v is not correct, reason %v, message %v",
							condition.(map[string]interface{})["type"],
							condition.(map[string]interface{})["reason"],
							condition.(map[string]interface{})["message"])
					}
				}
				return nil

			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Deleting a ClusterDeployment", func() {
		ginkgo.BeforeEach(func() {
			bma, err := util.LoadResourceFromJSON(util.BMATemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(bma.Object, testNamespace, "metadata", "namespace")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(bma.Object, testName, "metadata", "name")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = unstructured.SetNestedField(bma.Object, testNamespace, "spec", "clusterDeployment", "namespace")
			// create managedClusterView to real cluster
			_, err = util.CreateResource(dynamicClient, bmaGVR, bma)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", bmaGVR.Resource)
		})

		ginkgo.It("delete clusterdeployment should clean ref in bma", func() {
			gomega.Eventually(func() (interface{}, error) {
				cd, err := hiveClient.HiveV1().ClusterDeployments(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
				if err != nil {
					return []string{}, err
				}
				return cd.Finalizers, nil
			}, 60*time.Second, 1*time.Second).Should(gomega.Equal([]string{"baremetalasset.inventory.open-cluster-management.io"}))

			gomega.Eventually(func() bool {
				bma, err := util.GetResource(dynamicClient, bmaGVR, testNamespace, testName)
				if err != nil {
					return false
				}
				labels := bma.GetLabels()
				if labels["metal3.io/cluster-deployment-name"] != testName {
					return false
				}
				if labels["metal3.io/cluster-deployment-namespace"] != testNamespace {
					return false
				}
				return true
			}, 60*time.Second, 1*time.Second).Should(gomega.BeTrue())

			err := hiveClient.HiveV1().ClusterDeployments(testNamespace).Delete(context.Background(), testName, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				bma, err := util.GetResource(dynamicClient, bmaGVR, testNamespace, testName)
				if err != nil {
					return false
				}
				name, _, _ := unstructured.NestedString(bma.Object, "spec", "clusterDeployment", "name")
				namespace, _, _ := unstructured.NestedString(bma.Object, "spec", "clusterDeployment", "namespace")
				if name != "" || namespace != "" {
					return false
				}
				return true
			}, 60*time.Second, 1*time.Second).Should(gomega.BeTrue())

			gomega.Eventually(func() bool {
				_, err := hiveClient.HiveV1().ClusterDeployments(testNamespace).Get(context.Background(), testName, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 60*time.Second, 1*time.Second).Should(gomega.BeTrue())
		})
	})
})
