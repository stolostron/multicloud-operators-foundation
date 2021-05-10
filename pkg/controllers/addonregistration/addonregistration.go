package addonregistration

import (
	"context"
	"reflect"

	"github.com/open-cluster-management/addon-framework/pkg/agent"
	addonapiv1alpha1 "github.com/open-cluster-management/api/addon/v1alpha1"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/controllers/clusterrbac"
	proxyserverv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apis/proxy/v1beta1"
	certificatesv1 "k8s.io/api/certificates/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type foundationAgent struct {
	kubeClient kubernetes.Interface
	addonName  string
}

func NewAgent(kubeClient kubernetes.Interface, addonName string) *foundationAgent {
	return &foundationAgent{
		kubeClient: kubeClient,
		addonName:  addonName,
	}
}

func (f *foundationAgent) Manifests(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	return []runtime.Object{}, nil
}

func (f *foundationAgent) GetAgentAddonOptions() agent.AgentAddonOptions {
	return agent.AgentAddonOptions{
		AddonName: f.addonName,
		Registration: &agent.RegistrationOption{
			CSRConfigurations: agent.KubeClientSignerConfigurations(f.addonName, f.addonName),
			CSRApproveCheck: func(
				cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn, csr *certificatesv1.CertificateSigningRequest) bool {
				// TODO add more csr check here.
				return true
			},
			PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
				err := f.createOrUpdateRole(cluster.Name)
				if err != nil {
					return err
				}

				return f.createOrUpdateRoleBinding(cluster.Name)
			},
		},
	}
}

// createOrUpdateRole create or update a role for a give cluster
func (f *foundationAgent) createOrUpdateRole(clusterName string) error {
	role, err := f.kubeClient.RbacV1().Roles(clusterName).Get(context.TODO(), roleName(clusterName), metav1.GetOptions{})
	rules := buildRoleRules()
	if err != nil {
		if errors.IsNotFound(err) {
			acmRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName(clusterName),
					Namespace: clusterName,
				},
				Rules: rules,
			}
			_, err = f.kubeClient.RbacV1().Roles(clusterName).Create(context.TODO(), acmRole, metav1.CreateOptions{})
		}
		return err
	}

	if !reflect.DeepEqual(role.Rules, rules) {
		role.Rules = rules
		_, err := f.kubeClient.RbacV1().Roles(clusterName).Update(context.TODO(), role, metav1.UpdateOptions{})
		return err
	}

	return nil
}

// createOrUpdateRoleBinding create or update a role binding for a given cluster
func (f *foundationAgent) createOrUpdateRoleBinding(clusterName string) error {
	roleName := roleName(clusterName)
	groups := agent.DefaultGroups(clusterName, f.addonName)
	acmRoleBinding := clusterrbac.NewRoleBinding(roleName, clusterName).Groups(groups[0]).BindingOrDie()

	// role and rolebinding have the same name
	binding, err := f.kubeClient.RbacV1().RoleBindings(clusterName).Get(context.TODO(), roleName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = f.kubeClient.RbacV1().RoleBindings(clusterName).Create(context.TODO(), &acmRoleBinding, metav1.CreateOptions{})
		}
		return err
	}

	needUpdate := false
	if !reflect.DeepEqual(acmRoleBinding.RoleRef, binding.RoleRef) {
		needUpdate = true
		binding.RoleRef = acmRoleBinding.RoleRef
	}
	if !reflect.DeepEqual(acmRoleBinding.Subjects, binding.Subjects) {
		needUpdate = true
		binding.Subjects = acmRoleBinding.Subjects
	}
	if needUpdate {
		_, err = f.kubeClient.RbacV1().RoleBindings(clusterName).Update(context.TODO(), binding, metav1.UpdateOptions{})
		return err
	}

	return nil
}

func roleName(clusterName string) string {
	return clusterName + ":managed-cluster-foundation"
}

// BuildClusterRoleRules builds the clusteroles
func buildRoleRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		clusterrbac.NewRule("create", "get").Groups(proxyserverv1beta1.GroupName).Resources("clusterstatuses/aggregator").RuleOrDie(),
		clusterrbac.NewRule("get", "list", "watch").Groups(clusterv1beta1.GroupName).Resources("managedclusterinfos").RuleOrDie(),
		clusterrbac.NewRule("update", "patch").Groups(clusterv1beta1.GroupName).Resources("managedclusterinfos/status").RuleOrDie(),
		clusterrbac.NewRule("get", "list", "watch").Groups(actionv1beta1.GroupName).Resources("managedclusteractions").RuleOrDie(),
		clusterrbac.NewRule("update", "patch").Groups(actionv1beta1.GroupName).Resources("managedclusteractions/status").RuleOrDie(),
		clusterrbac.NewRule("get", "list", "watch").Groups(viewv1beta1.GroupName).Resources("managedclusterviews").RuleOrDie(),
		clusterrbac.NewRule("update", "patch").Groups(viewv1beta1.GroupName).Resources("managedclusterviews/status").RuleOrDie(),
		clusterrbac.NewRule("create", "update", "patch").Groups("").Resources("events").RuleOrDie(),
		clusterrbac.NewRule("get", "list", "watch", "create", "update", "delete").Groups("").Resources("secrets").RuleOrDie(),
	}
}
