package e2e

import (
	"context"
	"fmt"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stolostron/cluster-lifecycle-api/constants"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"

	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
)

var _ = ginkgo.Describe("Testing user create/update managedCluster without mangedClusterSet label", func() {
	var userName = rand.String(6)
	var clusterName = "e2e-" + userName
	var rbacName = "e2e-" + userName
	var userClusterClient clusterclient.Interface
	ginkgo.BeforeEach(func() {
		var err error
		// create rbac with managedClusterSet/join <all> permission for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("create").Groups(clusterv1beta2.GroupName).Resources("managedclustersets/join").RuleOrDie(),
			helpers.NewRule("create", "update", "get").Groups(clusterv1.GroupName).Resources("managedclusters").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, rbacName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, rbacName, rbacName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the cluster client
		userClusterClient, err = util.NewClusterClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		var err error
		err = util.CleanManagedCluster(clusterClient, clusterName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRoleBinding(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRole(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should create and update the managedCluster successfully", func() {
		cluster := util.NewManagedCluster(clusterName)
		// case 1: create managedCluster without managedClusterSet label
		gomega.Eventually(func() error {
			return util.CreateManagedCluster(userClusterClient, cluster)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// case 2: update managedCluster with the managedClusterSet label
		labels := map[string]string{
			"cluster.open-cluster-management.io/clusterset": "clusterSet-e2e",
		}
		gomega.Eventually(func() error {
			return util.UpdateManagedClusterLabels(userClusterClient, clusterName, labels)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// case 3: update managedCluster to remove the managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": "",
		}
		gomega.Eventually(func() error {
			return util.UpdateManagedClusterLabels(userClusterClient, clusterName, labels)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing user create/update managedCluster with mangedClusterSet label", func() {
	var userName = rand.String(6)
	var clusterName = "e2e-" + userName
	var rbacName = "e2e-" + userName
	var clusterSet1 = "clusterset1-e2e"
	var clusterSet2 = "clusterset2-e2e"
	var userClusterClient clusterclient.Interface
	ginkgo.BeforeEach(func() {
		var err error
		// create rbac with managedClusterSet/join clusterset-e2e permission for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("create").Groups(clusterv1beta2.GroupName).Resources("managedclustersets/join").Names(clusterSet1, clusterSet2).RuleOrDie(),
			helpers.NewRule("create", "update", "get").Groups(clusterv1.GroupName).Resources("managedclusters").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, rbacName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, rbacName, rbacName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the cluster client
		userClusterClient, err = util.NewClusterClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		var err error
		err = util.CleanManagedCluster(clusterClient, clusterName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRoleBinding(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRole(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should create and update the managedCluster successfully", func() {
		cluster := util.NewManagedCluster(clusterName)
		// case 1: cannot create managedCluster without managedClusterSet label
		gomega.Eventually(func() error {
			err := util.CreateManagedCluster(userClusterClient, cluster)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

		// case 2: cannot create managedCluster with wrong managedClusterSet label
		wrongLabel := "wrong-clusterSet"

		labels := map[string]string{
			"cluster.open-cluster-management.io/clusterset": wrongLabel,
		}
		cluster.SetLabels(labels)
		gomega.Eventually(func() error {
			err := util.CreateManagedCluster(userClusterClient, cluster)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

		// case 3: can create managedCluster with right managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": clusterSet1,
		}
		cluster.SetLabels(labels)
		gomega.Eventually(func() error {
			return util.CreateManagedCluster(userClusterClient, cluster)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// case 4: can update managedCluster with right managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": clusterSet2,
		}
		gomega.Eventually(func() error {
			return util.UpdateManagedClusterLabels(userClusterClient, clusterName, labels)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// case 5: cannot update managedCluster to remove managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": "",
		}
		gomega.Eventually(func() error {
			err := util.UpdateManagedClusterLabels(userClusterClient, clusterName, labels)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

		// case 6: cannot update managedCluster to remove managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": "wrong-set",
		}
		gomega.Eventually(func() error {
			err := util.UpdateManagedClusterLabels(userClusterClient, clusterName, labels)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
	})

})

var _ = ginkgo.Describe("Testing user create/update clusterdeployment with mangedClusterSet label", func() {
	var userName = rand.String(6)
	var clusterDeploymentName = "e2e-" + userName
	var rbacName = "e2e-" + userName
	var clusterSet1 = "clusterset1-e2e"
	var clusterSet2 = "clusterset2-e2e"
	var userHiveClient client.Client
	ginkgo.BeforeEach(func() {
		var err error
		// create rbac with managedClusterSet/join clusterset-e2e permission for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("create").Groups(clusterv1beta2.GroupName).Resources("managedclustersets/join").Names(clusterSet1, clusterSet2).RuleOrDie(),
			helpers.NewRule("create", "update", "get").Groups(clusterv1.GroupName).Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("create", "update", "get").Groups(hivev1.HiveAPIGroup).Resources("clusterdeployments").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, rbacName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, rbacName, rbacName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the cluster client
		userHiveClient, err = util.NewHiveClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		var err error
		err = util.CleanClusterDeployment(hiveClient, clusterDeploymentName, clusterDeploymentName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRoleBinding(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRole(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should create and update the clusterdeployment successfully", func() {
		// case 1: Normal user cannot create clusterdeployment without managedClusterSet label
		err := util.CreateClusterDeployment(userHiveClient, clusterDeploymentName, clusterDeploymentName, "", "", nil, false)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		// case 2: Normal user cannot create clusterdeployment with wrong managedClusterSet label		wrongLabel := "wrong-clusterSet"

		labels := map[string]string{
			"cluster.open-cluster-management.io/clusterset": "wrongLabel",
		}
		err = util.CreateClusterDeployment(userHiveClient, clusterDeploymentName, clusterDeploymentName, "", "", labels, false)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		// case 3:  Normal user can create clusterdeployment with right managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": clusterSet1,
		}
		err = util.CreateClusterDeployment(userHiveClient, clusterDeploymentName, clusterDeploymentName, "", "", labels, false)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// case 4: can update managedClusterDeployment with right managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": clusterSet2,
		}
		gomega.Eventually(func() error {
			return util.UpdateClusterDeploymentLabels(userHiveClient, clusterDeploymentName, clusterDeploymentName, labels)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// case 5: cannot update managedCluster to remove managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": "",
		}
		gomega.Eventually(func() error {
			err := util.UpdateClusterDeploymentLabels(userHiveClient, clusterDeploymentName, clusterDeploymentName, labels)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

		// case 6: cannot update managedCluster to a wrong managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": "wrong-set",
		}
		gomega.Eventually(func() error {
			err := util.UpdateClusterDeploymentLabels(userHiveClient, clusterDeploymentName, clusterDeploymentName, labels)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
	})
	ginkgo.It("should create and update the clusterdeployment from clusterpool successfully", func() {
		err := util.CreateClusterDeployment(userHiveClient, clusterDeploymentName, clusterDeploymentName, "pool", "pool-ns", nil, false)
		gomega.Expect(err).Should(gomega.HaveOccurred())
	})
	ginkgo.It("should create and update the clusterdeployment from AI successfully", func() {
		err := util.CreateClusterDeployment(userHiveClient, clusterDeploymentName, clusterDeploymentName, "", "", nil, true)
		gomega.Expect(err).Should(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing user create/update clusterpool with mangedClusterSet label", func() {
	var userName = rand.String(6)
	var clusterPoolName = "e2e-" + userName
	var rbacName = "e2e-" + userName
	var clusterSet1 = "clusterset1-e2e"
	var clusterSet2 = "clusterset2-e2e"
	var userHiveClient client.Client
	ginkgo.BeforeEach(func() {
		var err error
		// create rbac with managedClusterSet/join clusterset-e2e permission for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("create").Groups(clusterv1beta2.GroupName).Resources("managedclustersets/join").Names(clusterSet1, clusterSet2).RuleOrDie(),
			helpers.NewRule("create", "update", "get").Groups(hivev1.HiveAPIGroup).Resources("clusterpools").RuleOrDie(),
			helpers.NewRule("create", "update", "get").Groups(hivev1.HiveAPIGroup).Resources("clusterdeployments").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, rbacName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, rbacName, rbacName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the cluster client
		userHiveClient, err = util.NewHiveClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		var err error
		err = util.CleanClusterPool(hiveClient, clusterPoolName, clusterPoolName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRoleBinding(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteClusterRole(kubeClient, rbacName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should create and update the clusterdeployment successfully", func() {
		// case 1: Normal user cannot create clusterdeployment without managedClusterSet label
		err := util.CreateClusterPool(userHiveClient, clusterPoolName, clusterPoolName, nil)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		// case 2: Normal user cannot create clusterpool with wrong managedClusterSet label
		wrongLabel := "wrong-clusterSet"

		labels := map[string]string{
			"cluster.open-cluster-management.io/clusterset": wrongLabel,
		}
		err = util.CreateClusterPool(userHiveClient, clusterPoolName, clusterPoolName, labels)
		gomega.Expect(err).Should(gomega.HaveOccurred())

		// case 3: Normal user can create clusterpool with right managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": clusterSet1,
		}
		err = util.CreateClusterPool(userHiveClient, clusterPoolName, clusterPoolName, labels)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// case 4: can not update clusterpool with right managedClusterSet label
		labels = map[string]string{
			"cluster.open-cluster-management.io/clusterset": clusterSet2,
		}
		gomega.Eventually(func() error {
			return util.UpdateClusterPoolLabel(userHiveClient, clusterPoolName, clusterPoolName, labels)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Cluster admin user should fail when updating clusterpool and managedcluster clusterset.", func() {
	var (
		clusterPoolNamespace string
		managedClusterName   string
		managedClusterSet    string
		clusterPool          string
		clusterClaim         string
		err                  error
	)
	ginkgo.BeforeEach(func() {
		managedClusterName = util.RandomName()
		managedClusterSet = util.RandomName()

		clusterPoolNamespace = util.RandomName()
		clusterPool = util.RandomName()
		clusterClaim = util.RandomName()
		err = util.CreateNamespace(clusterPoolNamespace)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterSetLabel := map[string]string{"cluster.open-cluster-management.io/clusterset": managedClusterSet}
		err = util.CreateClusterPool(hiveClient, clusterPool, clusterPoolNamespace, clusterSetLabel)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterClaim(hiveClient, clusterClaim, clusterPoolNamespace, clusterPool)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterDeployment(hiveClient, managedClusterName, managedClusterName, clusterPool, clusterPoolNamespace, nil, false)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.ClaimCluster(hiveClient, managedClusterName, managedClusterName, clusterClaim)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		err = hiveClient.Delete(context.TODO(), &hivev1.ClusterDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      managedClusterName,
				Namespace: managedClusterName,
			},
		})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = hiveClient.Delete(context.TODO(), &hivev1.ClusterClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterClaim,
				Namespace: clusterPoolNamespace,
			},
		})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = hiveClient.Delete(context.TODO(), &hivev1.ClusterPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterPool,
				Namespace: clusterPoolNamespace,
			},
		})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.DeleteNamespace(clusterPoolNamespace)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("try to update clusterpool clusterset label, and it should fail.", func() {
		ginkgo.By("Try to update clusterpool clusterset, and it should fail")
		managedClusterSet1 := util.RandomName()
		clusterSetLabel := map[string]string{
			clusterv1beta2.ClusterSetLabel: managedClusterSet1,
		}
		err = util.UpdateClusterPoolLabel(hiveClient, clusterPool, clusterPoolNamespace, clusterSetLabel)
		gomega.Expect(err).Should(gomega.HaveOccurred())
	})

	ginkgo.It("try to update managedcluster clusterset label, and it should fail.", func() {
		ginkgo.By("Try to update claimed managedcluster clusterset, and it should fail")
		managedClusterSet1 := util.RandomName()
		clusterSetLabel := map[string]string{
			clusterv1beta2.ClusterSetLabel: managedClusterSet1,
		}
		err = util.UpdateManagedClusterLabels(clusterClient, managedClusterName, clusterSetLabel)
		gomega.Expect(err).Should(gomega.HaveOccurred())
	})
})

// var _ = ginkgo.Describe("Testing webhook cert rotation", func() {
// 	var userName = rand.String(6)
// 	var clusterName = "e2e-" + userName
// 	var rbacName = "e2e-" + userName
// 	var userClusterClient clusterclient.Interface
// 	ginkgo.BeforeEach(func() {
// 		var err error
// 		// create rbac with managedClusterSet/join <all> permission for user
// 		rules := []rbacv1.PolicyRule{
// 			helpers.NewRule("create").Groups(clusterv1beta2.GroupName).Resources("managedclustersets/join").RuleOrDie(),
// 			helpers.NewRule("create", "update", "get").Groups(clusterv1.GroupName).Resources("managedclusters").RuleOrDie(),
// 		}
// 		err = util.CreateClusterRole(kubeClient, rbacName, rules)
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 		err = util.CreateClusterRoleBindingForUser(kubeClient, rbacName, rbacName, userName)
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 		// impersonate user to the cluster client
// 		userClusterClient, err = util.NewClusterClientWithImpersonate(userName, nil)
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 	})
// 	ginkgo.AfterEach(func() {
// 		var err error
// 		err = util.CleanManagedCluster(clusterClient, clusterName)
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 		err = util.DeleteClusterRoleBinding(kubeClient, rbacName)
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 		err = util.DeleteClusterRole(kubeClient, rbacName)
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
// 	})

// 	ginkgo.It("should create and update the managedCluster after cert rotation successfully", func() {
// 		// delete secret/signing-key in openshift-service-ca ns to rotate the cert
// 		err := kubeClient.CoreV1().Secrets("openshift-service-ca").Delete(context.TODO(), "signing-key", metav1.DeleteOptions{})
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 		gomega.Eventually(func() error {
// 			_, err := kubeClient.CoreV1().Secrets("openshift-service-ca").Get(context.TODO(), "signing-key", metav1.GetOptions{})
// 			return err
// 		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

// 		err = kubeClient.CoreV1().Secrets(foundationNS).Delete(context.TODO(), "ocm-webhook", metav1.DeleteOptions{})
// 		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

// 		gomega.Eventually(func() error {
// 			_, err := kubeClient.CoreV1().Secrets(foundationNS).Get(context.TODO(), "ocm-webhook", metav1.GetOptions{})
// 			return err
// 		}, eventuallyTimeout, eventuallyInterval*5).ShouldNot(gomega.HaveOccurred())

// 		cluster := util.NewManagedCluster(clusterName)
// 		gomega.Eventually(func() error {
// 			return util.CreateManagedCluster(userClusterClient, cluster)
// 		}, eventuallyTimeout, eventuallyInterval*5).ShouldNot(gomega.HaveOccurred())
// 	})
// })

var _ = ginkgo.Describe("Testing clusterset create and update", func() {
	ginkgo.It("should get global Clusterset successfully", func() {
		gomega.Eventually(func() error {
			_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Get(context.Background(), clustersetutils.GlobalSetName, metav1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should not update global Clusterset successfully", func() {
		updateGlobalSet := clustersetutils.GlobalSet
		updateGlobalSet.Name = "updateset"
		updateGlobalSet.Spec.ClusterSelector.LabelSelector.MatchLabels = map[string]string{
			"vendor": "ocp",
		}
		_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Update(context.Background(), updateGlobalSet, metav1.UpdateOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should not create other labelselector based Clusterset successfully", func() {
		labelSelectorSet := &clusterv1beta2.ManagedClusterSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ocpset",
			},
			Spec: clusterv1beta2.ManagedClusterSetSpec{
				ClusterSelector: clusterv1beta2.ManagedClusterSelector{
					SelectorType: clusterv1beta2.LabelSelector,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"vendor": "openshift",
						},
					},
				},
			},
		}
		_, err := clusterClient.ClusterV1beta2().ManagedClusterSets().Create(context.Background(), labelSelectorSet, metav1.CreateOptions{})
		gomega.Expect(err).To(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing managed cluster deletion", func() {
	var userName = rand.String(6)
	var clusterName = "e2e-" + userName
	ginkgo.It("Only can delete a cluster when it is not a hosting cluster", func() {
		cluster := util.NewManagedCluster(clusterName)
		ginkgo.By(fmt.Sprintf("create a managedCluster %s as the hosting cluster", clusterName))
		gomega.Eventually(func() error {
			err := util.CreateManagedCluster(clusterClient, cluster)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		hostedCluster := util.NewManagedCluster(fmt.Sprintf("e2e-hosted-%s", rand.String(6)))
		hostedCluster.Annotations = map[string]string{
			constants.AnnotationKlusterletDeployMode:         "Hosted",
			constants.AnnotationKlusterletHostingClusterName: clusterName,
		}
		gomega.Eventually(func() error {
			err := util.CreateManagedCluster(clusterClient, hostedCluster)
			if err != nil {
				return fmt.Errorf(err.Error())
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The hosting cluster %s can not be deleted", clusterName))
		err := clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).Should(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("Delete the hosted cluster %s", hostedCluster.Name))
		err = clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), hostedCluster.Name, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		ginkgo.By(fmt.Sprintf("The hosted cluster %s was deleted successfully", hostedCluster.Name))
		gomega.Eventually(func() bool {
			_, err := clusterClient.ClusterV1().ManagedClusters().Get(
				context.TODO(), hostedCluster.Name, metav1.GetOptions{})
			return errors.IsNotFound(err)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		ginkgo.By(fmt.Sprintf("The cluster %s can be deleted now", clusterName))
		err = clusterClient.ClusterV1().ManagedClusters().Delete(
			context.TODO(), clusterName, metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

})
