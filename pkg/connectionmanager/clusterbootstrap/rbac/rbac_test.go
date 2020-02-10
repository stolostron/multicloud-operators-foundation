// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rbac

import (
	"testing"

	rbacv1helpers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common/rbac"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func actionCount(verb, resource string, actions []kubetesting.Action) int {
	testCount := 0
	for _, action := range actions {
		if action.Matches(verb, resource) {
			testCount++
		}
	}

	return testCount
}

func newRole(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: []metav1.OwnerReference{},
		},
		Rules: buildRoleRules(),
	}
}

func newCluster(name, namespace string) *clusterv1alpha1.Cluster {
	return &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newRolebinding(name, namespace string) *rbacv1.RoleBinding {
	binding := rbacv1helpers.NewRoleBinding(
		name,
		namespace).Users("hcm:clusters:" + namespace + ":" + name).BindingOrDie()
	return &binding
}

func TestCreateOrUpdateRole(t *testing.T) {
	fakeKubeClient := kubefake.NewSimpleClientset()
	err := CreateOrUpdateRole(fakeKubeClient, "cluster1", "cluster1", metav1.OwnerReference{})
	if err != nil {
		t.Errorf("error createting role: %v", err)
	}

	count := actionCount("create", "roles", fakeKubeClient.Actions())

	if count != 1 {
		t.Errorf("CreateOrUpdateRole() = %v, want %v", count, 1)
	}

	role := newRole("cluster1", "cluster1")
	fakeKubeClient = kubefake.NewSimpleClientset(role)
	err = CreateOrUpdateRole(fakeKubeClient, "cluster1", "cluster1", metav1.OwnerReference{})
	if err != nil {
		t.Errorf("error createting role: %v", err)
	}

	count = actionCount("create", "roles", fakeKubeClient.Actions())
	updateCount := actionCount("update", "roles", fakeKubeClient.Actions())

	if count != 0 && updateCount != 0 {
		t.Errorf("CreateOrUpdateRole() = %v/%v, want %v/%v", count, updateCount, 0, 0)
	}

	role.Rules = role.Rules[:len(role.Rules)-2]
	fakeKubeClient = kubefake.NewSimpleClientset(role)
	err = CreateOrUpdateRole(fakeKubeClient, "cluster1", "cluster1", metav1.OwnerReference{})
	if err != nil {
		t.Errorf("error createting role: %v", err)
	}

	count = actionCount("create", "roles", fakeKubeClient.Actions())
	updateCount = actionCount("update", "roles", fakeKubeClient.Actions())

	if count != 0 && updateCount != 1 {
		t.Errorf("CreateOrUpdateRole() = %v/%v, want %v/%v", count, updateCount, 0, 1)
	}
}

func TestCreateOrUpdateRoleBinding(t *testing.T) {
	fakeKubeClient := kubefake.NewSimpleClientset()
	err := CreateOrUpdateRoleBinding(fakeKubeClient, "cluster1", "cluster1", metav1.OwnerReference{})
	if err != nil {
		t.Errorf("error createting rolebinding: %v", err)
	}

	count := actionCount("create", "rolebindings", fakeKubeClient.Actions())

	if count != 1 {
		t.Errorf("CreateOrUpdateRoleBinding() = %v, want %v", count, 1)
	}

	rolebinding := newRolebinding("cluster1", "cluster1")
	fakeKubeClient = kubefake.NewSimpleClientset(rolebinding)
	err = CreateOrUpdateRoleBinding(fakeKubeClient, "cluster1", "cluster1", metav1.OwnerReference{})
	if err != nil {
		t.Errorf("error createting rolebinding: %v", err)
	}

	count = actionCount("create", "rolebindings", fakeKubeClient.Actions())
	updateCount := actionCount("update", "rolebindings", fakeKubeClient.Actions())

	if count != 0 && updateCount != 0 {
		t.Errorf("CreateOrUpdateRoleBinding() = %v/%v, want %v/%v", count, updateCount, 0, 0)
	}

	rolebinding.RoleRef.Name = "cluster2"
	fakeKubeClient = kubefake.NewSimpleClientset(rolebinding)
	err = CreateOrUpdateRoleBinding(fakeKubeClient, "cluster1", "cluster1", metav1.OwnerReference{})
	if err != nil {
		t.Errorf("error createting rolebinding: %v", err)
	}

	count = actionCount("create", "rolebindings", fakeKubeClient.Actions())
	updateCount = actionCount("update", "rolebindings", fakeKubeClient.Actions())

	if count != 0 && updateCount != 1 {
		t.Errorf("CreateOrUpdateRoleBinding() = %v/%v, want %v/%v", count, updateCount, 0, 1)
	}
}
