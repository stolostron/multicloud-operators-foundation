package util

import (
	"context"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func CreateClusterRole(kubeClient kubernetes.Interface, name string, rules []rbacv1.PolicyRule) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: rules,
	}

	_, err := kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
	return err
}

func UpdateClusterRole(kubeClient kubernetes.Interface, name string, rules []rbacv1.PolicyRule) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return CreateClusterRole(kubeClient, name, rules)
		case err != nil:
			return err
		}

		clusterRole.Rules = rules
		_, err = kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{})
		return err
	})

	return err
}

func CreateClusterRoleBindingForUser(kubeClient kubernetes.Interface, name, clusterRole, user string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     user,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
	}

	_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})

	return err
}

func DeleteClusterRoleBinding(kubeClient kubernetes.Interface, name string) error {
	err := kubeClient.RbacV1().ClusterRoleBindings().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func DeleteClusterRole(kubeClient kubernetes.Interface, name string) error {
	err := kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func CreateClusterRoleBinding(kubeClient kubernetes.Interface, name, clusterRole, user string) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     user,
			},
		},
	}
	_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBinding, metav1.CreateOptions{})
	return err
}
