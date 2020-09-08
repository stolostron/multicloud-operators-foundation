package utils

import (
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
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

func GetClustersetInRules(rules []rbacv1.PolicyRule) []string {
	var clustersetNames []string
	for _, rule := range rules {
		if ContainsString(rule.APIGroups, "*") && ContainsString(rule.Resources, "*") && ContainsString(rule.Verbs, "*") {
			return []string{"*"}
		}
		if ContainsString(rule.APIGroups, clusterv1alpha1.GroupName) {
			if ContainsString(rule.Resources, "managedclustersets/bind") {
				if ContainsString(rule.Verbs, "create") {
					for _, resourcename := range rule.ResourceNames {
						clustersetNames = append(clustersetNames, resourcename)
					}
				}
			}
		}
	}
	return clustersetNames
}
