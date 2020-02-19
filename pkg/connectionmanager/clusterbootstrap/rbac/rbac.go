// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rbac

import (
	"reflect"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/clusterregistry"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	rbacv1helpers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common/rbac"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var clusterKind = clusterregistry.SchemeGroupVersion.WithKind("Cluster")

// BuildClusterRoleRules builds the clusteroles
func buildRoleRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		rbacv1helpers.NewRule("create", "get", "list", "update").Groups(mcm.GroupName).Resources("clusterstatuses").RuleOrDie(),
		rbacv1helpers.NewRule("create").Groups(mcm.GroupName).Resources("clusterstatuses/topology").RuleOrDie(),
		rbacv1helpers.NewRule("create", "get").Groups(mcm.GroupName).Resources("clusterstatuses/aggregator").RuleOrDie(),
		rbacv1helpers.NewRule("create", "get").Groups(mcm.GroupName).Resources("clusterstatuses/metering-receiver").RuleOrDie(),
		rbacv1helpers.NewRule("get", "list", "watch").Groups(mcm.GroupName).Resources("works").RuleOrDie(),
		rbacv1helpers.NewRule("create").Groups(mcm.GroupName).Resources("works/result").RuleOrDie(),
		rbacv1helpers.NewRule("patch", "update").Groups(mcm.GroupName).Resources("works/status").RuleOrDie(),
		rbacv1helpers.NewRule("create", "get", "update").Groups("clusterregistry.k8s.io").Resources("clusters").RuleOrDie(),
		rbacv1helpers.NewRule("patch", "update").Groups("clusterregistry.k8s.io").Resources("clusters/status").RuleOrDie(),
		rbacv1helpers.NewRule("get", "list", "watch", "update", "patch").Groups("compliance.mcm.ibm.com").Resources("compliances").RuleOrDie(),
		rbacv1helpers.NewRule("get", "list", "watch", "update", "patch").Groups("policy.mcm.ibm.com").Resources("policies").RuleOrDie(),
		rbacv1helpers.NewRule("create", "get", "list", "watch").Groups("").Resources("secrets").RuleOrDie(),
		rbacv1helpers.NewRule("create", "get", "list", "update", "watch", "patch", "delete", "deletecollection").Groups("").Resources("endpoints").RuleOrDie(),
		rbacv1helpers.NewRule("create", "update", "patch").Groups("").Resources("events").RuleOrDie(),
		rbacv1helpers.NewRule("create", "update", "delete").Groups("").Resources("secrets").RuleOrDie(),
		// for deployables
		rbacv1helpers.NewRule("get", "list", "watch").Groups("app.ibm.com").Resources("deployables").RuleOrDie(),
		rbacv1helpers.NewRule("patch", "update").Groups("app.ibm.com").Resources("deployables/status").RuleOrDie(),
	}
}

// CreateOrUpdateRole create or update a role for a give cluster
func CreateOrUpdateRole(kubeclientset kubernetes.Interface, clusterName, clusterNamespace string, owner metav1.OwnerReference) error {
	role, err := kubeclientset.RbacV1().Roles(clusterNamespace).Get(clusterName, metav1.GetOptions{})
	rules := buildRoleRules()
	if errors.IsNotFound(err) {
		hcmRole := &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:            clusterName,
				Namespace:       clusterNamespace,
				OwnerReferences: []metav1.OwnerReference{owner},
			},
			Rules: rules,
		}
		_, createErr := kubeclientset.RbacV1().Roles(clusterNamespace).Create(hcmRole)
		return createErr
	} else if err != nil {
		return err
	}

	if !reflect.DeepEqual(role.Rules, rules) {
		role.Rules = rules
		_, err := kubeclientset.RbacV1().Roles(clusterNamespace).Update(role)
		return err
	}

	return nil
}

// CreateOrUpdateRoleBinding create or update a role binding for a given cluster
func CreateOrUpdateRoleBinding(kubeclientset kubernetes.Interface, clusterName, clusterNamespace string, owner metav1.OwnerReference) error {
	hcmRoleBinding := rbacv1helpers.NewRoleBinding(
		clusterName,
		clusterNamespace).Users("hcm:clusters:" + clusterNamespace + ":" + clusterName).BindingOrDie()
	hcmRoleBinding.OwnerReferences = []metav1.OwnerReference{owner}

	binding, err := kubeclientset.RbacV1().RoleBindings(clusterNamespace).Get(clusterName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = kubeclientset.RbacV1().RoleBindings(clusterNamespace).Create(&hcmRoleBinding)
		return err
	} else if err != nil {
		return err
	}

	needUpdate := false
	if !reflect.DeepEqual(hcmRoleBinding.RoleRef, binding.RoleRef) {
		needUpdate = true
		binding.RoleRef = hcmRoleBinding.RoleRef
	}
	if reflect.DeepEqual(hcmRoleBinding.Subjects, binding.Subjects) {
		needUpdate = true
		binding.Subjects = hcmRoleBinding.Subjects
	}
	if needUpdate {
		_, err = kubeclientset.RbacV1().RoleBindings(clusterNamespace).Update(binding)
		return err
	}

	return nil
}
