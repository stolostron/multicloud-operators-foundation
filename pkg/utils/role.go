package utils

import (
	"context"
	"fmt"
	"reflect"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Mergesubjects(subjects []rbacv1.Subject, cursubjects []rbacv1.Subject) []rbacv1.Subject {
	var subjectmap = make(map[rbacv1.Subject]bool)
	returnSubjects := subjects
	for _, subject := range subjects {
		subjectmap[subject] = true
	}
	for _, cursubject := range cursubjects {
		if _, ok := subjectmap[cursubject]; !ok {
			returnSubjects = append(returnSubjects, cursubject)
		}
	}
	return returnSubjects
}

func GetClustersetInRules(rules []rbacv1.PolicyRule) sets.String {
	clustersetNames := sets.NewString()
	for _, rule := range rules {
		if ContainsString(rule.APIGroups, "*") && ContainsString(rule.Resources, "*") && ContainsString(rule.Verbs, "*") {
			clustersetNames.Insert("*")
		}
		if !ContainsString(rule.APIGroups, clusterv1alpha1.GroupName) {
			continue
		}
		if !ContainsString(rule.Resources, "managedclustersets/bind") && !ContainsString(rule.Resources, "managedclustersets/join") && !ContainsString(rule.Resources, "*") {
			continue
		}

		if !ContainsString(rule.Verbs, "create") && !ContainsString(rule.Verbs, "*") {
			continue
		}
		for _, resourcename := range rule.ResourceNames {
			if resourcename == "*" {
				return sets.NewString("*")
			}
			clustersetNames.Insert(resourcename)
		}
	}
	return clustersetNames
}

func EqualSubjects(subjects1, subjects2 []rbacv1.Subject) bool {
	if len(subjects1) != len(subjects2) {
		return false
	}
	var subjectMap1 = make(map[rbacv1.Subject]bool)
	for _, curSubject := range subjects1 {
		subjectMap1[curSubject] = true
	}

	var subjectMap2 = make(map[rbacv1.Subject]bool)
	for _, curSubject := range subjects2 {
		subjectMap2[curSubject] = true
	}
	return reflect.DeepEqual(subjectMap1, subjectMap2)
}

//ApplyClusterRoleBinding merges objectmeta, requires subjects and role refs
func ApplyClusterRoleBinding(ctx context.Context, client client.Client, required *rbacv1.ClusterRoleBinding) error {
	existing := &rbacv1.ClusterRoleBinding{}
	err := client.Get(ctx, types.NamespacedName{Name: required.Name}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return client.Create(ctx, required)
		}
		return err
	}

	existingCopy := existing.DeepCopy()
	requiredCopy := required.DeepCopy()

	modified := false

	MergeMap(&modified, existingCopy.Labels, requiredCopy.Labels)

	roleRefIsSame := reflect.DeepEqual(existingCopy.RoleRef, requiredCopy.RoleRef)
	subjectsAreSame := EqualSubjects(existingCopy.Subjects, requiredCopy.Subjects)

	if subjectsAreSame && roleRefIsSame && !modified {
		return nil
	}

	existingCopy.Subjects = requiredCopy.Subjects
	existingCopy.RoleRef = requiredCopy.RoleRef
	return client.Update(ctx, existingCopy)
}

//managedcluster admin role
func GenerateClusterRoleName(clusterName, role string) string {
	return fmt.Sprintf("open-cluster-management:%s:%s", role, clusterName)
}

func GenerateClustersetClusterroleName(clustersetName, role string) string {
	return fmt.Sprintf("open-cluster-management:managedclusterset:%s:%s", role, clustersetName)
}

//clusterset clusterrolebinding
func GenerateClusterRoleBindingName(clusterName string) string {
	return fmt.Sprintf("open-cluster-management:clusterset:managedcluster:%s", clusterName)
}

//Delete cluster role
func DeleteClusterRole(kubeClient kubernetes.Interface, clusterRoleName string) error {
	err := kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRoleName, metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

//apply cluster role
func ApplyClusterRole(kubeClient kubernetes.Interface, clusterRoleName string, rules []rbacv1.PolicyRule) error {
	clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			clusterRole = &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterRoleName,
				},
				Rules: rules,
			}
			_, err = kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if !reflect.DeepEqual(clusterRole.Rules, rules) {
		clusterRole.Rules = rules
		_, err := kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{})
		return err
	}
	return nil
}

func BuildClusterRoleName(objName, rule string) string {
	return fmt.Sprintf("open-cluster-management:%s:%s", rule, objName)
}
