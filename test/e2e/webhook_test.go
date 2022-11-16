package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
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
	})
})

var _ = ginkgo.Describe("Testing webhook cert rotation", func() {
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

	ginkgo.It("should create and update the managedCluster after cert rotation successfully", func() {
		// delete secret/signing-key in openshift-service-ca ns to rotate the cert
		err := kubeClient.CoreV1().Secrets("openshift-service-ca").Delete(context.TODO(), "signing-key", metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			_, err := kubeClient.CoreV1().Secrets("openshift-service-ca").Get(context.TODO(), "signing-key", metav1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		err = kubeClient.CoreV1().Secrets(foundationNS).Delete(context.TODO(), "ocm-webhook", metav1.DeleteOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			_, err := kubeClient.CoreV1().Secrets(foundationNS).Get(context.TODO(), "ocm-webhook", metav1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval*5).ShouldNot(gomega.HaveOccurred())

		cluster := util.NewManagedCluster(clusterName)
		gomega.Eventually(func() error {
			return util.CreateManagedCluster(userClusterClient, cluster)
		}, eventuallyTimeout, eventuallyInterval*5).ShouldNot(gomega.HaveOccurred())
	})
})

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
