package utils

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
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

func TestEqualSubjects(t *testing.T) {
	type args struct {
		subjects1 []rbacv1.Subject
		subjects2 []rbacv1.Subject
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{subjects1: []rbacv1.Subject{{Kind: "R1", APIGroup: "G1", Name: "N1"}}, subjects2: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, false},
		{"test2", args{subjects1: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, false},
		{"test2", args{subjects2: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, false},
		{"test3", args{subjects1: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}, subjects2: []rbacv1.Subject{{Kind: "R2", APIGroup: "G2", Name: "N2"}}}, true},
		{"test4", args{
			subjects1: []rbacv1.Subject{
				{Kind: "R2", APIGroup: "G2", Name: "N2"},
				{Kind: "R1", APIGroup: "G1", Name: "N1"},
				{Kind: "R3", APIGroup: "G3", Name: "N3"},
			},
			subjects2: []rbacv1.Subject{
				{Kind: "R3", APIGroup: "G3", Name: "N3"},
				{Kind: "R1", APIGroup: "G1", Name: "N1"},
				{Kind: "R2", APIGroup: "G2", Name: "N2"},
			},
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualSubjects(tt.args.subjects1, tt.args.subjects2); got != tt.want {
				t.Errorf("EqualSubjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createClusterrolebinding(name, roleName string, labels map[string]string, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: subjects,
	}
}

func TestApplyClusterRoleBinding(t *testing.T) {
	ctx := context.Background()
	var objs []runtime.Object
	stopCh := make(chan struct{})
	defer close(stopCh)

	var labels = make(map[string]string)
	rb1 := createClusterrolebinding("crb1", GenerateClusterRoleName("r1", "admin"), labels, []rbacv1.Subject{})
	rb2 := createClusterrolebinding("crb2", GenerateClusterRoleName("r2", "admin"), labels, []rbacv1.Subject{})
	rb3 := createClusterrolebinding("crb3", GenerateClusterRoleName("r3", "admin"), labels, []rbacv1.Subject{})

	objs = append(objs, rb1, rb3)

	client := k8sfake.NewSimpleClientset(objs...)
	informers := informers.NewSharedInformerFactory(client, 10*time.Minute)
	informers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(rb1)
	informers.Start(stopCh)

	err := ApplyClusterRoleBinding(ctx, client, informers.Rbac().V1().ClusterRoleBindings().Lister(), rb1)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
	applied := verifyApply(ctx, client, rb1)
	if !applied {
		t.Errorf("Error to apply clusterolebinding.")
	}

	err = ApplyClusterRoleBinding(ctx, client, informers.Rbac().V1().ClusterRoleBindings().Lister(), rb2)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
	applied = verifyApply(ctx, client, rb2)
	if !applied {
		t.Errorf("Error to apply clusterolebinding.")
	}

	rb1Copy := rb1.DeepCopy()
	rb1Copy.Subjects = []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: "user1"}}
	err = ApplyClusterRoleBinding(ctx, client, informers.Rbac().V1().ClusterRoleBindings().Lister(), rb1Copy)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
	applied = verifyApply(ctx, client, rb1Copy)
	if !applied {
		t.Errorf("Error to apply clusterolebinding.")
	}

	err = ApplyClusterRoleBinding(ctx, client, informers.Rbac().V1().ClusterRoleBindings().Lister(), rb3)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
}

func verifyApply(ctx context.Context, client kubernetes.Interface, required *rbacv1.ClusterRoleBinding) bool {
	existing, err := client.RbacV1().ClusterRoleBindings().Get(ctx, required.Name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	if !reflect.DeepEqual(existing.RoleRef, required.RoleRef) {
		return false
	}
	if !EqualSubjects(existing.Subjects, required.Subjects) {
		return false
	}
	return true
}

func TestIsManagedClusterClusterrolebinding(t *testing.T) {
	type args struct {
		rolebindingName string
		role            string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{rolebindingName: "not:hanlde", role: "admin"}, false},
		{"test2", args{rolebindingName: "open-cluster-management:managedclusterset:admin:managedcluster:managedcluster1", role: "admin"}, true},
		{"test3", args{rolebindingName: "open-cluster-management:managedclusterset:view:managedcluster:managedcluster1", role: "false"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := IsManagedClusterClusterrolebinding(tt.args.rolebindingName, tt.args.role)
			if res != tt.want {
				t.Errorf("Failed to test IsManagedClusterClusterrolebinding, rolebinding name: %v, role: %v, want: %v", tt.args.rolebindingName, tt.args.role, tt.want)
			}
		})
	}
}

func TestContainsSubject(t *testing.T) {
	type args struct {
		rolebindingName string
		role            string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{rolebindingName: "not:hanlde", role: "admin"}, false},
		{"test2", args{rolebindingName: "open-cluster-management:managedclusterset:admin:managedcluster:managedcluster1", role: "admin"}, true},
		{"test3", args{rolebindingName: "open-cluster-management:managedclusterset:view:managedcluster:managedcluster1", role: "false"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := IsManagedClusterClusterrolebinding(tt.args.rolebindingName, tt.args.role)
			if res != tt.want {
				t.Errorf("Failed to test IsManagedClusterClusterrolebinding, rolebinding name: %v, role: %v, want: %v", tt.args.rolebindingName, tt.args.role, tt.want)
			}
		})
	}
}

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
						Verbs:         []string{"update", "get"},
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
						Verbs:         []string{"update", "get"},
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
						Verbs:         []string{"list", "create", "update"},
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
		{
			name: "resource type do not match",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"list", "create", "update"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusterset",
			expectedRst: sets.NewString(),
			expectAll:   false,
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
		{
			name: "resource type do not match",
			clusterrole: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"list", "create", "update"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			group:       "cluster.open-cluster-management.io",
			resource:    "managedclusterset",
			expectedRst: sets.NewString(),
			expectAll:   false,
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

func createRolebinding(name, namespace, roleName string, labels map[string]string, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     roleName,
		},
		Subjects: subjects,
	}
}

func TestApplyRoleBinding(t *testing.T) {
	var objs []runtime.Object
	ctx := context.Background()

	stopCh := make(chan struct{})
	defer close(stopCh)

	var labels = make(map[string]string)
	rb1 := createRolebinding("crb1", "local-cluster", "r1", labels, []rbacv1.Subject{})
	rb2 := createRolebinding("crb2", "local-cluster", "r2", labels, []rbacv1.Subject{})
	rb3 := createRolebinding("crb3", "local-cluster", "r2", labels, []rbacv1.Subject{})
	objs = append(objs, rb1, rb3)
	client := k8sfake.NewSimpleClientset(objs...)
	informers := informers.NewSharedInformerFactory(client, 10*time.Minute)
	informers.Rbac().V1().RoleBindings().Informer().GetIndexer().Add(rb1)
	informers.Start(stopCh)

	err := ApplyRoleBinding(ctx, client, informers.Rbac().V1().RoleBindings().Lister(), rb1)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
	applied := verifyAppliedRolebinding(ctx, client, rb1)
	if !applied {
		t.Errorf("Error to apply clusterolebinding.")
	}

	err = ApplyRoleBinding(ctx, client, informers.Rbac().V1().RoleBindings().Lister(), rb2)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
	applied = verifyAppliedRolebinding(ctx, client, rb2)
	if !applied {
		t.Errorf("Error to apply clusterolebinding.")
	}

	rb1Copy := rb1.DeepCopy()
	rb1.Subjects = []rbacv1.Subject{{Kind: rbacv1.UserKind, Name: "user1"}}
	err = ApplyRoleBinding(ctx, client, informers.Rbac().V1().RoleBindings().Lister(), rb1Copy)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
	applied = verifyAppliedRolebinding(ctx, client, rb1Copy)
	if !applied {
		t.Errorf("Error to apply clusterolebinding.")
	}

	err = ApplyRoleBinding(ctx, client, informers.Rbac().V1().RoleBindings().Lister(), rb3)
	if err != nil {
		t.Errorf("Error to apply clusterolebinding. Error:%v", err)
	}
}

func verifyAppliedRolebinding(ctx context.Context, client kubernetes.Interface, required *rbacv1.RoleBinding) bool {
	existing, err := client.RbacV1().RoleBindings(required.Namespace).Get(ctx, required.Name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	if !reflect.DeepEqual(existing.RoleRef, required.RoleRef) {
		return false
	}
	if !EqualSubjects(existing.Subjects, required.Subjects) {
		return false
	}
	return true
}

func verifyAppliedClusterRole(client kubernetes.Interface, required *rbacv1.ClusterRole) bool {
	existing, err := client.RbacV1().ClusterRoles().Get(context.TODO(), required.Name, metav1.GetOptions{})
	if err != nil {
		return false
	}
	if !reflect.DeepEqual(existing.Rules, required.Rules) {
		return false
	}
	if !reflect.DeepEqual(existing.Labels, required.Labels) {
		return false
	}
	return true
}

func createClusterRole(name string, labels map[string]string, rules []rbacv1.PolicyRule) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Rules: rules,
	}
}

func TestApplyClusterRole(t *testing.T) {
	c1 := createClusterRole(GenerateClusterRoleName("c1", "admin"), map[string]string{}, []rbacv1.PolicyRule{})
	c2 := createClusterRole(GenerateClusterRoleName("c2", "admin"), map[string]string{}, []rbacv1.PolicyRule{})
	client := k8sfake.NewSimpleClientset(c2)
	err := ApplyClusterRole(client, c1)
	if err != nil {
		t.Errorf("Error to apply clusterole. Error:%v", err)
	}
	applied := verifyAppliedClusterRole(client, c1)
	if !applied {
		t.Errorf("Error to apply clusterole.")
	}

	c2Copy := c2.DeepCopy()
	c2Copy.Labels["k"] = "v"
	c2Copy.Rules = []rbacv1.PolicyRule{{Verbs: []string{"*"}}}
	err = ApplyClusterRole(client, c2Copy)
	if err != nil {
		t.Errorf("Error to apply clusterole. Error:%v", err)
	}
	applied = verifyAppliedClusterRole(client, c2Copy)
	if !applied {
		t.Errorf("Error to apply clusterole.")
	}

	err = DeleteClusterRole(client, c1.Name)
	if err != nil {
		t.Errorf("Error to delete clusterole. Error:%v", err)
	}

	err = DeleteClusterRole(client, "test2")
	if err != nil {
		t.Errorf("Error to delete clusterole. Error:%v", err)
	}
}
