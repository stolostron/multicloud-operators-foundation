package syncrolebinding

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
	stopCh := make(chan struct{})
	defer close(stopCh)

	cb := generateRequiredRoleBinding("c0", nil, "cs0", "admin")
	objs := []runtime.Object{cb}

	ctc1 := generateclustersetToNamespace(nil)

	ms2 := map[string]sets.String{"cs1": sets.NewString("c0", "c1")}
	ctc2 := generateclustersetToNamespace(ms2)
	clustersetToClusters := helpers.NewClusterSetMapper()

	globalsetToClusters := helpers.NewClusterSetMapper()

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
		kubeClient := k8sfake.NewSimpleClientset(objs...)
		informers := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
		informers.Rbac().V1().RoleBindings().Informer().GetIndexer().Add(cb)
		informers.Start(stopCh)

		r := NewReconciler(kubeClient, informers.Rbac().V1().RoleBindings().Lister(), informers.Rbac().V1().RoleBindings().Informer().HasSynced, test.clusterSetCache, test.clusterSetCache, globalsetToClusters, clustersetToClusters, test.clustersetToNamespace)
		r.syncRoleBinding(ctx, test.clustersetToNamespace, test.clustersetToSubject, "admin")
		validateResult(t, &r, test.managedclusterName, test.exist)
		r.reconcile(ctx)
	}
}

func validateResult(t *testing.T, r *Reconciler, managedclusterName string, expectExist bool) {
	if !expectExist {
		return // no need to validate
	}
	_, err := r.kubeClient.RbacV1().RoleBindings(managedclusterName).Get(context.Background(), utils.GenerateClustersetResourceRoleBindingName("admin"), metav1.GetOptions{})
	if err != nil {
		t.Errorf("rolebinding %s should be exist, but got error: %v", utils.GenerateClustersetResourceRoleBindingName("admin"), err)
	}
}
