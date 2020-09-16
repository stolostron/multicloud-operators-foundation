package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var clusterSetGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "managedclustersets",
}
var clusterRoleGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "clusterroles",
}
var clusterRoleBindingGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "clusterrolebindings",
}
var updatedSubject = rbacv1.Subject{
	APIGroup:  "rbac.authorization.k8s.io",
	Kind:      "User",
	Namespace: "ns2",
	Name:      "n2",
}

var _ = ginkgo.Describe("Testing ManagedClusterSet", func() {

	var (
		clusterset         *unstructured.Unstructured
		clusterrole        *unstructured.Unstructured
		clusterrolebinding *unstructured.Unstructured
		err                error
	)
	ginkgo.BeforeEach(func() {
		// load object from json util
		clusterset, err = util.LoadResourceFromJSON(util.ManagedClusterSetTemplate)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// create ManagedClusterSet to real cluster
		clusterset, err = util.CreateResource(dynamicClient, clusterSetGVR, clusterset)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", clusterSetGVR.Resource)

		// create clusterrole
		clusterrole, err = util.LoadResourceFromJSON(util.ClusterRoleTemplate)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// create clusterRole to real cluster
		clusterrole, err = util.CreateResource(dynamicClient, clusterRoleGVR, clusterrole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", clusterRoleGVR.Resource)

		//create clusterrolebinding
		clusterrolebinding, err = util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// create clusterRoleBinding to real cluster
		clusterrolebinding, err = util.CreateResource(dynamicClient, clusterRoleBindingGVR, clusterrolebinding)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", clusterRoleBindingGVR.Resource)

	})
	ginkgo.AfterEach(func() {
		//clean up clusterset
		err := util.DeleteClusterResource(dynamicClient, clusterSetGVR, clusterset.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		//clean up clusterrole
		err = util.DeleteClusterResource(dynamicClient, clusterRoleGVR, clusterrole.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		//clean up clusterrolebinding
		err = util.DeleteClusterResource(dynamicClient, clusterRoleBindingGVR, clusterrolebinding.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Clusterrolebinding auto create/update/delete.", func() {

		ginkgo.It("clusterrolebinding should be auto updated successfully", func() {
			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClusterRoleBindingName("cluster1")
				return util.HasClusterResource(dynamicClient, clusterRoleBindingGVR, clusterroleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//update clusterrolebinding subject, and generated clusterrolebinding will be auto updated
			clusterrolebinding, err = util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = util.SetSubjects(clusterrolebinding, updatedSubject)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			clusterrolebinding, err = util.UpdateClusterResource(dynamicClient, clusterRoleBindingGVR, clusterrolebinding)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClusterRoleBindingName("cluster1")
				generatedClusterrolebinding, err := util.GetClusterResource(dynamicClient, clusterRoleBindingGVR, clusterroleBindingName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				subjects, _, err := unstructured.NestedSlice(generatedClusterrolebinding.Object, "subjects")
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				for _, subject := range subjects {
					subjectValue, _ := subject.(map[string]interface{})
					if subjectValue["kind"] == updatedSubject.Kind &&
						subjectValue["name"] == updatedSubject.Name &&
						subjectValue["namespace"] == updatedSubject.Namespace {
						return true, nil
					}
				}
				return false, nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("clusterrolebinding should be auto deleted successfully", func() {
			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClusterRoleBindingName("cluster1")
				return util.HasClusterResource(dynamicClient, clusterRoleBindingGVR, clusterroleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//Delete clusterset, clusterrolebinding should be auto deleted
			err := util.DeleteClusterResource(dynamicClient, clusterSetGVR, clusterset.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClusterRoleBindingName("cluster1")
				return util.HasClusterResource(dynamicClient, clusterRoleBindingGVR, clusterroleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())
		})
	})
})
