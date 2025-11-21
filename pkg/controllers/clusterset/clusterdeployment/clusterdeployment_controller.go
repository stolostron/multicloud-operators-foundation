package clusterdeployment

import (
	"context"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This controller sync the clusterdeployment's ClusterSet Label
// In ACM, clusterdeployment should be created in two ways
//  1. Directlly create a managedcluster, it will also create a clusterdeployment ref to this managedcluster.
//     So the clusterdeployment's clusterset label should be synced from this managedcluster.
//  2. Claim a cluster from clusterpool, all the clusters claimed from this pool should be in the same clusterset.
//     And the clusterdeployment's clusterset should be same as clusterpool. So we need to sync the clusterdeployment clusterset from clusterpool
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create ClusterDeployment controller, %v", err)
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
	c, err := controller.New("clusterset-clusterdeployment-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), &hivev1.ClusterDeployment{}),
		&handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	//watch all clusterpool related clusterdeployments
	err = c.Watch(
		source.Kind(mgr.GetCache(), &hivev1.ClusterPool{}),
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
				clusterPool, ok := a.(*hivev1.ClusterPool)
				if !ok {
					// not a clusterpool, returning empty
					klog.Error("Clusterpool handler received non-Clusterpool object")
					return []reconcile.Request{}
				}
				clusterdeployments := &hivev1.ClusterDeploymentList{}
				err := mgr.GetClient().List(context.TODO(), clusterdeployments, &client.ListOptions{})
				if err != nil {
					klog.Errorf("could not list clusterdeployments. Error: %v", err)
				}
				var requests []reconcile.Request
				for _, clusterdeployment := range clusterdeployments.Items {
					//If clusterdeployment is not created by clusterpool or already cliamed, ignore it
					if clusterdeployment.Spec.ClusterPoolRef == nil || len(clusterdeployment.Spec.ClusterPoolRef.ClaimName) != 0 {
						continue
					}
					//Only filter clusterpool related clusterdeployment
					if clusterdeployment.Spec.ClusterPoolRef.PoolName != clusterPool.Name || clusterdeployment.Spec.ClusterPoolRef.Namespace != clusterPool.Namespace {
						continue
					}
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      clusterdeployment.Name,
							Namespace: clusterdeployment.Namespace,
						},
					})
				}
				return requests

			}),
		))
	if err != nil {
		return err
	}

	//watch managedcluster related clusterdeployment
	err = c.Watch(
		source.Kind(mgr.GetCache(), &clusterv1.ManagedCluster{}),
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
				managedCluster, ok := a.(*clusterv1.ManagedCluster)
				if !ok {
					// not a managedcluster, returning empty
					klog.Error("managedCluster handler received non-managedCluster object")
					return []reconcile.Request{}
				}
				clusterdeployment := &hivev1.ClusterDeployment{}
				err := mgr.GetClient().Get(context.TODO(),
					types.NamespacedName{
						Name:      managedCluster.Name,
						Namespace: managedCluster.Name,
					},
					clusterdeployment,
				)

				if err != nil || clusterdeployment == nil {
					return []reconcile.Request{}
				}
				var requests []reconcile.Request
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      clusterdeployment.Name,
						Namespace: clusterdeployment.Namespace,
					},
				})
				return requests

			}),
		))
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterdeployment := &hivev1.ClusterDeployment{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterdeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	var targetLabels map[string]string
	//if the clusterdeployment is not created by clusterpool, update label follow managedcluster label
	if clusterdeployment.Spec.ClusterPoolRef == nil {
		//get managedclusterlabel
		managedcluster := clusterv1.ManagedCluster{}
		err := r.client.Get(ctx,
			types.NamespacedName{
				Name: clusterdeployment.Namespace,
			},
			&managedcluster,
		)
		if err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		targetLabels = managedcluster.Labels
		klog.V(5).Infof("Clusterdeployment's managedcluster: %+v", managedcluster)
	} else {
		// clusterpool's clusterdeployment, update clusterdeployment label by clusterpool
		clusterpool := &hivev1.ClusterPool{}
		err = r.client.Get(ctx, types.NamespacedName{Namespace: clusterdeployment.Spec.ClusterPoolRef.Namespace, Name: clusterdeployment.Spec.ClusterPoolRef.PoolName}, clusterpool)
		if err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		targetLabels = clusterpool.Labels
		klog.V(5).Infof("Clusterdeployment's clusterpool: %+v", clusterpool)
	}

	var isModified = false
	patch := client.MergeFrom(clusterdeployment.DeepCopy())
	utils.SyncMapField(&isModified, &clusterdeployment.Labels, targetLabels, clusterv1beta2.ClusterSetLabel)

	if isModified {
		err = r.client.Patch(ctx, clusterdeployment, patch)
		if err != nil {
			klog.Errorf("Can not patch clusterdeployment label. clusterdeployment: %v, error:%v", clusterdeployment.Name, err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
