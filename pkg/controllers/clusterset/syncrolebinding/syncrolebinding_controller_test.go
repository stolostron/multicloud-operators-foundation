package syncrolebinding

import (
	"context"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/cache"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	scheme = runtime.NewScheme()
)

func newTestReconciler(clustersetToClusters *helpers.ClusterSetMapper, clusterSetCache *cache.ClusterSetCache) *Reconciler {
	cb := generateRequiredRoleBinding("c0", nil, "admin")
	objs := []runtime.Object{cb}
	r := &Reconciler{
		client:               fake.NewFakeClient(objs...),
		scheme:               scheme,
		clustersetToClusters: clustersetToClusters,
		clusterSetCache:      clusterSetCache,
	}
	return r
}

func generateClustersetToClusters(ms map[string]sets.String) *helpers.ClusterSetMapper {
	clustersetToClusters := helpers.NewClusterSetMapper()
	for s, c := range ms {
		clustersetToClusters.UpdateClusterSetByObjects(s, c)
	}
	return clustersetToClusters
}

func TestSyncManagedClusterClusterroleBinding(t *testing.T) {
	ctc1 := generateClustersetToClusters(nil)

	ms2 := map[string]sets.String{"cs1": sets.NewString("c1", "c2")}
	ctc2 := generateClustersetToClusters(ms2)

	tests := []struct {
		name                   string
		clustersetToClusters   *helpers.ClusterSetMapper
		clusterSetCache        *cache.ClusterSetCache
		clustersetToSubject    map[string][]rbacv1.Subject
		clusterrolebindingName string
		exist                  bool
	}{
		{
			name:                 "no cluster",
			clustersetToClusters: ctc1,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetResourceRoleBindingName("admin"),
			exist:                  false,
		},
		{
			name:                 "delete c0:",
			clustersetToClusters: ctc1,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetResourceRoleBindingName("admin"),
			exist:                  false,
		},
		{
			name:                 "need create:",
			clustersetToClusters: ctc2,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			clusterrolebindingName: utils.GenerateClustersetResourceRoleBindingName("admin"),
			exist:                  true,
		},
	}

	for _, test := range tests {
		ctx := context.Background()
		r := newTestReconciler(test.clustersetToClusters, test.clusterSetCache)
		r.syncRoleBinding(ctx, test.clustersetToClusters, test.clustersetToSubject, "admin")
		validateResult(t, r, test.clusterrolebindingName, test.exist)
	}
}

func validateResult(t *testing.T, r *Reconciler, clusterrolebindingName string, exist bool) {
	ctx := context.Background()
	clusterrolebinding := &rbacv1.ClusterRoleBinding{}
	r.client.Get(ctx, types.NamespacedName{Name: clusterrolebindingName}, clusterrolebinding)
	if exist && clusterrolebinding == nil {
		t.Errorf("Failed to apply clusterrolebinding")
	}
}
