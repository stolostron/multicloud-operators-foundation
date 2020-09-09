package syncclusterrolebinding

import (
	"reflect"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

/*
func Test_generateClusterToClustersetMap(t *testing.T) {
	type args struct {
		clustersetToCluster map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{"case1", args{clustersetToCluster: map[string]string{"a1": "b1"}}, map[string]string{"b1": "a1"}},
		{"case1", args{clustersetToCluster: map[string]string{"a1": "b1", "a2": "b2"}}, map[string]string{"b1": "a1", "b2": "a2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateClusterToClustersetMap(tt.args.clustersetToCluster); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateClusterToClustersetMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
*/
func Test_generateClusterSubjectMap(t *testing.T) {
	type args struct {
		clustersetToCluster map[string][]string
		clustersetToSubject map[string][]rbacv1.Subject
	}
	tests := []struct {
		name string
		args args
		want map[string][]rbacv1.Subject
	}{
		{"case1", args{clustersetToCluster: map[string][]string{"s1": []string{"c1", "c2"}}, clustersetToSubject: map[string][]rbacv1.Subject{"s1": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}}}, map[string][]rbacv1.Subject{"c1": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}, "c2": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}}},
		{"case1", args{clustersetToCluster: map[string][]string{"s1": []string{"c1"}}, clustersetToSubject: map[string][]rbacv1.Subject{"s3": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}}}, map[string][]rbacv1.Subject{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateClusterSubjectMap(tt.args.clustersetToCluster, tt.args.clustersetToSubject); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateClusterSubjectMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

/*
func Test_shouldUpdate(t *testing.T) {
	type args struct {
		subjects1 []rbacv1.Subject
		subjects2 []rbacv1.Subject
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1", args{subjects1: []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}, {Kind: "k2", APIGroup: "a2", Name: "n2"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUpdate(tt.args.subjects1, tt.args.subjects2); got != tt.want {
				t.Errorf("shouldUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
*/
