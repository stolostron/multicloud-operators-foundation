package mutating

import (
	"encoding/base64"
	"errors"
	"testing"

	authenticationv1 "k8s.io/api/authentication/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
)

func TestMergeUserIdentityToAnnotations(t *testing.T) {
	testcases := []struct {
		name        string
		userInfo    authenticationv1.UserInfo
		annotations map[string]string
		namespace   string
		listers     rbaclisters.RoleBindingLister
		expected    map[string]string
	}{
		{
			name: "",
			userInfo: authenticationv1.UserInfo{
				Username: "testusername",
				Groups: []string{
					"test",             // not from iam
					"test:test:test",   // not from icp
					"icp:default:test", // is default
					"icp:test:test",    // is not default, but rolebind exist
					"icp:test:nil",     // rolebind not exist
				},
			},
			annotations: map[string]string{
				"origin": "origin",
			},
			namespace: "test",
			listers:   &mockRoleBindingLister{},
			expected: map[string]string{
				"origin":               "origin",
				UserIdentityAnnotation: base64.StdEncoding.EncodeToString([]byte("testusername")),
				UserGroupAnnotation:    base64.StdEncoding.EncodeToString([]byte("test,test:test:test,icp:default:test,icp:test:test")),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := MergeUserIdentityToAnnotations(tc.userInfo, tc.annotations, tc.namespace, tc.listers)
			if len(tc.expected) != len(actual) {
				t.Errorf("expected %d, got %d", len(tc.expected), len(actual))
			}
			for k, v := range tc.expected {
				if actual[k] != v {
					t.Errorf("expected %s, got %s", v, actual[k])
				}
			}
			for k, v := range actual {
				if tc.expected[k] != v {
					t.Errorf("expected %s, got %s", tc.expected[k], v)
				}
			}
		})
	}
}

func (m *mockRoleBindingNamespaceLister) Get(name string) (*rbacv1.RoleBinding, error) {
	if name == "icp:test:test" {
		return nil, nil
	}
	return nil, errors.New("not found")
}

type mockRoleBindingLister struct {
}

type mockRoleBindingNamespaceLister struct {
}

func (m *mockRoleBindingLister) RoleBindings(namespace string) rbaclisters.RoleBindingNamespaceLister {
	return &mockRoleBindingNamespaceLister{}
}

func (m *mockRoleBindingLister) List(selector labels.Selector) (ret []*rbacv1.RoleBinding, err error) {
	return nil, nil
}

func (m *mockRoleBindingNamespaceLister) List(selector labels.Selector) (ret []*rbacv1.RoleBinding, err error) {
	return nil, nil
}
