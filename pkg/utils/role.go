package utils

import (
	"context"
	"reflect"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ApplyClusterRoleBinding merges objectmeta, requires subjects and role refs
func ApplyClusterRoleBinding(ctx context.Context, kubeClient kubernetes.Interface, required *rbacv1.ClusterRoleBinding) error {
	existing, err := kubeClient.RbacV1().ClusterRoleBindings().Get(ctx, required.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err := kubeClient.RbacV1().ClusterRoleBindings().Create(ctx, required, metav1.CreateOptions{})
			return err
		}
		return err
	}

	existingCopy := existing.DeepCopy()
	requiredCopy := required.DeepCopy()

	modified := false

	MergeMap(&modified, &existingCopy.Labels, requiredCopy.Labels)

	roleRefIsSame := reflect.DeepEqual(existingCopy.RoleRef, requiredCopy.RoleRef)
	subjectsAreSame := EqualSubjects(existingCopy.Subjects, requiredCopy.Subjects)

	if subjectsAreSame && roleRefIsSame && !modified {
		return nil
	}

	existingCopy.Subjects = requiredCopy.Subjects
	existingCopy.RoleRef = requiredCopy.RoleRef
	_, err = kubeClient.RbacV1().ClusterRoleBindings().Update(ctx, existingCopy, metav1.UpdateOptions{})
	return err
}

// ApplyRoleBinding merges objectmeta, requires subjects and role refs
func ApplyRoleBinding(ctx context.Context, kubeClient kubernetes.Interface, required *rbacv1.RoleBinding) error {
	existing, err := kubeClient.RbacV1().RoleBindings(required.Namespace).Get(ctx, required.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.RbacV1().RoleBindings(required.Namespace).Create(ctx, required, metav1.CreateOptions{})
		}
		return err
	}

	existingCopy := existing.DeepCopy()
	requiredCopy := required.DeepCopy()

	modified := false

	MergeMap(&modified, &existingCopy.Labels, requiredCopy.Labels)

	roleRefIsSame := reflect.DeepEqual(existingCopy.RoleRef, requiredCopy.RoleRef)
	subjectsAreSame := EqualSubjects(existingCopy.Subjects, requiredCopy.Subjects)

	if subjectsAreSame && roleRefIsSame && !modified {
		return nil
	}

	existingCopy.Subjects = requiredCopy.Subjects
	existingCopy.RoleRef = requiredCopy.RoleRef
	_, err = kubeClient.RbacV1().RoleBindings(existingCopy.Namespace).Update(ctx, existingCopy, metav1.UpdateOptions{})
	return err
}

// managedcluster admin role
func GenerateClusterRoleName(clusterName, role string) string {
	return "open-cluster-management:" + role + ":" + clusterName
}
func GenerateClustersetClusterroleName(clustersetName, role string) string {
	return "open-cluster-management:managedclusterset:" + role + ":" + clustersetName
}

// clusterset clusterrolebinding
func GenerateClustersetClusterRoleBindingName(clusterName, role string) string {
	return "open-cluster-management:managedclusterset:" + role + ":managedcluster:" + clusterName
}

// clusterset resource rolebinding name
func GenerateClustersetResourceRoleBindingName(role string) string {
	return "open-cluster-management:managedclusterset:" + role
}

// Delete cluster role
func DeleteClusterRole(kubeClient kubernetes.Interface, clusterRoleName string) error {
	err := kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRoleName, metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// apply cluster role
func ApplyClusterRole(kubeClient kubernetes.Interface, requiredClusterrole *rbacv1.ClusterRole) error {
	existingClusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), requiredClusterrole.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), requiredClusterrole, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	if !reflect.DeepEqual(requiredClusterrole.Rules, existingClusterRole.Rules) || !reflect.DeepEqual(requiredClusterrole.Labels, existingClusterRole.Labels) {
		existingClusterRole.Rules = requiredClusterrole.Rules
		existingClusterRole.Labels = requiredClusterrole.Labels
		_, err := kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), existingClusterRole, metav1.UpdateOptions{})
		return err
	}
	return nil
}

func IsManagedClusterClusterrolebinding(clusterrolebindingName, role string) bool {
	requiredName := GenerateClustersetClusterRoleBindingName("", role)
	return strings.HasPrefix(clusterrolebindingName, requiredName)
}

func APIGroupMatches(rule *rbacv1.PolicyRule, requestedGroup string) bool {
	for _, ruleGroup := range rule.APIGroups {
		if ruleGroup == rbacv1.APIGroupAll {
			return true
		}
		if ruleGroup == requestedGroup {
			return true
		}
	}

	return false
}

func ResourceMatches(rule *rbacv1.PolicyRule, combinedRequestedResource, requestedSubresource string) bool {
	for _, ruleResource := range rule.Resources {
		// if everything is allowed, we match
		if ruleResource == rbacv1.ResourceAll {
			return true
		}
		// if we have an exact match, we match
		if ruleResource == combinedRequestedResource {
			return true
		}

		// We can also match a */subresource.
		// if there isn't a subresource, then continue
		if len(requestedSubresource) == 0 {
			continue
		}
		// if the rule isn't in the format */subresource, then we don't match, continue
		if len(ruleResource) == len(requestedSubresource)+2 &&
			strings.HasPrefix(ruleResource, "*/") &&
			strings.HasSuffix(ruleResource, requestedSubresource) {
			return true

		}
	}

	return false
}

func VerbMatches(rule *rbacv1.PolicyRule, requestedVerb string) bool {
	for _, verb := range rule.Verbs {
		if verb == requestedVerb {
			return true
		}
	}

	return false
}

// GetViewResourceFromClusterRole match the "get" permission of resource,
// which means this role has view permission to this resource
func GetViewResourceFromClusterRole(clusterRole *rbacv1.ClusterRole, group, resource string) (sets.String, bool) {
	names := sets.NewString()
	all := false
	for i := range clusterRole.Rules {
		rule := clusterRole.Rules[i]
		if !APIGroupMatches(&rule, group) {
			continue
		}

		if !VerbMatches(&rule, "get") && !VerbMatches(&rule, "list") && !VerbMatches(&rule, "*") {
			continue
		}

		if !ResourceMatches(&rule, resource, "") {
			continue
		}

		if len(rule.ResourceNames) == 0 {
			all = true
			return names, all
		}

		names.Insert(rule.ResourceNames...)
	}
	return names, all
}

// GetAdminResourceFromClusterRole match the "update" permission of resource,
// which means this role has admin permission to this resource
func GetAdminResourceFromClusterRole(clusterRole *rbacv1.ClusterRole, group, resource string) (sets.String, bool) {
	names := sets.NewString()
	all := false
	for i := range clusterRole.Rules {
		rule := clusterRole.Rules[i]
		if !APIGroupMatches(&rule, group) {
			continue
		}

		if !(VerbMatches(&rule, "update") && (VerbMatches(&rule, "get") || VerbMatches(&rule, "list"))) && !VerbMatches(&rule, "*") {
			continue
		}

		if !ResourceMatches(&rule, resource, "") {
			continue
		}
		if len(rule.ResourceNames) == 0 {
			all = true
			return names, all
		}

		names.Insert(rule.ResourceNames...)
	}
	return names, all
}
