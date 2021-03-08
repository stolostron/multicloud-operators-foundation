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
	clusterRoleNamePostfix        = "ListClusterRole"
	clusterRoleBindingNamePostfix = "ListClusterRoleBinding"
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

func generateClusterRoleBinding(userName string) *unstructured.Unstructured {
	clusterroleBinding := userName + clusterRoleBindingNamePostfix
	clusterRole := userName + clusterRoleNamePostfix
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "rbac.authorization.k8s.io/v1",
			"kind":       "ClusterRoleBinding",
			"metadata": map[string]interface{}{
				"name": clusterroleBinding,
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
	clusterRole := generateClusterRoleWithClusters(userName, []string{"cluster1"})

	_, err := util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	clusterRoleBinding := generateClusterRoleBinding(userName)
	_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterRoleBinding)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func deleteClusterRoleAndRoleBinding(userName string) {
	clusterRoleName := userName + clusterRoleNamePostfix
	clusterRoleBindingName := userName + clusterRoleBindingNamePostfix
	err := util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRoleName)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRoleBindingName)
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
		expectedError := "validateClusterView: failed to List Resource managedclusters.clusterview.open-cluster-management.io is forbidden: User \"Bob\" cannot list resource \"managedclusters\" in API group \"clusterview.open-cluster-management.io\" at the cluster scope: RBAC: clusterrole.rbac.authorization.k8s.io \"BobListClusterRole\" not found"
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
		watchedClient, err := userDynamicClient.Resource(managedClusterViewGVR).Watch(context.Background(), metav1.ListOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		defer watchedClient.Stop()
		go func() {
			expectedClusters := []string{cluster.GetName()}
			clusterRole := generateClusterRoleWithClusters(userName, expectedClusters)
			_, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterRole)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}()

		for {
			timeCount := 0
			clusterCount := 0
			expectedClusterCount := 1
			expectedClusterName := cluster.GetName()
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
				return
			}
			gomega.Expect(timeCount).ShouldNot(gomega.BeNumerically(">=", 10))
		}
	})
})
