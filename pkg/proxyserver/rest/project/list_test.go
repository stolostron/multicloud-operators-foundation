package project

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
)

func TestListProjects(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		objName     string
		obj         runtime.Object
		userInfo    user.Info
		expectedLen int
	}{
		{
			name:      "clusterRoleBinding with matching user and kubevirt role",
			namespace: "cluster1",
			objName:   "test-obj",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"clusterRoleBinding": map[string]interface{}{
							"subject": map[string]interface{}{
								"kind": "User",
								"name": "test-user",
							},
							"roleRef": map[string]interface{}{
								"name": "kubevirt.io:admin",
							},
						},
					},
				},
			},
			userInfo: &user.DefaultInfo{
				Name:   "test-user",
				Groups: []string{"group1"},
			},
			expectedLen: 1,
		},
		{
			name:      "roleBindings with matching group and kubevirt role",
			namespace: "cluster2",
			objName:   "test-obj",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"roleBindings": []interface{}{
							map[string]interface{}{
								"subject": map[string]interface{}{
									"kind": "Group",
									"name": "group1",
								},
								"roleRef": map[string]interface{}{
									"name": "kubevirt.io:edit",
								},
								"namespace": "project1",
							},
						},
					},
				},
			},
			userInfo: &user.DefaultInfo{
				Name:   "test-user",
				Groups: []string{"group1"},
			},
			expectedLen: 1,
		},
		{
			name:      "no matching bindings",
			namespace: "cluster3",
			objName:   "test-obj",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"roleBindings": []interface{}{
							map[string]interface{}{
								"subject": map[string]interface{}{
									"kind": "User",
									"name": "other-user",
								},
								"roleRef": map[string]interface{}{
									"name": "kubevirt.io:view",
								},
								"namespace": "project2",
							},
						},
					},
				},
			},
			userInfo: &user.DefaultInfo{
				Name:   "test-user",
				Groups: []string{"group1"},
			},
			expectedLen: 0,
		},
		{
			name:      "invalid roleRef structure",
			namespace: "cluster4",
			objName:   "test-obj",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"roleBindings": []interface{}{
							map[string]interface{}{
								"subject": map[string]interface{}{
									"kind": "User",
									"name": "test-user",
								},
								"roleRef": map[string]interface{}{}, // 缺少 name 字段
							},
						},
					},
				},
			},
			userInfo: &user.DefaultInfo{
				Name:   "test-user",
				Groups: []string{"group1"},
			},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects := listProjects(tt.namespace, tt.objName, tt.obj, tt.userInfo)
			if len(projects) != tt.expectedLen {
				t.Errorf("expected %d projects, got %d", tt.expectedLen, len(projects))
			}
		})
	}
}
