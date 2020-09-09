package helpers

import (
	"reflect"
	"sync"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestClustersetSubjectsMapper_Get(t *testing.T) {
	type fields struct {
		mutex                sync.RWMutex
		clustersetToSubjects map[string][]rbacv1.Subject
	}
	type args struct {
		k string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []rbacv1.Subject
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClustersetSubjectsMapper{
				mutex:                tt.fields.mutex,
				clustersetToSubjects: tt.fields.clustersetToSubjects,
			}
			if got := c.Get(tt.args.k); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClustersetSubjectsMapper.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
