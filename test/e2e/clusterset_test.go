package e2e

import (
	"context"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var updatedSubjectAdmin = rbacv1.Subject{
	APIGroup: "rbac.authorization.k8s.io",
	Kind:     "User",
	Name:     "admin2",
}
var updatedSubjectView = rbacv1.Subject{
	APIGroup: "rbac.authorization.k8s.io",
	Kind:     "User",
	Name:     "view2",
}

var (
	adminRoleBindingName        string
	viewRoleBindingName         string
	adminClusterroleBindingName string
	viewClusterroleBindingName  string
	clusterDeploymentNamespace  string
)

var _ = ginkgo.Describe("Testing ManagedClusterSet", func() {

	ginkgo.Context("Clusterset admin/view clusterrole auto create/delete.", func() {
		ginkgo.It("Clusterset admin/view clusterrole auto create/delete successfully", func() {
			clusterset, err := clusterClient.ClusterV1alpha1().ManagedClusterSets().Create(context.Background(), util.ManagedClusterSetRandomTemplate, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			gomega.Eventually(func() error {
				//clusterset admin clusterrole should be auto created
				adminClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClustersetRole, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				//clusterset view clusterrole should be auto created
				viewClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClustersetRole, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//delete clusterset
			err = clusterClient.ClusterV1alpha1().ManagedClusterSets().Delete(context.Background(), clusterset.Name, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				//clusterset admin clusterrole should be auto created
				adminClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), adminClustersetRole, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				//clusterset view clusterrole should be auto created
				viewClustersetRole := utils.GenerateClustersetClusterroleName(clusterset.GetName(), "admin")
				_, err := kubeClient.RbacV1().ClusterRoles().Get(context.Background(), viewClustersetRole, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Managedcluster Clusterrolebinding auto create/update.", func() {
		ginkgo.It("clusterrolebinding should be auto updated successfully", func() {
			gomega.Eventually(func() error {
				//clusterset admin clusterrolebinding should be auto created
				adminClusterroleBindingName = utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "admin")
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), adminClusterroleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//update clusterset admin clusterrolebinding subject, and generated clusterrolebinding will be auto updated
			updateAdminClusterrolebinding := util.ClusterRoleBindingAdminTemplate.DeepCopy()
			updatedAdminSubject := append(updateAdminClusterrolebinding.Subjects, updatedSubjectAdmin)
			updateAdminClusterrolebinding.Subjects = updatedAdminSubject

			_, err := kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateAdminClusterrolebinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedClusterrolebinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), adminClusterroleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedClusterrolebinding.Subjects {
					if subject.Kind == updatedSubjectAdmin.Kind &&
						subject.Name == updatedSubjectAdmin.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//Check view clusterrolebinding auto generated
			gomega.Eventually(func() error {
				//clusterset view clusterrolebinding should be auto created
				viewClusterroleBindingName = utils.GenerateClustersetClusterRoleBindingName(managedClusterName, "view")
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), viewClusterroleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//update clusterset view clusterrolebinding subject, and generated clusterrolebinding will be auto updated
			updateClusterrolebinding := util.ClusterRoleBindingViewTemplate.DeepCopy()
			updatedSubject := append(updateClusterrolebinding.Subjects, updatedSubjectView)
			updateClusterrolebinding.Subjects = updatedSubject

			_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(context.Background(), updateClusterrolebinding, metav1.UpdateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedClusterrolebinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), viewClusterroleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedClusterrolebinding.Subjects {
					if subject.Kind == updatedSubjectView.Kind &&
						subject.Name == updatedSubjectView.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

		})

	})

	ginkgo.Context("ManagedCluster namespace rolebinding auto create/update.", func() {
		ginkgo.It("ManagedCluster namespace rolebinding should be auto created successfully", func() {
			//Make sure clusterset admin rolebinding generated.
			gomega.Eventually(func() error {
				adminRoleBindingName = utils.GenerateClustersetResourceRoleBindingName("admin")
				_, err := kubeClient.RbacV1().RoleBindings(managedClusterName).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedRolebinding, err := kubeClient.RbacV1().RoleBindings(managedClusterName).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRolebinding.Subjects {
					if subject.Kind == updatedSubjectAdmin.Kind &&
						subject.Name == updatedSubjectAdmin.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//Make sure clusterset view rolebinding generated.
			gomega.Eventually(func() error {
				viewRoleBindingName = utils.GenerateClustersetResourceRoleBindingName("view")
				_, err := kubeClient.RbacV1().RoleBindings(managedClusterName).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedRolebinding, err := kubeClient.RbacV1().RoleBindings(managedClusterName).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRolebinding.Subjects {
					if subject.Kind == updatedSubjectView.Kind &&
						subject.Name == updatedSubjectView.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("ClusterClaim and clusterdeployment auto add to ManagedClusterset.", func() {
		ginkgo.It("ClusterClaim and clusterdeployment auto add to ManagedClusterset.", func() {
			//create clusterpool in clusterset
			clusterpool, err := hiveClient.HiveV1().ClusterPools(util.ClusterpoolTemplate.Namespace).Create(context.Background(), util.ClusterpoolTemplate, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			//create clustetcalim with clusterpool ref
			clusterclaim, err := hiveClient.HiveV1().ClusterClaims(util.ClusterclaimTemplate.Namespace).Create(context.Background(), util.ClusterclaimTemplate, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			// create a namespace for testing
			ns, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), util.NamespaceRandomTemplate, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			clusterDeploymentNamespace = ns.Name

			//create clusterdeployment with clusterpool ref
			clusterdeployment, err := hiveClient.HiveV1().ClusterDeployments(clusterDeploymentNamespace).Create(context.Background(), util.ClusterdeploymentTemplate, metav1.CreateOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			//Make sure clusterset admin rolebinding in clusterpool namespace generated.
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterpool.Namespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//Make sure clusterset view rolebinding in clusterpool namespace generated.
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterpool.Namespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//Make sure clusterset admin rolebinding in clusterdeployment namespace generated.
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterdeployment.Namespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//Make sure clusterset view rolebinding in clusterdeployment namespace generated.
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterdeployment.Namespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() bool {
				generatedRolebinding, err := kubeClient.RbacV1().RoleBindings(clusterdeployment.Namespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRolebinding.Subjects {
					if subject.Kind == updatedSubjectAdmin.Kind &&
						subject.Name == updatedSubjectAdmin.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			gomega.Eventually(func() bool {
				generatedRolebinding, err := kubeClient.RbacV1().RoleBindings(clusterdeployment.Namespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
				for _, subject := range generatedRolebinding.Subjects {
					if subject.Kind == updatedSubjectView.Kind &&
						subject.Name == updatedSubjectView.Name {
						return true
					}
				}
				return false
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.BeTrue())

			//cliam the clusterdeployment and remove from set
			gomega.Eventually(func() error {
				clusterdeployment, err = hiveClient.HiveV1().ClusterDeployments(clusterdeployment.Namespace).Get(context.Background(), clusterdeployment.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				clusterdeployment.Spec.ClusterPoolRef.ClaimName = "clusterclaim1"
				clusterdeployment.Labels = map[string]string{}
				clusterdeployment, err = hiveClient.HiveV1().ClusterDeployments(clusterdeployment.Namespace).Update(context.Background(), clusterdeployment, metav1.UpdateOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			//clusterset admin/view rolebinding is removed from clusterdeployment namespace
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterdeployment.Namespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterdeployment.Namespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			//clusterset admin/view rolebinding should not be removed from clusterclaim namespace, because clusterpool(ns is same as claim) still in this set
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterclaim.Namespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterclaim.Namespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Delete ManagedClusterset, managedcluster clusterrolebinding and namespace rolebinding should be auto deleted successfully", func() {
		ginkgo.It("Delete ManagedClusterset, managedcluster clusterrolebinding and namespace rolebinding should be auto deleted successfully", func() {
			//Delete clusterset, clusterrolebinding should be auto deleted
			err := clusterClient.ClusterV1alpha1().ManagedClusterSets().Delete(context.Background(), managedClusterSetName, metav1.DeleteOptions{})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

			//managedcluster cluster rolebinding deleted
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), adminClusterroleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), viewClusterroleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			//managedcluster namespace rolebinding deleted
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(managedClusterName).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(managedClusterName).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			//clusterpool namespace rolebinding deleted
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(util.ClusterpoolTemplate.Namespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(util.ClusterpoolTemplate.Namespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())

			//clusterdeployment namespace rolebinding deleted
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterDeploymentNamespace).Get(context.Background(), adminRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
			gomega.Eventually(func() error {
				_, err := kubeClient.RbacV1().RoleBindings(clusterDeploymentNamespace).Get(context.Background(), viewRoleBindingName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, eventuallyTimeout, eventuallyInterval).Should(gomega.HaveOccurred())
		})
	})
})
