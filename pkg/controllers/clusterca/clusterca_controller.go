package clusterca

import (
	"context"
	"reflect"

	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
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

	err = c.Watch(source.Kind(mgr.GetCache(), &clusterinfov1beta1.ManagedClusterInfo{},
		&handler.TypedEnqueueRequestForObject[*clusterinfov1beta1.ManagedClusterInfo]{}))
	if err != nil {
		return err
	}
	return nil
}

func (r *ClusterCaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}

	err := r.client.Get(ctx, req.NamespacedName, clusterInfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cluster := &clusterv1.ManagedCluster{}

	err = r.client.Get(ctx, types.NamespacedName{Name: clusterInfo.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	updatedClientConfig, needUpdate := updateClientConfig(
		cluster.Spec.ManagedClusterClientConfigs,
		clusterInfo.Status.DistributionInfo.OCP.ManagedClusterClientConfig,
		clusterInfo.Status.DistributionInfo.OCP.LastAppliedAPIServerURL)

	if needUpdate {
		cluster.Spec.ManagedClusterClientConfigs = updatedClientConfig
		err = r.client.Update(ctx, cluster, &client.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if clusterInfo.Status.DistributionInfo.OCP.LastAppliedAPIServerURL !=
		clusterInfo.Status.DistributionInfo.OCP.ManagedClusterClientConfig.URL {
		clusterInfo.Status.DistributionInfo.OCP.LastAppliedAPIServerURL =
			clusterInfo.Status.DistributionInfo.OCP.ManagedClusterClientConfig.URL
		err = r.client.Status().Update(ctx, clusterInfo)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// updateClientConfig merge config from clusterinfoconfigs to clusterconfigs, it returns 2 parameters:
//   - the updated cluster clientConfig
//   - whether it is needed to update the managedCluster object
func updateClientConfig(clusterConfigs []clusterv1.ClientConfig, clusterinfoConfig clusterinfov1beta1.ClientConfig,
	lastAppliedURL string) ([]clusterv1.ClientConfig, bool) {
	// If clusterinfo config is null return clusterconfigs
	if len(clusterinfoConfig.URL) == 0 {
		return clusterConfigs, false
	}

	// The lastAppliedURL comes from the value of infrastructures config(status.apiServerURL), if the
	// infrastructures config value changes, here we will replace the old clientConfig URL instead of
	// prepending a new one.
	for i, cluConfig := range clusterConfigs {
		if cluConfig.URL != clusterinfoConfig.URL && cluConfig.URL != lastAppliedURL {
			continue
		}

		if reflect.DeepEqual(cluConfig.CABundle, clusterinfoConfig.CABundle) && cluConfig.URL == clusterinfoConfig.URL {
			return clusterConfigs, false
		}

		clusterConfigs[i].URL = clusterinfoConfig.URL
		clusterConfigs[i].CABundle = clusterinfoConfig.CABundle
		return clusterConfigs, true
	}

	// do not have ca bundle in cluster config, *prepend* the clusterinfo config to the managed cluster config.
	// Because in a corner hypershift scenario, a hosted cluster is destroyed and recreated with the same name.
	// At this time, clusterinfoConfig holds the data of the newly created hosted cluster, which is corrent, so
	// we put the new value in the first element for the front-end and foundation logging controller
	// consumption(they always get the first value).
	// Details: https://issues.redhat.com/browse/ACM-2094
	clusterConfigs = append([]clusterv1.ClientConfig{{
		URL:      clusterinfoConfig.URL,
		CABundle: clusterinfoConfig.CABundle,
	}}, clusterConfigs...)

	return clusterConfigs, true
}
