package clusterinfo

import (
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/objectsource"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	claimSource := objectsource.NewClusterClaimSource(r.ClaimInformer)
	nodeSource := objectsource.NewNodeSource(r.NodeInformer)
	return ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(claimSource).
		WatchesRawSource(nodeSource).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}
