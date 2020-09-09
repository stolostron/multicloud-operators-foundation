package clusterrolebinding

import (
	"testing"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme = runtime.NewScheme()
)

func newTestReconciler(roleobjs, rolebindingobjs []runtime.Object) *Reconciler {
	objs := roleobjs
	objs = append(objs, rolebindingobjs...)
	var clusterroleToClusterset = make(map[string]sets.String)
	r := &Reconciler{
		client:                  fake.NewFakeClient(objs...),
		scheme:                  scheme,
		clusterroleToClusterset: clusterroleToClusterset,
		clustersetToSubject:     helpers.NewClustersetSubjectsMapper(),
	}
	return r
}

func TestReconcile(t *testing.T) {

	tests := []struct {
		name                   string
		clusterRoleObjs        []runtime.Object
		clusterRoleBindingObjs []runtime.Object
		expectedMapperData     map[string][]rbacv1.Subject
		req                    reconcile.Request
	}{
		{
			name: "one set in clusterrole",
			clusterRoleObjs: []runtime.Object{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterRole1",
					},
					Rules: []rbacv1.PolicyRule{
						{Verbs: []string{"*"}, APIGroups: []string{"*"}, Resources: []string{"*"}, ResourceNames: []string{"*"}},
					},
				},
			},
			clusterRoleBindingObjs: []runtime.Object{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterRolebinding1",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "clusterRole1",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "k1", APIGroup: "a1", Name: "n1"},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "clusterRole1",
				},
			},
			expectedMapperData: map[string][]rbacv1.Subject{"*": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}},
		},
		{
			name: "two clusterrole",
			clusterRoleObjs: []runtime.Object{
				&rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterRole1",
					},
					Rules: []rbacv1.PolicyRule{
						{Verbs: []string{"create"}, APIGroups: []string{clusterv1alpha1.GroupName}, Resources: []string{"managedclustersets/bind"}, ResourceNames: []string{"s1", "s2"}},
					},
				},
			},
			clusterRoleBindingObjs: []runtime.Object{
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterRolebinding1",
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "clusterRole1",
					},
					Subjects: []rbacv1.Subject{
						{Kind: "k1", APIGroup: "a1", Name: "n1"},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "clusterRole1",
				},
			},
			expectedMapperData: map[string][]rbacv1.Subject{"s1": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}, "s2": []rbacv1.Subject{{Kind: "k1", APIGroup: "a1", Name: "n1"}}},
		},
	}

	for _, test := range tests {
		r := newTestReconciler(test.clusterRoleBindingObjs, test.clusterRoleObjs)
		r.Reconcile(test.req)
		validateResult(t, r, test.expectedMapperData)
	}
}

func validateResult(t *testing.T, r *Reconciler, expectedMapperData map[string][]rbacv1.Subject) {
	mapperData := r.clustersetToSubject.GetMap()
	if len(mapperData) != len(expectedMapperData) {
		t.Errorf("Expect map is not same as result, return Map:%v, expect Map: %v", mapperData, expectedMapperData)
	}
	for clusterSet, subjects := range mapperData {
		if _, ok := expectedMapperData[clusterSet]; !ok {
			t.Errorf("Expect map is not same as result, return Map:%v, expect Map: %v", mapperData, expectedMapperData)
		}
		if !utils.EqualSubjects(expectedMapperData[clusterSet], subjects) {
			t.Errorf("Expect map is not same as result, return Map:%v, expect Map: %v", mapperData, expectedMapperData)
		}
	}
}
