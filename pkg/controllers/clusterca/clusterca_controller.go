package clusterca

import (
	"context"
	"reflect"

	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
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

	err = c.Watch(&source.Kind{Type: &clusterinfov1beta1.ManagedClusterInfo{}}, &handler.EnqueueRequestForObject{})
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

	updatedClientConfig, needUpdateMC, needUpdateMCI := updateClientConfig(
		cluster.Spec.ManagedClusterClientConfigs,
		clusterInfo.Status.DistributionInfo.OCP.ManagedClusterClientConfig,
		clusterInfo.Status.DistributionInfo.OCP.LastAppliedAPIServerURL)

	if needUpdateMC {
		cluster.Spec.ManagedClusterClientConfigs = updatedClientConfig
		err = r.client.Update(ctx, cluster, &client.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if needUpdateMCI {
		clusterInfo.Status.DistributionInfo.OCP.LastAppliedAPIServerURL =
			clusterInfo.Status.DistributionInfo.OCP.ManagedClusterClientConfig.URL
		// retry if update managed cluster info status failed with conflict, because if we do not
		// retry but return err here, next reconcile will stop at "updateClientConfig", since the
		// clientConfig of the ManagedCluster CR has already been updated successfully above in the current
		// reconcile, the "updateClientConfig" func will return needUpdateMCI=false in the next reconcile loop.
		err = r.UpdateManagedClusterInfoStatus(ctx, clusterInfo)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *ClusterCaReconciler) UpdateManagedClusterInfoStatus(
	ctx context.Context, instance *clusterinfov1beta1.ManagedClusterInfo) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return r.client.Status().Update(ctx, instance)
	})
}

// updateClientConfig merge config from clusterinfoconfigs to clusterconfigs, it returns 3 parameters:
//   - the updated cluster clientConfig
//   - whether it is needed to update the managedCluster object
//   - whether it is needed to update the managedClusterInfo object, only if the lastAppliedURL
//     changes, this will be true
func updateClientConfig(clusterConfigs []clusterv1.ClientConfig, clusterinfoConfig clusterinfov1beta1.ClientConfig,
	lastAppliedURL string) ([]clusterv1.ClientConfig, bool, bool) {
	// If clusterinfo config is null return clusterconfigs
	if len(clusterinfoConfig.URL) == 0 {
		return clusterConfigs, false, false
	}

	// The lastAppliedURL comes from the value of infrastructures config(status.apiServerURL), if the
	// infrastructures config value changes, here we will replace the old clientConfig URL instead of
	// prepending a new one.
	for i, cluConfig := range clusterConfigs {
		if cluConfig.URL != clusterinfoConfig.URL && cluConfig.URL != lastAppliedURL {
			continue
		}

		if reflect.DeepEqual(cluConfig.CABundle, clusterinfoConfig.CABundle) && cluConfig.URL == clusterinfoConfig.URL {
			return clusterConfigs, false, false
		}

		clusterConfigs[i].URL = clusterinfoConfig.URL
		clusterConfigs[i].CABundle = clusterinfoConfig.CABundle
		return clusterConfigs, true, cluConfig.URL != clusterinfoConfig.URL
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

	return clusterConfigs, true, true
}
