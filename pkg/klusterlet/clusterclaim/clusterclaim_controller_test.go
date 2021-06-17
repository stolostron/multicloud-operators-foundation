package clusterclaim

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"testing"

	tlog "github.com/go-logr/logr/testing"
	clusterclientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterfake "github.com/open-cluster-management/api/client/cluster/clientset/versioned/fake"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newClusterClaimReconciler(clusterClient clusterclientset.Interface, listFunc ListClusterClaimsFunc) *ClusterClaimReconciler {
	return &ClusterClaimReconciler{
		Log:               tlog.NullLogger{},
		ClusterClient:     clusterClient,
		ListClusterClaims: listFunc,
	}
}

func TestCreateOrUpdate(t *testing.T) {
	ctx := context.Background()
	// create
	clusterClient := clusterfake.NewSimpleClientset()
	reconciler := newClusterClaimReconciler(clusterClient, nil)

	claim1 := newClusterClaim("x", "y")
	if err := reconciler.createOrUpdate(ctx, claim1); err != nil {
		t.Errorf("Failed to create or update cluster claim: %v", err)
	}

	actions := clusterClient.Actions()
	if len(actions) != 2 {
		t.Errorf("Expect %d actions, but got: %v", 2, len(actions))
	}
	if actions[1].GetVerb() != "create" {
		t.Errorf("Expect action create, but got: %s", actions[1].GetVerb())
	}

	// update
	clusterClient = clusterfake.NewSimpleClientset(newClusterClaim("x", "y"))
	reconciler = newClusterClaimReconciler(clusterClient, nil)

	claim1 = newClusterClaim("x", "z")
	if err := reconciler.createOrUpdate(ctx, claim1); err != nil {
		t.Errorf("Failed to create or update cluster claim: %v", err)
	}

	actions = clusterClient.Actions()
	if len(actions) != 2 {
		t.Errorf("Expect 2 actions, but got: %v", len(actions))
	}
	if actions[1].GetVerb() != "update" {
		t.Errorf("Expect action update, but got: %s", actions[1].GetVerb())
	}
}

func TestSyncClaims(t *testing.T) {
	ctx := context.Background()
	expected := []*clusterv1alpha1.ClusterClaim{
		newClusterClaim("x", "1"),
		newClusterClaim("y", "2"),
		newClusterClaim("z", "3"),
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "o",
			},
		},
	}

	deletedClaim := &clusterv1alpha1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "p",
			Labels: map[string]string{labelHubManaged: ""},
		},
	}

	clusterClient := clusterfake.NewSimpleClientset(deletedClaim)
	reconciler := newClusterClaimReconciler(clusterClient, func() ([]*clusterv1alpha1.ClusterClaim, error) {
		return expected, nil
	})

	if err := reconciler.syncClaims(ctx); err != nil {
		t.Errorf("Failed to sync cluster claims: %v", err)
	}

	for _, item := range expected {
		claim, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(), item.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Unable to find cluster claims: %s", item.Name)
		}

		if !reflect.DeepEqual(item.Spec, claim.Spec) {
			t.Errorf("Expected cluster claim %v, but got %v", item, claim)
		}
	}

	if _, err := clusterClient.ClusterV1alpha1().ClusterClaims().Get(context.Background(),
		deletedClaim.Name, metav1.GetOptions{}); !errors.IsNotFound(err) {
		t.Errorf("deleted cluster claim %v is not deleted", deletedClaim.Name)
	}

}
