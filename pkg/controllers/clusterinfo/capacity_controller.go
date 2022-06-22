package clusterinfo

import (
	"context"

	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	resourceSocket       clusterv1.ResourceName = "socket"
	resourceCoreWorker   clusterv1.ResourceName = "core_worker"
	resourceSocketWorker clusterv1.ResourceName = "socket_worker"
)

type CapacityReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

// newAutoDetectReconciler returns a new reconcile.Reconciler
func newCapacityReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &CapacityReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

func (r *CapacityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1.ManagedCluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		return reconcile.Result{}, nil
	}
	capacity := cluster.DeepCopy().Status.Capacity
	if capacity == nil {
		capacity = clusterv1.ResourceList{}
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, nil
	case err != nil:
		return ctrl.Result{}, err
	}

	nodes := clusterInfo.Status.NodeList
	socketWorkerCapacity := *resource.NewQuantity(int64(0), resource.DecimalSI)
	coreWorkerCapacity := *resource.NewQuantity(int64(0), resource.DecimalSI)
	for _, node := range nodes {
		if isWorker(node) {
			coreWorkerCapacity.Add(node.Capacity[clusterv1.ResourceCPU])
			socketWorkerCapacity.Add(node.Capacity[resourceSocket])
		}
	}
	capacity[resourceSocketWorker] = socketWorkerCapacity
	capacity[resourceCoreWorker] = coreWorkerCapacity

	if apiequality.Semantic.DeepEqual(capacity, cluster.Status.Capacity) {
		return ctrl.Result{}, nil
	}

	cluster.Status.Capacity = capacity
	return ctrl.Result{}, r.client.Status().Update(ctx, cluster)
}

func isWorker(node clusterinfov1beta1.NodeStatus) bool {
	if node.Labels == nil {
		return false
	}

	if _, ok := node.Labels["node-role.kubernetes.io/worker"]; ok {
		return true
	}

	return false
}
