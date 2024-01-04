package controllers

import (
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch ClusterClaims and Nodes changes for ManagedClusterInfo reconcile
	return ctrl.NewControllerManagedBy(mgr).
		Watches(&clusterv1alpha1.ClusterClaim{}, &handler.EnqueueRequestForObject{}).
		Watches(&corev1.Node{}, &handler.EnqueueRequestForObject{}).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}
