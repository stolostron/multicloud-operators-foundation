package utils

import (
	"reflect"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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

func generateMapKey(subject rbacv1.Subject) string {
	return subject.APIGroup + subject.Kind + subject.Name
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
		if !ContainsString(rule.Resources, "managedclustersets/bind") && !ContainsString(rule.Resources, "*") {
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
