package clusterset

import (
	"context"
	"fmt"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Reconciler struct {
	client              client.Client
	scheme              *runtime.Scheme
	clustersetToCluster map[string][]string
}

func SetupWithManager(mgr manager.Manager, clustersetToCluster map[string][]string) error {
	if err := add(mgr, newReconciler(mgr, clustersetToCluster)); err != nil {
		klog.Errorf("Failed to create auto-detect controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clustersetToCluster map[string][]string) reconcile.Reconciler {
	return &Reconciler{
		client:              mgr.GetClient(),
		scheme:              mgr.GetScheme(),
		clustersetToCluster: clustersetToCluster,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.ManagedClusterSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//	var clusterroleToClusterset map[string][]string
	ctx := context.Background()
	cluster := &clusterv1.ManagedCluster{}
	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	clustersetList := &clusterv1alpha1.ManagedClusterSetList{}
	err = r.client.List(ctx, clustersetList)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, clusterset := range clustersetList.Items {
		selectedClusters := r.getSelecedClusters(ctx, clusterset)
		r.clustersetToCluster[clusterset.Name] = selectedClusters
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) getSelecedClusters(ctx context.Context, clusterset clusterv1alpha1.ManagedClusterSet) []string {
	var selectedClusters []string
	for _, selector := range clusterset.Spec.ClusterSelectors {
		clusters, err := r.selectClusters(ctx, selector)
		if err != nil {
			klog.Errorf("Error to select cluster. %+v", err)
		}
		for _, cluster := range clusters {
			selectedClusters = append(selectedClusters, cluster)
		}
	}
	return selectedClusters
}

// selectClusters returns names of managed clusters which match the cluster selector
func (r *Reconciler) selectClusters(ctx context.Context, selector clusterv1alpha1.ClusterSelector) (clusterNames []string, err error) {
	switch {
	case len(selector.ClusterNames) > 0 && selector.LabelSelector != nil:
		// return error if both ClusterNames and LabelSelector is specified for they are mutually exclusive
		// This case should be handled by validating webhook
		return nil, fmt.Errorf("both ClusterNames and LabelSelector is specified in ClusterSelector: %v", selector.LabelSelector)
	case len(selector.ClusterNames) > 0:
		// select clusters with cluster names
		for _, clusterName := range selector.ClusterNames {
			cluster := &clusterv1.ManagedCluster{}
			err := r.client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
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
		err = r.client.List(ctx, clusters, &client.ListOptions{LabelSelector: labelSelector})
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
