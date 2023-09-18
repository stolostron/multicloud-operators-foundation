package e2e

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

var (
	managedClusterViewGVR = schema.GroupVersionResource{
		Group:    "clusterview.open-cluster-management.io",
		Version:  "v1",
		Resource: "managedclusters",
	}
	managedClusterSetViewGVR = schema.GroupVersionResource{
		Group:    "clusterview.open-cluster-management.io",
		Version:  "v1alpha1",
		Resource: "managedclustersets",
	}
	clusterRoleNamePostfix        = "-ViewClusterRole"
	clusterRoleBindingNamePostfix = "-ViewClusterRoleBinding"
)

func validateClusterView(UserDynamicClient dynamic.Interface, ViewGVR, resourceGVR schema.GroupVersionResource, expectedNames []string) error {
	resourceList, err := util.ListResource(UserDynamicClient, ViewGVR, "", "")
	if err != nil {
		return fmt.Errorf("validateClusterView: failed to List Resource %v", err)
	}

	if len(resourceList) != len(expectedNames) {
		return fmt.Errorf("validateClusterView: reources count %v != expected count %v, resources: %+v", len(resourceList), len(expectedNames), resourceList)
	}
	for _, item := range resourceList {
		name, _, err := unstructured.NestedString(item.Object, "metadata", "name")
		if err != nil {
			return fmt.Errorf("validateClusterView: failed to get resource name %v", err)
		}
		exist := false
		for _, expectedName := range expectedNames {
			if name == expectedName {
				exist = true
				break
			}
		}
		if !exist {
			return fmt.Errorf("validateClusterView: resource %v is not in expected resource list %v", name, expectedNames)
		}

		rsExisted, err := util.HasResource(dynamicClient, resourceGVR, "", name)
		if err != nil {
			return fmt.Errorf("validateClusterView: failed to get resource %v. err:%v", name, err)
		}
		if !rsExisted {
			return fmt.Errorf("validateClusterView: no resource %v", name)
		}
	}

	return nil
}

