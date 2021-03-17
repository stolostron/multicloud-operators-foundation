package e2e

import (
	"context"
	"errors"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"time"
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

func generateClusterRoleWithClusters(userName string, clusters []string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": userName + clusterRoleNamePostfix,
			},
			"rules": []map[string]interface{}{
				{
					"apiGroups": []interface{}{
						"clusterview.open-cluster-management.io",
					},
					"resources": []interface{}{
						"managedclusters",
					},
					"resourceNames": []interface{}{},
					"verbs": []interface{}{
						"list", "watch",
					},
				},
				{
					"apiGroups": []interface{}{
						"cluster.open-cluster-management.io",
					},
					"resources": []interface{}{
						"managedclusters",
					},
					"resourceNames": clusters,
					"verbs": []interface{}{
						"list", "get",
					},
				},
			},
		},
	}
}

func generateClusterRoleWithClusterSets(userName string, clusterSets []string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRole",
			"metadata": map[string]interface{}{
				"name": userName + clusterRoleNamePostfix,
			},
			"rules": []map[string]interface{}{
				{
					"apiGroups": []interface{}{
						"clusterview.open-cluster-management.io",
					},
					"resources": []interface{}{
						"managedclustersets",
					},
					"resourceNames": []interface{}{},
					"verbs": []interface{}{
						"list", "watch",
					},
				},
				{
					"apiGroups": []interface{}{
						"cluster.open-cluster-management.io",
					},
					"resources": []interface{}{
						"managedclustersets",
					},
					"resourceNames": clusterSets,
					"verbs": []interface{}{
						"list", "get",
					},
				},
			},
		},
	}
}

func generateClusterRoleBinding(userName, clusterRole string) *unstructured.Unstructured {
	clusterRoleBinding := userName + clusterRoleBindingNamePostfix
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]interface{}{
				"name": clusterRoleBinding,
			},
			"roleRef": map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     clusterRole,
			},
			"subjects": []map[string]interface{}{
				{
					"kind":     "User",
					"apiGroup": "rbac.authorization.k8s.io",
					"name":     userName,
				},
			},
		},
	}
}

