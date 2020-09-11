package clustersetmapper

import (
	"context"
	"fmt"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// put all managedClusterSets into queue if there is the managedCluster event
	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []reconcile.Request {
				if _, ok := obj.Object.(*clusterv1.ManagedCluster); !ok {
					// not a managedCluster, returning empty
					klog.Error("managedCluster handler received non-managedCluster object")
					return []reconcile.Request{}
				}

				managedClusterSets := &clusterv1alpha1.ManagedClusterSetList{}
				err := mgr.GetClient().List(context.TODO(), managedClusterSets, &client.ListOptions{})
				if err != nil {
					klog.Errorf("failed to list managedClusterSet %v", err)
				}

				var requests []reconcile.Request
				for _, managedClusterSet := range managedClusterSets.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name: managedClusterSet.Name,
						},
					})
				}

				klog.V(5).Infof("List managedClusterSet %+v", requests)
				return requests
			}),
		})
	if err != nil {
		return nil
	}

	if err = c.Watch(&source.Kind{Type: &clusterv1alpha1.ManagedClusterSet{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	managedClusterSet := &clusterv1alpha1.ManagedClusterSet{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, managedClusterSet)
	if err != nil {
		if errors.IsNotFound(err) {
			// managedClusterSet has been deleted
			r.clusterSetMapper.DeleteClusterSet(req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	selectedClusters := sets.NewString()
	for _, clusterSelector := range managedClusterSet.Spec.ClusterSelectors {
		clusters, err := r.selectClusters(clusterSelector)
		if err != nil {

		}
		selectedClusters.Insert(clusters...)
	}

	r.clusterSetMapper.UpdateClusterSetByClusters(managedClusterSet.Name, selectedClusters)

	klog.V(5).Infof("clusterSetMapper: %+v", r.clusterSetMapper.GetAllClusterSetToClusters())
	return ctrl.Result{}, nil
}

// selectClusters returns names of managed clusters which match the cluster selector
func (r *Reconciler) selectClusters(selector clusterv1alpha1.ClusterSelector) (clusterNames []string, err error) {
	switch {
	case len(selector.ClusterNames) > 0 && selector.LabelSelector != nil:
		// return error if both ClusterNames and LabelSelector is specified for they are mutually exclusive
		// This case should be handled by validating webhook
		return nil, fmt.Errorf("both ClusterNames and LabelSelector is specified in ClusterSelector: %v", selector.LabelSelector)
	case len(selector.ClusterNames) > 0:
		// select clusters with cluster names
		for _, clusterName := range selector.ClusterNames {
			cluster := &clusterv1.ManagedCluster{}
			err = r.client.Get(context.Background(), types.NamespacedName{Name: clusterName}, cluster)
			switch {
			case errors.IsNotFound(err):
				continue
			case err != nil:
				return nil, fmt.Errorf("unable to fetch ManagedCluster %q: %w", clusterName, err)
			default:
				clusterNames = append(clusterNames, clusterName)
			}
		}
		return clusterNames, nil
	case selector.LabelSelector != nil:
		// select clusters with label selector
		labelSelector, err := convertLabels(selector.LabelSelector)
		if err != nil {
			// This case should be handled by validating webhook
			return nil, fmt.Errorf("invalid label selector: %v, %w", selector.LabelSelector, err)
		}
		clusters := &clusterv1.ManagedClusterList{}
		err = r.client.List(context.Background(),
			clusters,
			&client.ListOptions{
				LabelSelector: labelSelector},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to list ManagedClusters with label selector: %v, %w", selector.LabelSelector, err)
		}
		for _, cluster := range clusters.Items {
			clusterNames = append(clusterNames, cluster.Name)
		}
		return clusterNames, nil
	default:
		// no cluster selected if neither ClusterNames nor LabelSelector is specified
		return clusterNames, nil
	}
}

// convertLabels returns label
func convertLabels(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return labels.Nothing(), err
		}

		return selector, nil
	}

	return labels.Everything(), nil
}
