package clusterclaim

import (
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func (r *clusterClaimReconciler) SetupWithManager(mgr ctrl.Manager, clusterInformer clusterv1alpha1informer.ClusterClaimInformer) error {
	// setup a controller for ClusterClaim with customized event source and handler
	c, err := controller.New("ClusterClaim", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &clusterv1alpha1.ClusterClaim{}), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// There are 2 use cases for this watch:
	// 1. when labels of the managedclusterinfo changed, we need to sync labels to clusterclaims
	// 2. at the beginning of the pod, we need this watch to trigger the reconcile of all clusterclaims
	if err := c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta1.ManagedClusterInfo{}), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}
