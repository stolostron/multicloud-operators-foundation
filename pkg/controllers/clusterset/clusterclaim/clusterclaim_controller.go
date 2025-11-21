package clusterclaim

import (
	"context"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	utils "github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This controller sync the clusterclaim's ClusterSetLabel with releated clusterpool's ClusterSetLabel
// if the clusterpool did not exist, do nothing.
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create ClusterSetMapper controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterset-clusterclaim-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), &hivev1.ClusterClaim{},
		&handler.TypedEnqueueRequestForObject[*hivev1.ClusterClaim]{}))
	if err != nil {
		return err
	}

	// watch clusterdeployment's claim
	err = c.Watch(
		source.Kind(mgr.GetCache(), &hivev1.ClusterDeployment{},
			handler.TypedEnqueueRequestsFromMapFunc[*hivev1.ClusterDeployment](
				func(ctx context.Context, clusterDeployment *hivev1.ClusterDeployment) []reconcile.Request {
					// this clusterdeployment is not related to any pool
					if clusterDeployment.Spec.ClusterPoolRef == nil {
						return []reconcile.Request{}
					}
					// this clusterdeployment is not claimed
					if len(clusterDeployment.Spec.ClusterPoolRef.ClaimName) == 0 {
						return []reconcile.Request{}
					}
					clusterclaim := &hivev1.ClusterClaim{}
					claimInfo := types.NamespacedName{Namespace: clusterDeployment.Spec.ClusterPoolRef.Namespace, Name: clusterDeployment.Spec.ClusterPoolRef.ClaimName}
					err = mgr.GetClient().Get(context.TODO(), claimInfo, clusterclaim)
					if err != nil {
						klog.Errorf("Failed to get clusterclaim, error: %v", err)
						return []reconcile.Request{}
					}

					var requests []reconcile.Request
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      clusterclaim.Name,
							Namespace: clusterclaim.Namespace,
						},
					})

					return requests
				}),
		),
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterclaim := &hivev1.ClusterClaim{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterclaim)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// update clusterclaim label by clusterdeployment label
	if len(clusterclaim.Spec.Namespace) == 0 {
		return ctrl.Result{}, nil
	}

	clusterDeploymentName := clusterclaim.Spec.Namespace
	clusterDeployment := &hivev1.ClusterDeployment{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: clusterDeploymentName, Name: clusterDeploymentName}, clusterDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.V(5).Infof("Clusterclaim's clusterdeployment: %+v", clusterDeployment)

	var isModified = false
	patch := client.MergeFrom(clusterclaim.DeepCopy())
	utils.SyncMapField(&isModified, &clusterclaim.Labels, clusterDeployment.Labels, clusterv1beta2.ClusterSetLabel)

	if isModified {
		err = r.client.Patch(ctx, clusterclaim, patch)
		if err != nil {
			klog.Errorf("Can not patch clusterclaim label: %+v", clusterclaim.Name)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
