package cache

import (
	"testing"
	"time"

	clusterfake "github.com/open-cluster-management/api/client/cluster/clientset/versioned/fake"
	clusterv1informers "github.com/open-cluster-management/api/client/cluster/informers/externalversions"
	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	managedClusterList = clusterv1.ManagedClusterList{
		Items: []clusterv1.ManagedCluster{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1", ResourceVersion: "1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster2", ResourceVersion: "2"},
			},
		},
	}

	clusterRoleList = rbacv1.ClusterRoleList{
		Items: []rbacv1.ClusterRole{
			{
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
			{
				ObjectMeta: metav1.ObjectMeta{Name: "role2", ResourceVersion: "2"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"list"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "role3", ResourceVersion: "3"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:         []string{"*"},
						APIGroups:     []string{"cluster.open-cluster-management.io"},
						Resources:     []string{"managedclusters"},
						ResourceNames: []string{"cluster1", "cluster2", "cluster3"},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "role4", ResourceVersion: "3"},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"list"},
						APIGroups: []string{"clusterview.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
					{
						Verbs:     []string{"create"},
						APIGroups: []string{"cluster.open-cluster-management.io"},
						Resources: []string{"managedclusters"},
					},
				},
			},
		},
	}
	clusterRoleBindingList = rbacv1.ClusterRoleBindingList{
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
			{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "rolebinding4", ResourceVersion: "2"},
				Subjects: []rbacv1.Subject{
					{
						Kind:     "Group",
						APIGroup: "rbac.authorization.k8s.io",
						Name:     "group4",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "role4",
				},
			},
		},
	}
)

func newManagedCluster(names ...string) []*clusterv1.ManagedCluster {
	ret := []*clusterv1.ManagedCluster{}
	for _, name := range names {
		ret = append(ret, &clusterv1.ManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}

	return ret
}

func validateClusterCacheListList(clusterList *clusterv1.ManagedClusterList, expectedSet sets.String) bool {
	clusterSet := sets.String{}
	for _, cluster := range clusterList.Items {
		clusterSet.Insert(cluster.Name)
	}
	if clusterSet.Len() != expectedSet.Len() || !clusterSet.HasAll(expectedSet.List()...) {
		return false
	}
	return true
}

func validateError(t *testing.T, err, expectedError error) {
	if expectedError != nil {
		assert.EqualError(t, err, expectedError.Error())
	} else {
		assert.NoError(t, err)
	}
}
func fakeNewClusterCache(stopCh chan struct{}) *ClusterCache {
	fakeKubeClient := fake.NewSimpleClientset(&clusterRoleList, &clusterRoleBindingList)
	informers := informers.NewSharedInformerFactory(fakeKubeClient, 10*time.Minute)
	for key := range clusterRoleBindingList.Items {
		informers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(&clusterRoleBindingList.Items[key])
	}
	for key := range clusterRoleList.Items {
		informers.Rbac().V1().ClusterRoles().Informer().GetIndexer().Add(&clusterRoleList.Items[key])
	}
	informers.Start(stopCh)
	fakeClusterClient := clusterfake.NewSimpleClientset(&managedClusterList)
	clusterInformers := clusterv1informers.NewSharedInformerFactory(fakeClusterClient, 10*time.Minute)
	for key := range managedClusterList.Items {
		clusterInformers.Cluster().V1().ManagedClusters().Informer().GetIndexer().Add(&managedClusterList.Items[key])
	}
	clusterInformers.Start(stopCh)

	return NewClusterCache(
		clusterInformers.Cluster().V1().ManagedClusters(),
		informers.Rbac().V1().ClusterRoles(),
		informers.Rbac().V1().ClusterRoleBindings(),
	)

}
func TestClusterCacheList(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)
	clusterCache := fakeNewClusterCache(stopCh)

	tests := []struct {
		name             string
		user             user.Info
		expectedClusters sets.String
		expectedErr      error
	}{
		{
			name: "user1 test cluster1",
			user: &user.DefaultInfo{
				Name:   "user1",
				UID:    "user1-uid",
				Groups: []string{},
			},
			expectedClusters: sets.String{}.Insert("cluster1"),
		},
		{
			name: "group2 test cluster1,2",
			user: &user.DefaultInfo{
				Name:   "",
				UID:    "group2-uid",
				Groups: []string{"group2"},
			},
			expectedClusters: sets.String{}.Insert("cluster1", "cluster2"),
		},
		{
			name: "group3 test no cluster3",
			user: &user.DefaultInfo{
				Name:   "",
				UID:    "group3-uid",
				Groups: []string{"group3"},
			},
			expectedClusters: sets.String{}.Insert("cluster1", "cluster2"),
		},
		{
			name: "group4 no get role",
			user: &user.DefaultInfo{
				Name:   "group4",
				UID:    "group4-uid",
				Groups: []string{"group4"},
			},
			expectedClusters: sets.String{},
		},
		{
			name: "no user4",
			user: &user.DefaultInfo{
				Name:   "user4",
				UID:    "user4-uid",
				Groups: []string{""},
			},
			expectedClusters: sets.String{},
		},
	}
	clusterCache.cache.Synchronize()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clusterList, err := clusterCache.List(test.user, labels.Everything())
			validateError(t, err, test.expectedErr)
			assert.Equal(t, validateClusterCacheListList(clusterList, test.expectedClusters), true)
		})
	}

}
