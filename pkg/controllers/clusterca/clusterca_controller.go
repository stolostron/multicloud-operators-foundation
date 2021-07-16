package clusterca

import (
	"context"
	"reflect"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
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

type ClusterCaReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create clusterca controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ClusterCaReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterca-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterinfov1beta1.ManagedClusterInfo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

func (r *ClusterCaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterinfo := &clusterinfov1beta1.ManagedClusterInfo{}

	err := r.client.Get(ctx, req.NamespacedName, clusterinfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cluster := &clusterv1.ManagedCluster{}

	err = r.client.Get(ctx, types.NamespacedName{Name: clusterinfo.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	updatedClientConfig, needUpdate := updateClientConfig(cluster.Spec.ManagedClusterClientConfigs, clusterinfo.Status.DistributionInfo.OCP.ManagedClusterClientConfig)
	if !needUpdate {
		return ctrl.Result{}, nil
	}

	cluster.Spec.ManagedClusterClientConfigs = updatedClientConfig
	err = r.client.Update(ctx, cluster, &client.UpdateOptions{})

	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

//updateClientConfig merge config from clusterinfoconfigs to clusterconfigs
func updateClientConfig(clusterConfigs []clusterv1.ClientConfig, clusterinfoConfig clusterinfov1beta1.ClientConfig) ([]clusterv1.ClientConfig, bool) {
	//If clusterinfo config is null return clusterconfigs
	if len(clusterinfoConfig.URL) == 0 {
		return clusterConfigs, false
	}

	for i, cluConfig := range clusterConfigs {
		if cluConfig.URL != clusterinfoConfig.URL {
			continue
		}
		if !reflect.DeepEqual(cluConfig.CABundle, clusterinfoConfig.CABundle) {
			clusterConfigs[i].CABundle = clusterinfoConfig.CABundle
			return clusterConfigs, true
		}
		return clusterConfigs, false
	}

	//do not have ca bundle in cluster config, append it to managedcluster config
	tempConfig := clusterv1.ClientConfig{
		URL:      clusterinfoConfig.URL,
		CABundle: clusterinfoConfig.CABundle,
	}
	clusterConfigs = append(clusterConfigs, tempConfig)

	return clusterConfigs, true
}