var _ = ginkgo.Describe("Testing ClusterView to get managedClusters", func() {
	var userName = rand.String(6)
	var clusterRoleName = userName + clusterRoleNamePostfix
	var clusterRoleBindingName = userName + clusterRoleBindingNamePostfix
	var cluster1 = util.RandomName()
	var cluster2 = util.RandomName()
	var cluster3 = util.RandomName()
	var userDynamicClient dynamic.Interface
	var err error
	ginkgo.BeforeEach(func() {
		// create clusterRole and clusterRoleBinding for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingName, clusterRoleName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// prepare 3 clusters
		err = util.ImportManagedCluster(clusterClient, cluster1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.ImportManagedCluster(clusterClient, cluster2)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.ImportManagedCluster(clusterClient, cluster3)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// cleanup clusters
		err = util.CleanManagedCluster(clusterClient, cluster1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CleanManagedCluster(clusterClient, cluster2)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CleanManagedCluster(clusterClient, cluster3)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	// all cases are running in order with the same clusterRole and clusterRoleBinding of the user
	ginkgo.It("should list the managedClusters.clusterview successfully", func() {
		ginkgo.By("authorize cluster1, cluster2 to user")
		expectedClusters := []string{cluster1, cluster2}
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").Names(expectedClusters...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("append cluster3 to user role")
		expectedClusters = []string{cluster1, cluster2, cluster3}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").Names(expectedClusters...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("delete cluster2 to user")
		expectedClusters = []string{cluster1, cluster3}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").Names(expectedClusters...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("delete cluster3")
		err = util.CleanManagedCluster(clusterClient, cluster3)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		expectedClusters = []string{cluster1}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").Names(expectedClusters...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("delete clusterRole")
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		expectedClusters = []string{}
		expectedError := "validateClusterView: failed to List Resource managedclusters.clusterview.open-cluster-management.io is forbidden: User \"" + userName + "\" cannot list resource \"managedclusters\" in API group \"clusterview.open-cluster-management.io\" at the cluster scope: RBAC: clusterrole.rbac.authorization.k8s.io \"" + userName + clusterRoleNamePostfix + "\" not found"
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(errors.New(expectedError)))

		ginkgo.By("add clusterRole")
		expectedClusters = []string{cluster1, cluster2}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").Names(expectedClusters...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing ClusterView to watch managedClusters", func() {
	var userName = rand.String(6)
	var clusterRoleName = userName + clusterRoleNamePostfix
	var clusterRoleBindingName = userName + clusterRoleBindingNamePostfix
	var clusterName = util.RandomName()
	var userDynamicClient dynamic.Interface
	var err error
	ginkgo.BeforeEach(func() {
		// create clusterRole and clusterRoleBinding for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclusters").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclusters").Names(clusterName).RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingName, clusterRoleName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// cleanup cluster
		err := util.CleanManagedCluster(clusterClient, clusterName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.It("should watch the managedClusters.clusterview successfully", func() {
		var watchedClient watch.Interface

		gomega.Eventually(func() error {
			watchedClient, err = userDynamicClient.Resource(managedClusterViewGVR).Watch(context.Background(), metav1.ListOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		defer watchedClient.Stop()

		go func() {
			time.Sleep(time.Second * 1)
			// prepare 1 cluster
			err = util.ImportManagedCluster(clusterClient, clusterName)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}()

		timeCount := 0
		clusterCount := 0
		expectedClusterCount := 1
		for {
			select {
			case event, ok := <-watchedClient.ResultChan():
				gomega.Expect(ok).Should(gomega.BeTrue())
				if event.Type == watch.Added {
					obj := event.Object.DeepCopyObject()
					cluster := obj.(*unstructured.Unstructured)
					name, _, _ := unstructured.NestedString(cluster.Object, "metadata", "name")
					gomega.Expect(name).Should(gomega.Equal(clusterName))
					clusterCount++
					break
				}
			case <-time.After(1 * time.Second):
				timeCount++
			}
			if expectedClusterCount == clusterCount {
				break
			}
			gomega.Expect(timeCount).ShouldNot(gomega.BeNumerically(">=", 10))
		}
	})
})

var _ = ginkgo.Describe("Testing ClusterView to get managedClusterSets", func() {
	var userName = rand.String(6)
	var clusterRoleName = userName + clusterRoleNamePostfix
	var clusterRoleBindingName = userName + clusterRoleBindingNamePostfix
	var clusterSet1 = util.RandomName()
	var clusterSet2 = util.RandomName()
	var clusterSet3 = util.RandomName()
	var userDynamicClient dynamic.Interface
	var err error
	ginkgo.BeforeEach(func() {
		// create clusterRole and clusterRoleBinding for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingName, clusterRoleName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// prepare 3 clusterSets
		err = util.CreateManagedClusterSet(clusterClient, clusterSet1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CreateManagedClusterSet(clusterClient, clusterSet2)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CreateManagedClusterSet(clusterClient, clusterSet3)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// cleanup clusterSets
		err = util.DeleteManagedClusterSet(clusterClient, clusterSet1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteManagedClusterSet(clusterClient, clusterSet2)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteManagedClusterSet(clusterClient, clusterSet3)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	// all cases are running in order with the same clusterRole and clusterRoleBinding of the user
	ginkgo.It("should list the managedClusterSets.clusterView successfully", func() {
		ginkgo.By("authorize clusterSet1, clusterSet2 to user")
		expectedClusterSets := []string{clusterSet1, clusterSet2}
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").Names(expectedClusterSets...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("append clusterSet3 to user role")
		expectedClusterSets = []string{clusterSet1, clusterSet2, clusterSet3}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").Names(expectedClusterSets...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("delete clusterSet2 in user role")
		expectedClusterSets = []string{clusterSet1, clusterSet3}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").Names(expectedClusterSets...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("delete clusterSet3")
		err = util.DeleteManagedClusterSet(clusterClient, clusterSet3)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterSetInRole := []string{clusterSet1, clusterSet3}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").Names(clusterSetInRole...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		expectedClusterSets = []string{clusterSet1}
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("delete clusterRole")
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		expectedClusterSets = []string{}
		expectedError := "validateClusterView: failed to List Resource managedclustersets.clusterview.open-cluster-management.io is forbidden: User \"" + userName + "\" cannot list resource \"managedclustersets\" in API group \"clusterview.open-cluster-management.io\" at the cluster scope: RBAC: clusterrole.rbac.authorization.k8s.io \"" + userName + clusterRoleNamePostfix + "\" not found"
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(errors.New(expectedError)))

		ginkgo.By("add clusterRole")
		expectedClusterSets = []string{clusterSet1, clusterSet2}
		rules = []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").Names(expectedClusterSets...).RuleOrDie(),
		}
		err = util.UpdateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing ClusterView to watch managedClusterSets", func() {
	var userName = rand.String(6)
	var clusterRoleName = userName + clusterRoleNamePostfix
	var clusterRoleBindingName = userName + clusterRoleBindingNamePostfix
	var clusterSetName = util.RandomName()
	var userDynamicClient dynamic.Interface
	var err error
	ginkgo.BeforeEach(func() {
		// create clusterRole and clusterRoleBinding for user
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "watch").Groups("clusterview.open-cluster-management.io").Resources("managedclustersets").RuleOrDie(),
			helpers.NewRule("list", "get").Groups("cluster.open-cluster-management.io").Resources("managedclustersets").Names(clusterSetName).RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingName, clusterRoleName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// cleanup clusterSet
		err = util.DeleteManagedClusterSet(clusterClient, clusterSetName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.It("should watch the managedClusterSets.clusterView successfully", func() {
		var watchedClient watch.Interface
		var err error

		gomega.Eventually(func() error {
			watchedClient, err = userDynamicClient.Resource(managedClusterSetViewGVR).Watch(context.Background(), metav1.ListOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		defer watchedClient.Stop()

		go func() {
			time.Sleep(time.Second * 1)
			// prepare 1 clusterSet
			err = util.CreateManagedClusterSet(clusterClient, clusterSetName)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}()

		timeCount := 0
		clusterCount := 0
		expectedClusterSetCount := 1
		for {
			select {
			case event, ok := <-watchedClient.ResultChan():
				gomega.Expect(ok).Should(gomega.BeTrue())
				if event.Type == watch.Added {
					obj := event.Object.DeepCopyObject()
					clusterSet := obj.(*unstructured.Unstructured)
					name, _, _ := unstructured.NestedString(clusterSet.Object, "metadata", "name")
					gomega.Expect(name).Should(gomega.Equal(clusterSetName))
					clusterCount++
					break
				}
			case <-time.After(1 * time.Second):
				timeCount++
			}
			if expectedClusterSetCount == clusterCount {
				break
			}
			gomega.Expect(timeCount).ShouldNot(gomega.BeNumerically(">=", 10))
		}
	})
})
