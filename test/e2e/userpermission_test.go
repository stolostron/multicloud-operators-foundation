package e2e

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	clusterviewclient "github.com/stolostron/cluster-lifecycle-api/client/clusterview/clientset/versioned/typed/clusterview/v1alpha1"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/test/e2e/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"
	clusterpermissionv1alpha1 "open-cluster-management.io/cluster-permission/api/v1alpha1"
	cpv1alpha1 "open-cluster-management.io/cluster-permission/client/clientset/versioned/typed/api/v1alpha1"
)

const (
	discoverableClusterRoleLabel = clusterviewv1alpha1.DiscoverableClusterRoleLabel
)

var _ = ginkgo.Describe("Testing UserPermission API", func() {
	var (
		userName               = rand.String(6)
		groupName              = rand.String(6)
		clusterRoleName        = userName + "-ViewClusterRole"
		clusterRoleBindingName = userName + "-ViewClusterRoleBinding"
		discoverableRoleName   = "test-discoverable-" + rand.String(6)
		cluster1               = util.RandomName()
		cluster2               = util.RandomName()
		userPermissionClient   clusterviewclient.UserPermissionInterface
		groupPermissionClient  clusterviewclient.UserPermissionInterface
		cpClient               cpv1alpha1.ApiV1alpha1Interface
		err                    error
	)

	ginkgo.BeforeEach(func() {
		// Create ClusterPermission client
		cfg, err := util.NewKubeConfig()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		cpClient, err = cpv1alpha1.NewForConfig(cfg)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Create discoverable ClusterRole
		discoverableRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: discoverableRoleName,
				Labels: map[string]string{
					discoverableClusterRoleLabel: "true",
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}
		_, err = kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), discoverableRole, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// Create ClusterRole and ClusterRoleBinding for user to access UserPermission API
		rules := []rbacv1.PolicyRule{
			helpers.NewRule("list", "get").Groups("clusterview.open-cluster-management.io").
				Resources("userpermissions").RuleOrDie(),
		}
		err = util.CreateClusterRole(kubeClient, clusterRoleName, rules)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		err = util.CreateClusterRoleBindingForUser(kubeClient, clusterRoleBindingName, clusterRoleName, userName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// Create ClusterRoleBinding for group
		groupBindingName := groupName + "-ViewClusterRoleBinding"
		groupBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: groupBindingName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:     rbacv1.GroupKind,
					APIGroup: rbacv1.GroupName,
					Name:     groupName,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleName,
			},
		}
		_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), groupBinding, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// Create impersonated UserPermission clients
		userConfig := rest.CopyConfig(cfg)
		userConfig.Impersonate.UserName = userName
		userViewClient, err := clusterviewclient.NewForConfig(userConfig)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		userPermissionClient = userViewClient.UserPermissions()

		groupConfig := rest.CopyConfig(cfg)
		groupConfig.Impersonate.UserName = "group-test-user"
		groupConfig.Impersonate.Groups = []string{groupName}
		groupViewClient, err := clusterviewclient.NewForConfig(groupConfig)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		groupPermissionClient = groupViewClient.UserPermissions()

		// Create managed clusters
		err = util.ImportManagedCluster(clusterClient, cluster1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.ImportManagedCluster(clusterClient, cluster2)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// Wait for clusters to be created
		gomega.Eventually(func() error {
			_, err := clusterClient.ClusterV1().ManagedClusters().Get(context.TODO(), cluster1, metav1.GetOptions{})
			return err
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

	})

	ginkgo.AfterEach(func() {
		// Cleanup ClusterPermissions
		err = cpClient.ClusterPermissions(cluster1).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{})
		if err != nil {
			ginkgo.GinkgoLogr.Info("Failed to delete ClusterPermissions", "error", err)
		}

		// Cleanup discoverable ClusterRole
		err = util.DeleteClusterRole(kubeClient, discoverableRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// Cleanup ClusterRole and ClusterRoleBindings
		err = util.DeleteClusterRole(kubeClient, clusterRoleName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, clusterRoleBindingName)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.DeleteClusterRoleBinding(kubeClient, groupName+"-ViewClusterRoleBinding")
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		// Cleanup clusters
		err = util.CleanManagedCluster(clusterClient, cluster1)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		err = util.CleanManagedCluster(clusterClient, cluster2)
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should list user permissions from ClusterPermission with ClusterRoleBinding", func() {
		ginkgo.By("Creating ClusterPermission with ClusterRoleBinding for user")
		clusterPermission := &clusterpermissionv1alpha1.ClusterPermission{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-permission-crb",
				Namespace: cluster1,
			},
			Spec: clusterpermissionv1alpha1.ClusterPermissionSpec{
				ClusterRoleBinding: &clusterpermissionv1alpha1.ClusterRoleBinding{
					RoleRef: &rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     discoverableRoleName,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     rbacv1.UserKind,
							APIGroup: rbacv1.GroupName,
							Name:     userName,
						},
					},
				},
			},
		}
		_, err = cpClient.ClusterPermissions(cluster1).Create(context.TODO(), clusterPermission, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Verifying user can list their permissions")
		gomega.Eventually(func() error {
			userPermList, err := userPermissionClient.List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list userpermissions: %w", err)
			}

			// User should have permission for the discoverable role
			found := false
			for _, item := range userPermList.Items {
				if item.Name == discoverableRoleName {
					found = true
					// Verify bindings exist
					if len(item.Status.Bindings) == 0 {
						return fmt.Errorf("expected bindings, got none")
					}
					break
				}
			}
			if !found {
				return fmt.Errorf("expected permission for role %s not found in list", discoverableRoleName)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Verifying user can get specific permission by name")
		gomega.Eventually(func() error {
			userPerm, err := userPermissionClient.Get(context.TODO(), discoverableRoleName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get userpermission: %w", err)
			}

			// Verify ClusterRoleDefinition exists
			if len(userPerm.Status.ClusterRoleDefinition.Rules) == 0 {
				return fmt.Errorf("expected clusterRoleDefinition rules, got none")
			}

			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should list user permissions from ClusterPermission with RoleBinding", func() {
		ginkgo.By("Creating ClusterPermission with RoleBinding for group")
		clusterPermission := &clusterpermissionv1alpha1.ClusterPermission{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-permission-rb",
				Namespace: cluster1,
			},
			Spec: clusterpermissionv1alpha1.ClusterPermissionSpec{
				RoleBindings: &[]clusterpermissionv1alpha1.RoleBinding{
					{
						Namespace: "default",
						RoleRef: clusterpermissionv1alpha1.RoleRef{
							APIGroup: rbacv1.GroupName,
							Kind:     "ClusterRole",
							Name:     discoverableRoleName,
						},
						Subjects: []rbacv1.Subject{
							{
								Kind:     rbacv1.GroupKind,
								APIGroup: rbacv1.GroupName,
								Name:     groupName,
							},
						},
					},
				},
			},
		}
		_, err = cpClient.ClusterPermissions(cluster1).Create(context.TODO(), clusterPermission, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Verifying group members can list their permissions")
		gomega.Eventually(func() error {
			userPermList, err := groupPermissionClient.List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list userpermissions: %w", err)
			}

			// Group should have permission for the discoverable role
			found := false
			for _, item := range userPermList.Items {
				if item.Name == discoverableRoleName {
					found = true
					// Verify bindings with namespace scope
					if len(item.Status.Bindings) == 0 {
						return fmt.Errorf("expected bindings, got none")
					}

					// Verify scope is namespace-scoped
					if item.Status.Bindings[0].Scope != clusterviewv1alpha1.BindingScopeNamespace {
						return fmt.Errorf("expected scope 'Namespace', got %s", item.Status.Bindings[0].Scope)
					}
					break
				}
			}
			if !found {
				return fmt.Errorf("expected permission for role %s not found in list", discoverableRoleName)
			}
			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should show synthetic managedcluster:admin and managedcluster:view roles", func() {
		ginkgo.By("Creating ClusterRole that grants managedclusteractions create permission")
		adminClusterRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-admin-" + rand.String(6),
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"action.open-cluster-management.io"},
					Resources: []string{"managedclusteractions"},
					Verbs:     []string{"create"},
				},
			},
		}
		_, err = kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), adminClusterRole, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		defer func() {
			_ = kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), adminClusterRole.Name, metav1.DeleteOptions{})
		}()

		ginkgo.By("Creating ClusterRoleBinding that grants the admin role to user")
		adminBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-admin-binding-" + rand.String(6),
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					APIGroup: rbacv1.GroupName,
					Name:     userName,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     adminClusterRole.Name,
			},
		}
		_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), adminBinding, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		defer func() {
			_ = kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), adminBinding.Name, metav1.DeleteOptions{})
		}()

		ginkgo.By("Verifying user has synthetic managedcluster:admin permission")
		gomega.Eventually(func() error {
			userPerm, err := userPermissionClient.Get(context.TODO(), "managedcluster:admin", metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get managedcluster:admin permission: %w", err)
			}

			// Verify synthetic role definition includes wildcard permissions
			if len(userPerm.Status.ClusterRoleDefinition.Rules) == 0 {
				return fmt.Errorf("expected clusterRoleDefinition rules, got none")
			}

			// Verify first rule has wildcard verbs
			if len(userPerm.Status.ClusterRoleDefinition.Rules[0].Verbs) == 0 || userPerm.Status.ClusterRoleDefinition.Rules[0].Verbs[0] != "*" {
				return fmt.Errorf("expected wildcard verbs for admin role, got %v", userPerm.Status.ClusterRoleDefinition.Rules[0].Verbs)
			}

			return nil
		}, eventuallyTimeout, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("should not show permissions the user doesn't have access to", func() {
		ginkgo.By("Creating ClusterPermission for a different user")
		otherUser := "other-" + rand.String(6)
		clusterPermission := &clusterpermissionv1alpha1.ClusterPermission{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-other-user-permission",
				Namespace: cluster1,
			},
			Spec: clusterpermissionv1alpha1.ClusterPermissionSpec{
				ClusterRoleBinding: &clusterpermissionv1alpha1.ClusterRoleBinding{
					RoleRef: &rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     discoverableRoleName,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     rbacv1.UserKind,
							APIGroup: rbacv1.GroupName,
							Name:     otherUser,
						},
					},
				},
			},
		}
		_, err = cpClient.ClusterPermissions(cluster1).Create(context.TODO(), clusterPermission, metav1.CreateOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Verifying original user cannot see other user's permissions")
		gomega.Consistently(func() error {
			userPermList, err := userPermissionClient.List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list userpermissions: %w", err)
			}

			// Original user should not have permission for the discoverable role
			for _, item := range userPermList.Items {
				if item.Name == discoverableRoleName {
					return fmt.Errorf("user should not have permission for role assigned to other user")
				}
			}
			return nil
		}, 10, eventuallyInterval).ShouldNot(gomega.HaveOccurred())
	})
})
