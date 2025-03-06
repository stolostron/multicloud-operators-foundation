package cache

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
)

func TestIsKubeVirtPermission(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"kubevirt-admin", true},
		{"kubevirt-view", true},
		{"kubevirt-edit", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isKubeVirtPermission(c.name); got != c.expected {
				t.Errorf("isKubeVirtPermission(%q) = %v, want %v", c.name, got, c.expected)
			}
		})
	}
}

func TestListKubeVirtProjects(t *testing.T) {
	cases := []struct {
		name         string
		namespace    string
		obj          runtime.Object
		userInfo     user.Info
		expectedProj []kubeVirtProject
	}{
		{
			name:      "ClusterRoleBinding matches user group",
			namespace: "test-ns",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"clusterRoleBinding": map[string]interface{}{
							"subject": map[string]interface{}{
								"kind": "Group",
								"name": "developers",
							},
						},
					},
				},
			},
			userInfo: &user.DefaultInfo{
				Groups: []string{"developers"},
				Name:   "alice",
			},
			expectedProj: []kubeVirtProject{{name: "any", cluster: "test-ns"}},
		},
		{
			name:      "RoleBinding matches user name",
			namespace: "test-ns",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"roleBindings": []interface{}{
							map[string]interface{}{
								"namespace": "proj-ns",
								"subject": map[string]interface{}{
									"kind": "User",
									"name": "alice",
								},
							},
						},
					},
				},
			},
			userInfo: &user.DefaultInfo{
				Groups: []string{},
				Name:   "alice",
			},
			expectedProj: []kubeVirtProject{{name: "proj-ns", cluster: "test-ns"}},
		},
		{
			name:      "No matching bindings",
			namespace: "test-ns",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"roleBindings": []interface{}{
							map[string]interface{}{
								"namespace": "proj-ns",
								"subject": map[string]interface{}{
									"kind": "User",
									"name": "bob",
								},
							},
						},
					},
				},
			},
			userInfo:     &user.DefaultInfo{Groups: []string{}, Name: "alice"},
			expectedProj: []kubeVirtProject{},
		},
		{
			name:      "Invalid roleBindings structure",
			namespace: "test-ns",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"roleBindings": []interface{}{
							map[string]interface{}{"invalidKey": "value"},
						},
					},
				},
			},
			userInfo:     &user.DefaultInfo{Groups: []string{}, Name: "alice"},
			expectedProj: []kubeVirtProject{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			projects := listKubeVirtProjects(c.namespace, "test-name", c.obj, c.userInfo)
			if len(projects) != len(c.expectedProj) {
				t.Errorf("Expected %d projects, got %d", len(c.expectedProj), len(projects))
			}
			for i, proj := range projects {
				if proj.name != c.expectedProj[i].name || proj.cluster != c.expectedProj[i].cluster {
					t.Errorf("Expected project %v, got %v", c.expectedProj[i], proj)
				}
			}
		})
	}
}
