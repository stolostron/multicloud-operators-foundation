package clusterclaim

import (
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/objectsource"
	coreinformers "k8s.io/client-go/informers/core/v1"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func (r *clusterClaimReconciler) SetupWithManager(mgr ctrl.Manager,
	clusterInformer clusterv1alpha1informer.ClusterClaimInformer,
	nodeInformer coreinformers.NodeInformer) error {
	// setup a controller for ClusterClaim with customized event source and handler
	c, err := controller.New("ClusterClaim", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	claimSource := objectsource.NewClusterClaimSource(clusterInformer)
	if err := c.Watch(claimSource, objectsource.NewClusterClaimEventHandler()); err != nil {
		return err
	}

	// There are 3 use cases for this watch:
	// 1. when labels of the managedclusterinfo changed, we need to sync labels to clusterclaims
	// 2. at the beginning of the pod, we need this watch to trigger the reconcile of all clusterclaims
	// 3. when nodes' schedulable status changed, we need to sync the status to the clusterclaim - schedulable.open-cluster-management.io
	if err := c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta1.ManagedClusterInfo{}), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	nodeSource := &objectsource.NodeSource{NodeInformer: nodeInformer.Informer()}
	if err := c.Watch(nodeSource, &objectsource.NodeEventHandler{}); err != nil {
		return err
	}

	return nil
}
