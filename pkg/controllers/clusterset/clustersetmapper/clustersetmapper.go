package clustersetmapper

import (
	"context"

	clusterv1alapha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterSetLabel = "cluster.open-cluster-management.io/clusterset"
)

type Reconciler struct {
	client           client.Client
	scheme           *runtime.Scheme
	clusterSetMapper *helpers.ClusterSetMapper
}

func SetupWithManager(mgr manager.Manager, clusterSetMapper *helpers.ClusterSetMapper) error {
	if err := add(mgr, newReconciler(mgr, clusterSetMapper)); err != nil {
		klog.Errorf("Failed to create ClusterSetMapper controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clusterSetMapper *helpers.ClusterSetMapper) reconcile.Reconciler {
	return &Reconciler{
		client:           mgr.GetClient(),
		scheme:           mgr.GetScheme(),
		clusterSetMapper: clusterSetMapper,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clustersetmapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// put all managedCluster which have this clusterset label into queue if there is the managedClusterset event
	err = c.Watch(&source.Kind{Type: &clusterv1alapha1.ManagedClusterSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				if _, ok := obj.Object.(*clusterv1alapha1.ManagedClusterSet); !ok {
					// not a managedClusterset, returning empty
					klog.Error("managedClusterset handler received non-managedClusterset object")
					return []reconcile.Request{}
				}

				managedClusters := &clusterv1.ManagedClusterList{}

				//List Clusterset related cluster
				labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
					clusterSetLabel: obj.Meta.GetName(),
				}}
				selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
				if err != nil {
					return nil
				}

				err = mgr.GetClient().List(context.TODO(), managedClusters, &client.ListOptions{LabelSelector: selector})
				if err != nil {
					klog.Errorf("failed to list managedClusterSet %v", err)
				}

				var requests []reconcile.Request
				for _, managedCluster := range managedClusters.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name: managedCluster.Name,
						},
					})
				}

				klog.V(5).Infof("List managedCluster %+v", requests)
				return requests
			}),
		})
	if err != nil {
		return nil
	}
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	managedCluster := &clusterv1.ManagedCluster{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, managedCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// managedCluster has been deleted
			r.clusterSetMapper.DeleteClusterInClusterSet(req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if _, ok := managedCluster.Labels[clusterSetLabel]; !ok {
		r.clusterSetMapper.DeleteClusterInClusterSet(req.Name)
		return ctrl.Result{}, nil
	}

	managedClustersetName := managedCluster.Labels[clusterSetLabel]

	//If the managedclusterset do not exist, delete this clusterset in map
	managedClusterset := &clusterv1alapha1.ManagedClusterSet{}
	err = r.client.Get(ctx, types.NamespacedName{Name: managedClustersetName}, managedClusterset)
	if err != nil {
		if errors.IsNotFound(err) {
			r.clusterSetMapper.DeleteClusterSet("managedClusterset")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	managedClusterName := managedCluster.GetName()
	r.clusterSetMapper.UpdateClusterInClusterSet(managedClusterName, managedClustersetName)

	klog.V(5).Infof("clusterSetMapper: %+v", r.clusterSetMapper.GetAllClusterSetToClusters())
	return ctrl.Result{}, nil
}
