package syncclusterrolebinding

import (
	"context"
	"testing"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

var (
	scheme = runtime.NewScheme()
)

func generateClustersetToClusters(ms map[string]sets.Set[string]) *helpers.ClusterSetMapper {
	clustersetToClusters := helpers.NewClusterSetMapper()
	for s, c := range ms {
		// Convert new generic set to legacy sets.String for compatibility
		legacySet := sets.NewString(c.UnsortedList()...)
		clustersetToClusters.UpdateClusterSetByObjects(s, legacySet)
	}
	return clustersetToClusters
}

func TestSyncManagedClusterClusterroleBinding(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	ca0 := generateRequiredClusterRoleBinding("c0", nil, "cs0", "admin")
	cv0 := generateRequiredClusterRoleBinding("c0", nil, "cs1", "view")
	cv1 := generateRequiredClusterRoleBinding("c1", nil, "cs2", "view")
	objs := []runtime.Object{ca0, cv0, cv1}

	ctc1 := generateClustersetToClusters(nil)

	ms2 := map[string]sets.Set[string]{"cs1": sets.New("c1", "c2")}
	ctc2 := generateClustersetToClusters(ms2)
	gs := map[string]sets.Set[string]{"global": sets.New("c1", "c2")}
	gsm := generateClustersetToClusters(gs)
	tests := []struct {
		name                   string
		clustersetToClusters   *helpers.ClusterSetMapper
		globalsetToClusters    *helpers.ClusterSetMapper
		clusterSetCache        *cache.AuthCache
		clustersetToSubject    map[string][]rbacv1.Subject
		clusterrolebindingName string
		exist                  bool
	}{
		{
			name:                 "no cluster",
			clustersetToClusters: ctc1,
			globalsetToClusters:  gsm,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetClusterRoleBindingName("c1", "admin"),
			exist:                  false,
		},
		{
			name:                 "delete c0:",
			clustersetToClusters: ctc1,
			globalsetToClusters:  gsm,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetClusterRoleBindingName("c0", "admin"),
			exist:                  false,
		},
		{
			name:                 "need create:",
			clustersetToClusters: ctc2,
			globalsetToClusters:  gsm,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetClusterRoleBindingName("c1", "admin"),
			exist:                  true,
		},
	}

	for _, test := range tests {
		ctx := context.Background()
		kubeClient := k8sfake.NewSimpleClientset(objs...)
		informers := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)

		informers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(ca0)
		informers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(cv0)
		informers.Rbac().V1().ClusterRoleBindings().Informer().GetIndexer().Add(cv1)

		informers.Start(stopCh)

		r := NewReconciler(kubeClient, informers.Rbac().V1().ClusterRoleBindings().Lister(), informers.Rbac().V1().ClusterRoleBindings().Informer().HasSynced, test.clusterSetCache, test.clusterSetCache, test.globalsetToClusters, test.clustersetToClusters)
		r.reconcile(ctx)
		r.syncManagedClusterClusterroleBinding(ctx, test.clustersetToClusters, test.clustersetToSubject, "admin")
		validateResult(t, test.name, &r, test.clusterrolebindingName, test.exist)
	}
}

func validateResult(t *testing.T, caseName string, r *Reconciler, clusterrolebindingName string, expectedExist bool) {
	if !expectedExist {
		return // no need to check
	}

	_, err := r.kubeClient.RbacV1().ClusterRoleBindings().Get(context.Background(), clusterrolebindingName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Case: %v, Failed to get clusterrolebinding, err: %v", caseName, err)
	}
}

func Test_getClusterNameInClusterrolebinding(t *testing.T) {
	type args struct {
		clusterrolebindingName string
		role                   string
	}
	tests := []struct {
		name                   string
		clusterrolebindingName string
		want                   string
	}{
		{
			name:                   "right name",
			clusterrolebindingName: "open-cluster-management:managedclusterset:admin:managedcluster:managedcluster1",
			want:                   "managedcluster1",
		},
		{
			name:                   "wrong name",
			clusterrolebindingName: "",
			want:                   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getClusterNameInClusterrolebinding(tt.clusterrolebindingName); got != tt.want {
				t.Errorf("getClusterNameInClusterrolebinding() = %v, want %v", got, tt.want)
			}
		})
	}
}
