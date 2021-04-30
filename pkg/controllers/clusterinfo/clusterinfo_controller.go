package clusterinfo

import (
	"context"
	"reflect"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"

	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	clusterFinalizerName = "managedclusterinfo.finalizers.open-cluster-management.io"
)

type ClusterInfReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	caData []byte
}

func SetupWithManager(mgr manager.Manager, caData []byte) error {
	if err := add(mgr, newClusterInfoReconciler(mgr, caData)); err != nil {
		return err
	}
	if err := add(mgr, newAutoDetectReconciler(mgr)); err != nil {
		return err
	}
	if err := add(mgr, newCapacityReconciler(mgr)); err != nil {
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newClusterInfoReconciler(mgr manager.Manager, caData []byte) reconcile.Reconciler {
	return &ClusterInfReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		caData: caData,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterinfo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &clusterinfov1beta1.ManagedClusterInfo{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

func (r *ClusterInfReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1.ManagedCluster{}

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !cluster.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			err := r.deleteClusterInfo(cluster.Name)
			if err != nil {
				return reconcile.Result{}, err
			}

			cluster.ObjectMeta.Finalizers = utils.RemoveString(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			return reconcile.Result{}, r.client.Update(context.TODO(), cluster)
		}

		return reconcile.Result{}, nil
	}

	if !utils.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
		cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
		return reconcile.Result{}, r.client.Update(context.TODO(), cluster)
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, r.client.Create(ctx, r.newClusterInfoByManagedCluster(cluster))
	case err != nil:
		return ctrl.Result{}, err
	}

	endpoint := ""
	if len(cluster.Spec.ManagedClusterClientConfigs) != 0 {
		endpoint = cluster.Spec.ManagedClusterClientConfigs[0].URL
	}

	if !reflect.DeepEqual(r.caData, clusterInfo.Spec.LoggingCA) ||
		clusterInfo.Spec.MasterEndpoint != endpoint {
		clusterInfo.Spec.LoggingCA = r.caData
		clusterInfo.Spec.MasterEndpoint = endpoint
		return ctrl.Result{}, r.client.Update(ctx, clusterInfo)
	}

	// TODO: the conditions of managed cluster need to be deprecated.
	newConditions := cluster.Status.Conditions
	syncedCondition := meta.FindStatusCondition(clusterInfo.Status.Conditions, clusterinfov1beta1.ManagedClusterInfoSynced)
	if syncedCondition != nil {
		newConditions = append(newConditions, *syncedCondition)
	}

	if !reflect.DeepEqual(newConditions, clusterInfo.Status.Conditions) {
		clusterInfo.Status.Conditions = newConditions
		err = r.client.Status().Update(ctx, clusterInfo)
		if err != nil {
			klog.Warningf("will reconcile since failed to update ManagedClusterInfo status %v, %v", cluster.Name, err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *ClusterInfReconciler) deleteClusterInfo(name string) error {
	err := r.client.Delete(context.Background(), &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
	})
	return client.IgnoreNotFound(err)
}

func (r *ClusterInfReconciler) newClusterInfoByManagedCluster(cluster *clusterv1.ManagedCluster) *clusterinfov1beta1.ManagedClusterInfo {
	endpoint := ""
	if len(cluster.Spec.ManagedClusterClientConfigs) != 0 {
		endpoint = cluster.Spec.ManagedClusterClientConfigs[0].URL
	}

	return &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Name,
			Labels:    cluster.Labels,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			MasterEndpoint: endpoint,
			LoggingCA:      r.caData,
		},
	}
}
