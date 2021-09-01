package cache

import (
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
)

func validateSet(set, expectedSet sets.String) bool {
	if set.Len() != expectedSet.Len() || !set.HasAll(expectedSet.List()...) {
		return false
	}
	return true
}

func TestSyncManagedClusterCache(t *testing.T) {
	stopCh := make(chan struct{})
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

	clusterCache := &ClusterCache{
		clusterLister: clusterInformers.Cluster().V1().ManagedClusters().Lister(),
	}
	autheCache := NewAuthCache(informers.Rbac().V1().ClusterRoles(),
		informers.Rbac().V1().ClusterRoleBindings(),
		"cluster.open-cluster-management.io", "managedclusters",
		clusterInformers.Cluster().V1().ManagedClusters().Informer(),
		clusterCache.ListResources,
		utils.GetViewResourceFromClusterRole,
	)

	autheCache.synchronize()
	tests := []struct {
		name         string
		user         user.Info
		expectedSets sets.String
	}{
		{
			name: "user1 test cluster1",
			user: &user.DefaultInfo{
				Name:   "user1",
				UID:    "user1-uid",
				Groups: []string{},
			},
			expectedSets: sets.String{}.Insert("cluster1"),
		},
		{
			name: "group2 test cluster1,2",
			user: &user.DefaultInfo{
				Name:   "",
				UID:    "group2-uid",
				Groups: []string{"group2"},
			},
			expectedSets: sets.String{}.Insert("cluster1", "cluster2"),
		},
		{
			name: "group3 test no cluster3",
			user: &user.DefaultInfo{
				Name:   "",
				UID:    "group3-uid",
				Groups: []string{"group3"},
			},
			expectedSets: sets.String{}.Insert("cluster1", "cluster2"),
		},
		{
			name: "group4 no get role",
			user: &user.DefaultInfo{
				Name:   "group4",
				UID:    "group4-uid",
				Groups: []string{"group4"},
			},
			expectedSets: sets.String{},
		},
		{
			name: "no user4",
			user: &user.DefaultInfo{
				Name:   "user4",
				UID:    "user4-uid",
				Groups: []string{""},
			},
			expectedSets: sets.String{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, validateSet(autheCache.listNames(test.user), test.expectedSets), true)
		})
	}
	close(stopCh)
}
