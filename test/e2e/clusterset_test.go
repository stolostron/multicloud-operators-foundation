package e2e

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var updatedSubject = rbacv1.Subject{
	APIGroup: "rbac.authorization.k8s.io",
	Kind:     "User",
	Name:     "n2",
}

var (
	roleBindingName        string
	clusterroleBindingName string
)

var _ = ginkgo.Describe("Testing ManagedClusterSet", func() {

	ginkgo.Context("Clusterset admin/view clusterrole auto create/delete.", func() {
		ginkgo.It("Clusterset admin/view clusterrole auto create/delete successfully", func() {
			clusterset, err := util.CreateManagedClusterSet(dynamicClient)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				//clusterset-admin clusterrole should be auto created
				adminClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				exist, err := util.HasClusterResource(dynamicClient, util.ClusterRoleGVR, adminClustersetRole)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			gomega.Eventually(func() bool {
				//clusterset-view clusterrole should be auto created
				viewClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "view")
				exist, err := util.HasClusterResource(dynamicClient, util.ClusterRoleGVR, viewClustersetRole)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//delete clusterset
			err = util.DeleteClusterResource(dynamicClient, util.ManagedClusterSetGVR, clusterset.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				//clusterset-admin clusterrole should be auto deleted
				adminClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				exist, err := util.HasClusterResource(dynamicClient, util.ClusterRoleGVR, adminClustersetRole)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())

			gomega.Eventually(func() bool {
				//clusterset-view clusterrole should be auto deleted
				viewClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "view")
				exist, err := util.HasClusterResource(dynamicClient, util.ClusterRoleGVR, viewClustersetRole)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("Managedcluster Clusterrolebinding auto create/update.", func() {
		ginkgo.It("clusterrolebinding should be auto updated successfully", func() {
			gomega.Eventually(func() bool {
				clusterroleBindingName = utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "admin")
				exist, err := util.HasClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//update clusterrolebinding subject, and generated clusterrolebinding will be auto updated
			clusterrolebinding, err := util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = util.SetSubjects(clusterrolebinding, updatedSubject)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			_, err = util.UpdateClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterrolebinding)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedClusterrolebinding, err := util.GetClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				subjects, _, err := unstructured.NestedSlice(generatedClusterrolebinding.Object, "subjects")
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				for _, subject := range subjects {
					subjectValue, _ := subject.(map[string]interface{})
					if subjectValue["kind"] == updatedSubject.Kind &&
						subjectValue["name"] == updatedSubject.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})

	})

	ginkgo.Context("ManagedCluster namespace rolebinding auto create/update.", func() {
		ginkgo.It("ManagedCluster namespace rolebinding should be auto created successfully", func() {
			gomega.Eventually(func() bool {
				roleBindingName = utils.GenerateClustersetResourceRoleBindingName("admin")
				exist, err := util.HasResource(dynamicClient, util.RoleBindingGVR, managedClusterName, roleBindingName)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//update clusterrolebinding subject, and generated rrolebinding will be auto updated
			clusterrolebinding, err := util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = util.SetSubjects(clusterrolebinding, updatedSubject)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			_, err = util.UpdateClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterrolebinding)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedRolebinding, err := util.GetResource(dynamicClient, util.RoleBindingGVR, managedClusterName, roleBindingName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				subjects, _, err := unstructured.NestedSlice(generatedRolebinding.Object, "subjects")
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

				for _, subject := range subjects {
					subjectValue, _ := subject.(map[string]interface{})
					if subjectValue["kind"] == updatedSubject.Kind &&
						subjectValue["name"] == updatedSubject.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		})
	})

	ginkgo.Context("Delete ManagedClusterset, managedcluster clusterrolebinding and namespace rolebinding should be auto deleted successfully", func() {
		ginkgo.It("Delete ManagedClusterset, managedcluster clusterrolebinding and namespace rolebinding should be auto deleted successfully", func() {
			//managedcluster clusterrolebinding exist
			gomega.Eventually(func() bool {
				exist, err := util.HasClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//managedcluster namespace rolebinding exist
			gomega.Eventually(func() bool {
				exist, err := util.HasResource(dynamicClient, util.RoleBindingGVR, managedClusterName, roleBindingName)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//Delete clusterset, clusterrolebinding should be auto deleted
			err := util.DeleteClusterResource(dynamicClient, util.ManagedClusterSetGVR, managedClustersetName)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			//managedcluster namespace rolebinding deleted
			gomega.Eventually(func() bool {
				exist, err := util.HasResource(dynamicClient, util.RoleBindingGVR, managedClusterName, roleBindingName)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())

			//managedcluster cluster rolebinding deleted
			gomega.Eventually(func() bool {
				exist, err := util.HasResource(dynamicClient, util.RoleBindingGVR, "", roleBindingName)
				if err != nil {
					return false
				}
				return exist
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())
		})
	})
})
