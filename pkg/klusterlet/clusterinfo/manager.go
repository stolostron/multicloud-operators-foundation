package controllers

import (
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/objectsource"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	claimSource := objectsource.NewClusterClaimSource(r.ClaimInformer)
	nodeSource := &objectsource.NodeSource{NodeInformer: r.NodeInformer.Informer()}
	return ctrl.NewControllerManagedBy(mgr).
		WatchesRawSource(claimSource, objectsource.NewClusterClaimEventHandler()).
		WatchesRawSource(nodeSource, &objectsource.NodeEventHandler{}).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}
