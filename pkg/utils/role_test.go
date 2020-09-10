package utils

import (
	"testing"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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
		{"test1", args{subjects: []rbacv1.Subject{{Kind: "R1", APIGroup: "G1", Name: "N1"}}, cursubjects: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}, {Kind: "R1", APIGroup: "G1", Name: "N1"}}},
		{"test2", args{cursubjects: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}},
		{"test3", args{subjects: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}},
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
	policyr2 := createPolicyRule([]string{clusterv1alpha1.GroupName}, []string{"*"}, []string{"*"}, []string{"*"})
	policyr3 := createPolicyRule([]string{clusterv1alpha1.GroupName}, []string{"*"}, []string{"*"}, []string{"res1", "res2"})
	policyr4 := createPolicyRule([]string{clusterv1alpha1.GroupName}, []string{"create"}, []string{"managedclustersets/bind"}, []string{"res1", "res2"})

	type args struct {
		rules []rbacv1.PolicyRule
	}
	tests := []struct {
		name string
		args args
		want sets.String
	}{
		{"test1", args{rules: []rbacv1.PolicyRule{}}, sets.NewString()},
		{"test2", args{rules: []rbacv1.PolicyRule{*policyr1}}, sets.NewString("*")},
		{"test3", args{rules: []rbacv1.PolicyRule{*policyr2}}, sets.NewString("*")},
		{"test4", args{rules: []rbacv1.PolicyRule{*policyr3}}, sets.NewString("res1", "res2")},
		{"test5", args{rules: []rbacv1.PolicyRule{*policyr4}}, sets.NewString("res1", "res2")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := GetClustersetInRules(tt.args.rules)
			if !res.Equal(tt.want) {
				t.Errorf("Mergesubjects() = %v, want %v", res, tt.want)
			}
		})
	}
}
