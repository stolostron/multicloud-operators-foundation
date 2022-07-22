package cache

import (
	"testing"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterinformers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

var (
	managedClusterSetList = clusterv1beta1.ManagedClusterSetList{
		Items: []clusterv1beta1.ManagedClusterSet{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "clusterset1", ResourceVersion: "1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "clusterset2", ResourceVersion: "2"},
			},
		},
	}

	clusterSetRoleList = rbacv1.ClusterRoleList{
		Items: []rbacv1.ClusterRole{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "role1", ResourceVersion: "1"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclustersets"},
					},
					{
						Verbs:         []string{"get"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclustersets"},
						ResourceNames: []string{"clusterset1"},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "role2", ResourceVersion: "2"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclustersets"},
					},
					{
						Verbs:         []string{"get"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclustersets"},
						ResourceNames: []string{"clusterset1", "clusterset2"},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "role3", ResourceVersion: "3"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclustersets"},
					},
					{
						Verbs:         []string{"get"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclustersets"},
						ResourceNames: []string{"clusterset1", "clusterset2", "clusterset3"},
					},
				},
			},
		},
	}
	clusterSetRoleBindingList = rbacv1.ClusterRoleBindingList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items: []rbacv1.ClusterRoleBinding{
			{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "rolebinding1", ResourceVersion: "1"},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "User",
						APIGroup: "rbac.authorization.k8s.io",
						Name:     "user1",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "role1",
				},
			},
			{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "rolebinding2", ResourceVersion: "2"},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "Group",
						APIGroup: "rbac.authorization.k8s.io",
						Name:     "group2",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "role2",
				},
			},
			{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "rolebinding3", ResourceVersion: "2"},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "Group",
						APIGroup: "rbac.authorization.k8s.io",
						Name:     "group3",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "role3",
				},
			},
		},
	}
)

func newManagedClusterSet(names ...string) []*clusterv1beta1.ManagedClusterSet {
	ret := []*clusterv1beta1.ManagedClusterSet{}
	for _, name := range names {
		ret = append(ret, &clusterv1beta1.ManagedClusterSet{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}

	return ret
}

func validateClusterSetCacheListList(clusterSetList *clusterv1beta1.ManagedClusterSetList, expectedSet sets.String) bool {
	clusterSets := sets.String{}
	for _, clusterSet := range clusterSetList.Items {
		clusterSets.Insert(clusterSet.Name)
	}
	if clusterSets.Len() != expectedSet.Len() || !clusterSets.HasAll(expectedSet.List()...) {
		return false
	}
	return true
}

func fakeNewClusterSetCache(stopCh chan struct{}) *ClusterSetCache {
	fakeKubeClient := fake.NewSimpleClientset(&clusterSetRoleList, &clusterSetRoleBindingList)
	informers := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	for key := range clusterSetRoleBindingList.Items {
		informers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(&clusterSetRoleBindingList.Items[key])
	}
	for key := range clusterSetRoleList.Items {
		informers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&clusterSetRoleList.Items[key])
	}
	informers.Start(stopCh)
	fakeClusterSetClient := clusterfake.NewSimpleClientset(&managedClusterSetList)
	clusterInformers := clusterinformers.NewSharedInformerFactory(fakeClusterSetClient, 10*time.Minute)
	for key := range managedClusterSetList.Items {
		clusterInformers.Cluster().V1beta1().ManagedClusterSets().Informer().GetIndexer().Add(&managedClusterSetList.Items[key])
	}
	clusterInformers.Start(stopCh)

	return NewClusterSetCache(
		clusterInformers.Cluster().V1beta1().ManagedClusterSets(),
		informers.Rbac().V1().ClusterRoles(),
		informers.Rbac().V1().ClusterRoleBindings(),
		utils.GetViewResourceFromClusterRole,
	)

}
func TestClusterSetCacheList(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	clusterSetCache := fakeNewClusterSetCache(stopCh)

	tests := []struct {
		name                string
		user                user.Info
		expectedClusterSets sets.String
		expectedErr         error
	}{
		{
			name: "user1 test clusterset1",
			user: &user.DefaultInfo{
				Name:   "user1",
				UID:    "user1-uid",
				Groups: []string{},
			},
			expectedClusterSets: sets.String{}.Insert("clusterset1"),
		},
		{
			name: "group2 test clusterset1,2",
			user: &user.DefaultInfo{
				Name:   "",
				UID:    "group2-uid",
				Groups: []string{"group2"},
			},
			expectedClusterSets: sets.String{}.Insert("clusterset1", "clusterset2"),
		},
		{
			name: "group3 test no cluster3",
			user: &user.DefaultInfo{
				Name:   "",
				UID:    "group3-uid",
				Groups: []string{"group3"},
			},
			expectedClusterSets: sets.String{}.Insert("clusterset1", "clusterset2"),
		},
		{
			name: "no user4",
			user: &user.DefaultInfo{
				Name:   "user4",
				UID:    "user4-uid",
				Groups: []string{""},
			},
			expectedClusterSets: sets.String{},
		},
	}
	clusterSetCache.Cache.synchronize()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterSetList, err := clusterSetCache.List(test.user, labels.Everything())
			validateError(t, err, test.expectedErr)
			assert.Equal(t, validateClusterSetCacheListList(clusterSetList, test.expectedClusterSets), true)
		})
	}

}

func TestClusterSetCacheListObj(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	clusterSetCache := fakeNewClusterSetCache(stopCh)
	tests := []struct {
		name                string
		user                user.Info
		expectedClusterSets sets.String
		expectedErr         error
	}{
		{
			name: "user1 test clusterset1",
			user: &user.DefaultInfo{
				Name:   "user1",
				UID:    "user1-uid",
				Groups: []string{},
			},
			expectedClusterSets: sets.String{}.Insert("clusterset1"),
		},
	}
	clusterSetCache.Cache.synchronize()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterSetList, err := clusterSetCache.ListObjects(test.user)
			validateError(t, err, test.expectedErr)
			assert.Equal(t, validateClusterSetCacheListList(clusterSetList.(*clusterv1beta1.ManagedClusterSetList), test.expectedClusterSets), true)
		})
	}
}
