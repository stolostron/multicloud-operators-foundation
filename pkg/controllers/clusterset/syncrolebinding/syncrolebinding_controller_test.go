package syncrolebinding

import (
	"context"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/cache"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

var (
	scheme = runtime.NewScheme()
)

func newTestReconciler(clustersetToNamespace *helpers.ClusterSetMapper, clusterSetCache *cache.AuthCache) *Reconciler {
	cb := generateRequiredRoleBinding("c0", nil, "cs0", "admin")
	objs := []runtime.Object{cb}
	r := &Reconciler{
		kubeClient:            k8sfake.NewSimpleClientset(objs...),
		clustersetToNamespace: clustersetToNamespace,
		clusterSetAdminCache:  clusterSetCache,
		clusterSetViewCache:   clusterSetCache,
	}
	return r
}

func generateclustersetToNamespace(ms map[string]sets.String) *helpers.ClusterSetMapper {
	clustersetToNamespace := helpers.NewClusterSetMapper()
	for s, c := range ms {
		clustersetToNamespace.UpdateClusterSetByObjects(s, c)
	}
	return clustersetToNamespace
}

func TestSyncManagedClusterClusterroleBinding(t *testing.T) {
	ctc1 := generateclustersetToNamespace(nil)

	ms2 := map[string]sets.String{"cs1": sets.NewString("c0", "c1")}
	ctc2 := generateclustersetToNamespace(ms2)

	tests := []struct {
		name                  string
		clustersetToNamespace *helpers.ClusterSetMapper
		clusterSetCache       *cache.AuthCache
		clustersetToSubject   map[string][]rbacv1.Subject
		managedclusterName    string
		exist                 bool
	}{
		{
			name:                  "no clusters:",
			clustersetToNamespace: ctc1,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			managedclusterName: "c0",
			exist:              false,
		},
		{
			name:                  "c0 should be exist:",
			clustersetToNamespace: ctc2,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			managedclusterName: "c0",
			exist:              true,
		},
		{
			name:                  "need create c1:",
			clustersetToNamespace: ctc2,
			clustersetToSubject: map[string][]rbacv1.Subject{
				"cs1": {
					{
						Kind: "k1", APIGroup: "a1", Name: "n1",
					},
				},
			},
			managedclusterName: "c1",
			exist:              true,
		},
	}

	for _, test := range tests {
		ctx := context.Background()
		r := newTestReconciler(test.clustersetToNamespace, test.clusterSetCache)
		r.syncRoleBinding(ctx, test.clustersetToNamespace, test.clustersetToSubject, "admin")
		validateResult(t, r, test.managedclusterName, test.exist)
	}
}

func validateResult(t *testing.T, r *Reconciler, managedclusterName string, exist bool) {
	ctx := context.Background()
	managedclusterRolebinding, _ := r.kubeClient.RbacV1().RoleBindings(managedclusterName).Get(ctx, utils.GenerateClustersetResourceRoleBindingName("admin"), metav1.GetOptions{})
	if exist && managedclusterRolebinding == nil {
		t.Errorf("Failed to apply managedclusterRolebinding")
	}
}
