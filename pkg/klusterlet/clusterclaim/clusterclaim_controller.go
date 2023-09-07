package clusterclaim

import (
	"context"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterclientset "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ListClusterClaimsFunc func() ([]*clusterv1alpha1.ClusterClaim, error)

// ClusterClaimReconciler reconciles cluster claim objects
type ClusterClaimReconciler struct {
	Log               logr.Logger
	ListClusterClaims ListClusterClaimsFunc
	ClusterClient     clusterclientset.Interface
	ClusterInformers  clusterv1alpha1informer.ClusterClaimInformer
}

func (r *ClusterClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	err := r.syncClaims(ctx)
	return ctrl.Result{}, err
}

func (r *ClusterClaimReconciler) syncClaims(ctx context.Context) error {
	r.Log.V(4).Info("Sync cluster claims")
	claims, err := r.ListClusterClaims()
	if err != nil {
		return err
	}

	errs := []error{}
	claimSet := sets.String{}

	for _, claim := range claims {
		if err := r.createOrUpdate(ctx, claim, clusterClaimCreateOnlyList); err != nil {
			errs = append(errs, err)
		}
		claimSet.Insert(claim.Name)
	}

	labelSelector := fmt.Sprintf("%s=%s", labelHubManaged, "")
	existedObjs, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		errs = append(errs, client.IgnoreNotFound(err))
		return utils.NewMultiLineAggregate(errs)
	}

	for _, claim := range existedObjs.Items {
		// these claims should not be deleted in any scenario once they are created.
		switch claim.Name {
		case ClaimK8sID, ClaimOCMKubeVersion, ClaimOCMPlatform, ClaimOCMProduct, ClaimOpenshiftVersion, ClaimOpenshiftID:
			continue
		}

		if claimSet.Has(claim.Name) {
			continue
		}
		err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Delete(ctx, claim.Name, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utils.NewMultiLineAggregate(errs)
}

func (r *ClusterClaimReconciler) createOrUpdate(ctx context.Context, newClaim *clusterv1alpha1.ClusterClaim, clusterClaimCreateOnlyList []string) error {
	oldClaim, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Get(ctx, newClaim.Name, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		_, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Create(ctx, newClaim, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("unable to create ClusterClaim: %v, %w", newClaim, err)
		}
	case err != nil:
		return fmt.Errorf("unable to get ClusterClaim %q: %w", newClaim.Name, err)
	case !reflect.DeepEqual(oldClaim.Spec, newClaim.Spec):
		// if newClaim.Name is in clusterClaimCreateOnlyList, then do nothing
		if utils.ContainsString(clusterClaimCreateOnlyList, newClaim.Name) && oldClaim.Spec.Value != "" {
			return nil
		}
		oldClaim.Spec = newClaim.Spec
		_, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().Update(ctx, oldClaim, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update ClusterClaim %q: %w", oldClaim.Name, err)
		}
	}
	return nil
}
