package utils

import (
	"reflect"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func Test_GenerateObjectSubjectMap(t *testing.T) {
	ctc1 := generateClustersetToObjects(nil)

	ms2 := map[string]sets.String{"cs1": sets.NewString("c1", "c2")}
	ctc2 := generateClustersetToObjects(ms2)

	type args struct {
		clustersetToClusters *helpers.ClusterSetMapper
		clustersetToSubject  map[string][]rbacv1.Subject
	}
	tests := []struct {
		name string
		args args
		want map[string][]rbacv1.Subject
	}{
		{
			name: "no clusters:",
			args: args{
				clustersetToClusters: ctc1,
				clustersetToSubject: map[string][]rbacv1.Subject{
					"cs1": {
						{
							Kind: "k1", APIGroup: "a1", Name: "n1",
						},
					},
				},
			},
			want: map[string][]rbacv1.Subject{},
		},
		{
			name: "need create:",
			args: args{
				clustersetToClusters: ctc2,
				clustersetToSubject: map[string][]rbacv1.Subject{
					"cs1": {
						{
							Kind: "k1", APIGroup: "a1", Name: "n1",
						},
					},
				},
			},
			want: map[string][]rbacv1.Subject{
				"c1": {
					{
						Kind:     "k1",
						APIGroup: "a1",
						Name:     "n1",
					},
				},
				"c2": {
					{
						Kind:     "k1",
						APIGroup: "a1",
						Name:     "n1",
					},
				},
			},
		},
		{
			name: "test all:",
			args: args{
				clustersetToClusters: ctc2,
				clustersetToSubject: map[string][]rbacv1.Subject{
					"*": {
						{
							Kind: "k1", APIGroup: "a1", Name: "n1",
						},
					},
				},
			},
			want: map[string][]rbacv1.Subject{
				"c1": {
					{
						Kind:     "k1",
						APIGroup: "a1",
						Name:     "n1",
					},
				},
				"c2": {
					{
						Kind:     "k1",
						APIGroup: "a1",
						Name:     "n1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateObjectSubjectMap(tt.args.clustersetToClusters, tt.args.clustersetToSubject); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateObjectSubjectMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ConvertToClusterSetNamespaceMap(t *testing.T) {
	tests := []struct {
		name string
		cto  map[string]sets.String
		want map[string]sets.String
	}{
		{
			name: "no clusters:",
			cto:  map[string]sets.String{},
			want: map[string]sets.String{},
		},
		{
			name: "claim ns:",
			cto: map[string]sets.String{
				"cs1": sets.NewString("clustercalims/ns1/cc1", "clustercalims/ns2/cc2"),
			},
			want: map[string]sets.String{
				"cs1": sets.NewString("ns1", "ns2"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clustersetToObjectsMapper := generateClustersetToObjects(tt.cto)
			want := generateClustersetToObjects(tt.want)

			if got, _ := ConvertToClusterSetNamespaceMap(clustersetToObjectsMapper); !reflect.DeepEqual(got, want) {
				t.Errorf("ConvertToClusterSetNamespaceMap() = %v, want %v", got, want)
			}
		})
	}
}

func generateClustersetToObjects(ms map[string]sets.String) *helpers.ClusterSetMapper {
	clustersetToClusters := helpers.NewClusterSetMapper()
	for s, c := range ms {
		clustersetToClusters.UpdateClusterSetByObjects(s, c)
	}
	return clustersetToClusters
}
