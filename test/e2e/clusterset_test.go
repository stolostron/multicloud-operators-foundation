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
		clusterset, err = util.ApplyClusterResource(dynamicClient, util.ManagedClusterSetGVR, clusterset)
		//clusterset, err = util.CreateResource(dynamicClient, util.ManagedClusterSetGVR, clusterset)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", util.ManagedClusterSetGVR.Resource)

		//set ManagedClusterset for ManagedCluster
		clustersetlabel := map[string]string{
			utils.ClusterSetLabel: clusterset.GetName(),
		}
		gomega.Eventually(func() error {
			managedCluster, err := util.GetClusterResource(dynamicClient, util.ManagedClusterGVR, managedClusterName)
			if err != nil {
				return err
			}
			err = util.AddLabels(managedCluster, clustersetlabel)
			if err != nil {
				return err
			}
			_, err = util.UpdateClusterResource(dynamicClient, util.ManagedClusterGVR, managedCluster)
			return err
		}, eventuallyTimeout, eventuallyInterval).Should(gomega.Succeed())

		// create clusterrole
		clusterrole, err = util.LoadResourceFromJSON(util.ClusterRoleTemplate)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// create clusterRole to real cluster
		clusterrole, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleGVR, clusterrole)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", util.ClusterRoleGVR.Resource)

		//create clusterrolebinding
		clusterrolebinding, err = util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// create clusterRoleBinding to real cluster
		clusterrolebinding, err = util.ApplyClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterrolebinding)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred(), "Failed to create %s", util.ClusterRoleBindingGVR.Resource)

	})
	ginkgo.AfterEach(func() {
		//clean up clusterset
		err := util.DeleteClusterResource(dynamicClient, util.ManagedClusterSetGVR, clusterset.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		//clean up clusterrole
		err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleGVR, clusterrole.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		//clean up clusterrolebinding
		err = util.DeleteClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterrolebinding.GetName())
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.Context("Clusterrolebinding auto create/update/delete.", func() {

		ginkgo.It("clusterrolebinding should be auto updated successfully", func() {
			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "admin")
				return util.HasClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//update clusterrolebinding subject, and generated clusterrolebinding will be auto updated
			clusterrolebinding, err = util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = util.SetSubjects(clusterrolebinding, updatedSubject)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			clusterrolebinding, err = util.UpdateClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterrolebinding)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "admin")
				generatedClusterrolebinding, err := util.GetClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
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
				clusterroleBindingName := utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "admin")
				return util.HasClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//Delete clusterset, clusterrolebinding should be auto deleted
			err := util.DeleteClusterResource(dynamicClient, util.ManagedClusterSetGVR, clusterset.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Eventually(func() (interface{}, error) {
				clusterroleBindingName := utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "admin")
				return util.HasClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterroleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("ManagedCluster namespace rolebinding auto create/update/delete.", func() {
		ginkgo.It("ManagedCluster namespace rolebinding should be auto created successfully", func() {
			gomega.Eventually(func() (interface{}, error) {
				roleBindingName := utils.GenerateClustersetResourceRoleBindingName("admin")
				return util.HasResource(dynamicClient, util.RoleBindingGVR, "", roleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//update clusterrolebinding subject, and generated rrolebinding will be auto updated
			clusterrolebinding, err = util.LoadResourceFromJSON(util.ClusterRoleBindingTemplate)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = util.SetSubjects(clusterrolebinding, updatedSubject)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			clusterrolebinding, err = util.UpdateClusterResource(dynamicClient, util.ClusterRoleBindingGVR, clusterrolebinding)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() (interface{}, error) {
				roleBindingName := utils.GenerateClustersetResourceRoleBindingName("admin")

				generatedRolebinding, err := util.GetResource(dynamicClient, util.RoleBindingGVR, managedClusterName, roleBindingName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				subjects, _, err := unstructured.NestedSlice(generatedRolebinding.Object, "subjects")
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

		ginkgo.It("ManagedCluster namespace rolebinding should be auto deleted successfully", func() {
			gomega.Eventually(func() (interface{}, error) {
				roleBindingName := utils.GenerateClustersetResourceRoleBindingName("admin")
				return util.HasResource(dynamicClient, util.RoleBindingGVR, "", roleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//Delete clusterset, clusterrolebinding should be auto deleted
			err := util.DeleteClusterResource(dynamicClient, util.ManagedClusterSetGVR, clusterset.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Eventually(func() (interface{}, error) {
				roleBindingName := utils.GenerateClustersetResourceRoleBindingName("admin")
				return util.HasResource(dynamicClient, util.RoleBindingGVR, "", roleBindingName)
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("Clusterset admin/view clusterrole auto create/delete.", func() {
		ginkgo.It("Clusterset admin/view clusterrole auto create/delete successfully", func() {
			clusterset, err := util.CreateManagedClusterSet(dynamicClient)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() (interface{}, error) {
				//clusterset-admin clusterrole should be auto created
				adminClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				_, err = util.GetClusterResource(dynamicClient, util.ClusterRoleGVR, adminClustersetRole)
				if err != nil {
					return false, nil
				}
				return true, nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			gomega.Eventually(func() (interface{}, error) {
				//clusterset-view clusterrole should be auto created
				viewClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "view")
				_, err = util.GetClusterResource(dynamicClient, util.ClusterRoleGVR, viewClustersetRole)
				if err != nil {
					return false, nil
				}
				return true, nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//delete clusterset
			err = util.DeleteClusterResource(dynamicClient, util.ManagedClusterSetGVR, clusterset.GetName())
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() (interface{}, error) {
				//clusterset-admin clusterrole should be auto deleted
				adminClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				_, err = util.GetClusterResource(dynamicClient, util.ClusterRoleGVR, adminClustersetRole)
				if err != nil {
					return false, nil
				}
				return true, nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())

			gomega.Eventually(func() (interface{}, error) {
				//clusterset-view clusterrole should be auto deleted
				viewClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "view")
				_, err = util.GetClusterResource(dynamicClient, util.ClusterRoleGVR, viewClustersetRole)
				if err != nil {
					return false, nil
				}
				return true, nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeFalse())
		})
	})
})
