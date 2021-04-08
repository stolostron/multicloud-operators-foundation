package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestGetAdminResourceFromClusterRole(t *testing.T) {
	tests := []struct {
		name        string
		clusterrole *rbacv1.ClusterRole
		group       string
		resource    string
		expectedRst sets.String
		expectAll   bool
	}{
		{
			name: "get one cluster",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"update"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{"cluster1"},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusters",
			expectedRst: sets.NewString("cluster1"),
			expectAll:   false,
		},
		{
			name: "get all clusters 1",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"update"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusters",
			expectedRst: sets.NewString("cluster1"),
			expectAll:   true,
		},
		{
			name: "get all clusters 2",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"get", "create", "update"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusters",
			expectedRst: sets.NewString("cluster1"),
			expectAll:   true,
		},
		{
			name: "get all clusters 3",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			group:     "cluster.open-cluster-management.io",
			resource:  "managedclusters",
			expectAll: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			returnset, all := GetAdminResourceFromClusterRole(test.clusterrole, test.group, test.resource)
			if test.expectAll {
				assert.Equal(t, test.expectAll, all)
				return
			}
			assert.Equal(t, test.expectedRst, returnset)
		})
	}
}
func TestGetViewResourceFromClusterRole(t *testing.T) {
	tests := []struct {
		name        string
		clusterrole *rbacv1.ClusterRole
		group       string
		resource    string
		expectedRst sets.String
		expectAll   bool
	}{
		{
			name: "get one cluster",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"get"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{"cluster1"},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusters",
			expectedRst: sets.NewString("cluster1"),
			expectAll:   false,
		},
		{
			name: "get all clusters 1",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"get"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusters",
			expectedRst: sets.NewString("cluster1"),
			expectAll:   true,
		},
		{
			name: "get all clusters 2",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"get", "create", "update"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusters",
			expectedRst: sets.NewString("cluster1"),
			expectAll:   true,
		},
		{
			name: "get all clusters 3",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{"*"},
						Resources: []string{"*"},
					},
				},
			},
			group:     "cluster.open-cluster-management.io",
			resource:  "managedclusters",
			expectAll: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			returnset, all := GetViewResourceFromClusterRole(test.clusterrole, test.group, test.resource)
			if test.expectAll {
				assert.Equal(t, test.expectAll, all)
				return
			}
			assert.Equal(t, test.expectedRst, returnset)
		})
	}
}
func TestResourceMatches(t *testing.T) {
	tests := []struct {
		name        string
		rule        *rbacv1.PolicyRule
		resource    string
		subResource string
		expectedRst bool
	}{
		{
			name: "has resource",
			rule: &rbacv1.PolicyRule{
				Resources: []string{"managedclusters"},
			},
			resource:    "managedclusters",
			subResource: "",
			expectedRst: true,
		},
		{
			name: "has resource and subresource",
			rule: &rbacv1.PolicyRule{
				Resources: []string{"managedclusters", "*/status"},
			},
			resource:    "managedclusters",
			subResource: "status",
			expectedRst: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, ResourceMatches(test.rule, test.resource, test.subResource), test.expectedRst)
		})
	}
}

func TestAPIGroupMatches(t *testing.T) {
	tests := []struct {
		name        string
		rule        *rbacv1.PolicyRule
		group       string
		expectedRst bool
	}{
		{
			name: "has group",
			rule: &rbacv1.PolicyRule{
				APIGroups: []string{"cluster.open-cluster-management.io"},
			},
			group:       "cluster.open-cluster-management.io",
			expectedRst: true,
		},
		{
			name: "has all groups",
			rule: &rbacv1.PolicyRule{
				APIGroups: []string{"*"},
			},
			group:       "",
			expectedRst: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, APIGroupMatches(test.rule, test.group), test.expectedRst)
		})
	}
}
