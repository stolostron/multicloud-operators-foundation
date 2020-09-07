package utils

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestMergesubjects(t *testing.T) {
	type args struct {
		subjects    []rbacv1.Subject
		cursubjects []rbacv1.Subject
	}
	tests := []struct {
		name string
		args args
		want []rbacv1.Subject
	}{
		{"test1", args{subjects: []rbacv1.Subject{rbacv1.Subject{Kind: "R1", APIGroup: "G1", Name: "N1"}}, cursubjects: []rbacv1.Subject{rbacv1.Subject{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, []rbacv1.Subject{rbacv1.Subject{Kind: "R2", APIGroup: "G2", Name: "N2"}, rbacv1.Subject{Kind: "R1", APIGroup: "G1", Name: "N1"}}},
		{"test2", args{cursubjects: []rbacv1.Subject{rbacv1.Subject{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, []rbacv1.Subject{rbacv1.Subject{Kind: "R2", APIGroup: "G2", Name: "N2"}}},
		{"test3", args{subjects: []rbacv1.Subject{rbacv1.Subject{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, []rbacv1.Subject{rbacv1.Subject{Kind: "R2", APIGroup: "G2", Name: "N2"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Mergesubjects(tt.args.subjects, tt.args.cursubjects)
			if len(res) != len(tt.want) {
				t.Errorf("Mergesubjects() = %v, want %v", res, tt.want)
			}
		})
	}
}

func createPolicyRule(groups, verbs, res, resnames []string) *rbacv1.PolicyRule {
	return &rbacv1.PolicyRule{
		APIGroups:     groups,
		Verbs:         verbs,
		Resources:     res,
		ResourceNames: resnames,
	}
}

func TestGetClustersetInRules(t *testing.T) {
	policyr1 := createPolicyRule([]string{"*"}, []string{"*"}, []string{"*"}, []string{"*"})
	type args struct {
		rules []rbacv1.PolicyRule
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"test1", args{rules: []rbacv1.PolicyRule{}}, []string{}},
		{"test2", args{rules: []rbacv1.PolicyRule{*policyr1}}, []string{"*"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := GetClustersetInRules(tt.args.rules)
			if len(res) != len(tt.want) {
				t.Errorf("Mergesubjects() = %v, want %v", res, tt.want)
			}
		})
	}
}