func createClusterRoleAndRoleBinding(userName string) {
	clusterRole := generateClusterRoleWithClusters(userName, []string{""})
	_, err := util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	clusterRoleBinding := generateClusterRoleBinding(userName, clusterRole.GetName())
	_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterRoleBinding)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func deleteClusterRoleAndRoleBinding(userName string) {
	clusterRoleName := userName + clusterRoleNamePostfix
	clusterRoleBindingName := userName + clusterRoleBindingNamePostfix
	err := util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRoleName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterRoleBindingName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func createClusterSetRoleAndRoleBinding(userName string) {
	clusterRole := generateClusterRoleWithClusterSets(userName, []string{""})
	_, err := util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	clusterRoleBinding := generateClusterRoleBinding(userName, clusterRole.GetName())
	_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterRoleBinding)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func deleteClusterSetRoleAndRoleBinding(userName string) {
	clusterRoleName := userName + clusterRoleNamePostfix
	clusterRoleBindingName := userName + clusterRoleBindingNamePostfix
	err := util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRoleName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterRoleBindingName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func validateClusterView(UserDynamicClient dynamic.Interface, ViewGVR, resourceGVR schema.GroupVersionResource, expectedNames []string) error {
	resourceList, err := util.ListResource(UserDynamicClient, ViewGVR, "", "")
	if err != nil {
		return fmt.Errorf("validateClusterView: failed to List Resource %v", err)
	}

	if len(resourceList) != len(expectedNames) {
		return fmt.Errorf("validateClusterView: reources count %v != expected count %v", len(resourceList), len(expectedNames))
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
	var userName = "Bob"
	var cluster1, cluster2, cluster3 *unstructured.Unstructured
	var userDynamicClient dynamic.Interface
	ginkgo.BeforeEach(func() {
		var err error
		// create clusterRole and clusterRoleBinding for user
		createClusterRoleAndRoleBinding(userName)

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// prepare 3 clusters
		cluster1, err = util.CreateManagedCluster(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		cluster2, err = util.CreateManagedCluster(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		cluster3, err = util.CreateManagedCluster(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		deleteClusterRoleAndRoleBinding(userName)

		// cleanup clusters
		err := util.DeleteManagedCluster(dynamicClient, cluster1.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteManagedCluster(dynamicClient, cluster2.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteManagedCluster(dynamicClient, cluster3.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	// all cases are running in order with the same clusterRole and clusterRoleBinding of the user
	ginkgo.It("should list the managedClusters.clusterview successfully", func() {
		// Case 1: authorize cluster1, cluster2 to user
		expectedClusters := []string{cluster1.GetName(), cluster2.GetName()}
		clusterRole := generateClusterRoleWithClusters(userName, expectedClusters)
		_, err := util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 2: append cluster3 to user role
		expectedClusters = []string{cluster1.GetName(), cluster2.GetName(), cluster3.GetName()}
		clusterRole = generateClusterRoleWithClusters(userName, expectedClusters)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 3: delete cluster2 to user
		expectedClusters = []string{cluster1.GetName(), cluster3.GetName()}
		clusterRole = generateClusterRoleWithClusters(userName, expectedClusters)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 4: delete cluster3
		err = util.DeleteManagedCluster(dynamicClient, cluster3.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterInRole := []string{cluster1.GetName(), cluster3.GetName()}
		clusterRole = generateClusterRoleWithClusters(userName, clusterInRole)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		expectedClusters = []string{cluster1.GetName()}
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 5: delete clusterRole
		err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, userName+clusterRoleNamePostfix)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		expectedClusters = []string{}
		expectedError := "validateClusterView: failed to List Resource managedclusters.clusterview.open-cluster-management.io is forbidden: User \"" + userName + "\" cannot list resource \"managedclusters\" in API group \"clusterview.open-cluster-management.io\" at the cluster scope: RBAC: clusterrole.rbac.authorization.k8s.io \"" + userName + clusterRoleNamePostfix + "\" not found"
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(errors.New(expectedError)))

		// Case 6: add clusterRole
		expectedClusters = []string{cluster1.GetName(), cluster2.GetName()}
		clusterRole = generateClusterRoleWithClusters(userName, expectedClusters)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterViewGVR,
				util.ManagedClusterGVR, expectedClusters)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing ClusterView to watch managedClusters", func() {
	var userName = "Tom"
	var cluster *unstructured.Unstructured
	var userDynamicClient dynamic.Interface
	ginkgo.BeforeEach(func() {
		var err error
		// create clusterRole and clusterRoleBinding for user
		createClusterRoleAndRoleBinding(userName)

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// prepare 1 cluster
		cluster, err = util.CreateManagedCluster(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		deleteClusterRoleAndRoleBinding(userName)

		// cleanup cluster
		err := util.DeleteManagedCluster(dynamicClient, cluster.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.It("should watch the managedClusters.clusterview successfully", func() {
		var watchedClient watch.Interface
		var err error

		gomega.Eventually(func() error {
			watchedClient, err = userDynamicClient.Resource(managedClusterViewGVR).Watch(context.Background(), metav1.ListOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		defer watchedClient.Stop()
		go func() {
			expectedClusters := []string{cluster.GetName()}
			clusterRole := generateClusterRoleWithClusters(userName, expectedClusters)
			_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}()

		timeCount := 0
		clusterCount := 0
		expectedClusterCount := 1
		expectedClusterName := cluster.GetName()
		for {
			select {
			case event, ok := <-watchedClient.ResultChan():
				gomega.Expect(ok).Should(gomega.BeTrue())
				if event.Type == watch.Added {
					obj := event.Object.DeepCopyObject()
					cluster := obj.(*unstructured.Unstructured)
					name, _, _ := unstructured.NestedString(cluster.Object, "metadata", "name")
					gomega.Expect(name).Should(gomega.Equal(expectedClusterName))
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
	var userName = "Tonny"
	var clusterSet1, clusterSet2, clusterSet3 *unstructured.Unstructured
	var userDynamicClient dynamic.Interface
	ginkgo.BeforeEach(func() {
		var err error
		// create clusterRole and clusterRoleBinding for user
		createClusterSetRoleAndRoleBinding(userName)

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// prepare 3 clusterSets
		clusterSet1, err = util.CreateManagedClusterSet(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		clusterSet2, err = util.CreateManagedClusterSet(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		clusterSet3, err = util.CreateManagedClusterSet(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		deleteClusterSetRoleAndRoleBinding(userName)

		// cleanup clusterSets
		err := util.DeleteManagedClusterSet(dynamicClient, clusterSet1.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteManagedClusterSet(dynamicClient, clusterSet2.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteManagedClusterSet(dynamicClient, clusterSet3.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	// all cases are running in order with the same clusterRole and clusterRoleBinding of the user
	ginkgo.It("should list the managedClusterSets.clusterview successfully", func() {
		// Case 1: authorize clusterSet1, clusterSet2 to user
		expectedClusterSets := []string{clusterSet1.GetName(), clusterSet2.GetName()}
		clusterRole := generateClusterRoleWithClusterSets(userName, expectedClusterSets)
		_, err := util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 2: append clusterSet3 to user role
		expectedClusterSets = []string{clusterSet1.GetName(), clusterSet2.GetName(), clusterSet3.GetName()}
		clusterRole = generateClusterRoleWithClusterSets(userName, expectedClusterSets)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 3: delete clusterSet2 in user role
		expectedClusterSets = []string{clusterSet1.GetName(), clusterSet3.GetName()}
		clusterRole = generateClusterRoleWithClusterSets(userName, expectedClusterSets)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 4: delete clusterSet3
		err = util.DeleteManagedClusterSet(dynamicClient, clusterSet3.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		clusterSetInRole := []string{clusterSet1.GetName(), clusterSet3.GetName()}
		clusterRole = generateClusterRoleWithClusterSets(userName, clusterSetInRole)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		expectedClusterSets = []string{clusterSet1.GetName()}
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		// Case 5: delete clusterRole
		err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, userName+clusterRoleNamePostfix)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		expectedClusterSets = []string{}
		expectedError := "validateClusterView: failed to List Resource managedclustersets.clusterview.open-cluster-management.io is forbidden: User \"" + userName + "\" cannot list resource \"managedclustersets\" in API group \"clusterview.open-cluster-management.io\" at the cluster scope: RBAC: clusterrole.rbac.authorization.k8s.io \"" + userName + clusterRoleNamePostfix + "\" not found"
		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Equal(errors.New(expectedError)))

		// Case 6: add clusterRole
		expectedClusterSets = []string{clusterSet1.GetName(), clusterSet2.GetName()}
		clusterRole = generateClusterRoleWithClusterSets(userName, expectedClusterSets)
		_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		gomega.Eventually(func() error {
			return validateClusterView(userDynamicClient, managedClusterSetViewGVR,
				util.ManagedClusterSetGVR, expectedClusterSets)
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})

var _ = ginkgo.Describe("Testing ClusterView to watch managedClusterSets", func() {
	var userName = "Lucy"
	var clusterSet *unstructured.Unstructured
	var userDynamicClient dynamic.Interface
	ginkgo.BeforeEach(func() {
		var err error
		// create clusterRole and clusterRoleBinding for user
		createClusterSetRoleAndRoleBinding(userName)

		// impersonate user to the default kubeConfig
		userDynamicClient, err = util.NewDynamicClientWithImpersonate(userName, nil)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// prepare 1 clusterSet
		clusterSet, err = util.CreateManagedClusterSet(dynamicClient)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.AfterEach(func() {
		// cleanup clusterRole and clusterRoleBinding
		deleteClusterSetRoleAndRoleBinding(userName)

		// cleanup clusterSet
		err := util.DeleteManagedClusterSet(dynamicClient, clusterSet.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})
	ginkgo.It("should watch the managedClusterSets.clusterview successfully", func() {
		var watchedClient watch.Interface
		var err error

		gomega.Eventually(func() error {
			watchedClient, err = userDynamicClient.Resource(managedClusterSetViewGVR).Watch(context.Background(), metav1.ListOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		defer watchedClient.Stop()
		go func() {
			expectedClusterSets := []string{clusterSet.GetName()}
			clusterRole := generateClusterRoleWithClusterSets(userName, expectedClusterSets)
			_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}()

		timeCount := 0
		clusterCount := 0
		expectedClusterSetCount := 1
		expectedClusterSetName := clusterSet.GetName()
		for {
			select {
			case event, ok := <-watchedClient.ResultChan():
				gomega.Expect(ok).Should(gomega.BeTrue())
				if event.Type == watch.Added {
					obj := event.Object.DeepCopyObject()
					clusterSet := obj.(*unstructured.Unstructured)
					name, _, _ := unstructured.NestedString(clusterSet.Object, "metadata", "name")
					gomega.Expect(name).Should(gomega.Equal(expectedClusterSetName))
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
