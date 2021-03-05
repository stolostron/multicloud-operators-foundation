package rbac

import (
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	"testing"
)

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
