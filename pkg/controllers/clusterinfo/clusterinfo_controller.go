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

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
	caData []byte
}

func SetupWithManager(mgr manager.Manager, caData []byte) error {
	if err := add(mgr, newReconciler(mgr, caData)); err != nil {
		klog.Errorf("Failed to create ManagedClusterInfo controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, caData []byte) reconcile.Reconciler {
	return &Reconciler{
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

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterv1.ManagedCluster{}

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !cluster.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			klog.Infof("deleting ManagedClusterInfo %v", cluster.Name)
			err := r.deleteExternalResources(cluster.Name, cluster.Name)
			if err != nil {
				klog.Warningf("will reconcile since failed to delete ManagedClusterInfo %v : %v", cluster.Name, err)
				return reconcile.Result{}, err
			}

			klog.Infof("removing ManagedClusterInfo Finalizer in ManagedCluster %v", cluster.Name)
			cluster.ObjectMeta.Finalizers = utils.RemoveString(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedCluster %v, %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
		}

		return reconcile.Result{}, nil
	}

	if !utils.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
		klog.Infof("adding ManagedClusterInfo Finalizer to ManagedCluster %v", cluster.Name)
		cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
		if err := r.client.Update(context.TODO(), cluster); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ManagedCluster %v, %v", cluster.Name, err)
			return reconcile.Result{}, err
		}
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(ctx, r.newClusterInfoByManagedCluster(cluster)); err != nil {
				klog.Warningf("will reconcile since failed to create ManagedClusterInfo %v, %v", cluster.Name, err)
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
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
		err = r.client.Update(ctx, clusterInfo)
		if err != nil {
			klog.Warningf("will reconcile since failed to update ManagedClusterInfo %v, %v", cluster.Name, err)
			return ctrl.Result{}, err
		}
	}

	// TODO: the conditions of managed cluster need to be deprecated.
	clusterConditions := cluster.Status.Conditions
	newClusterInfo := clusterInfo.DeepCopy()

	// empty node list in clusterInfo status when cluster is offline
	if utils.ClusterIsOffLine(clusterConditions) {
		newClusterInfo.Status.NodeList = []clusterinfov1beta1.NodeStatus{}
	}
	newClusterInfo.Status.Conditions = clusterConditions

	syncedCondition := meta.FindStatusCondition(clusterInfo.Status.Conditions, clusterinfov1beta1.ManagedClusterInfoSynced)
	if syncedCondition != nil {
		newClusterInfo.Status.Conditions = append(newClusterInfo.Status.Conditions, *syncedCondition)
	}

	if !reflect.DeepEqual(newClusterInfo.Status, clusterInfo.Status) {
		err = r.client.Status().Update(ctx, newClusterInfo)
		if err != nil {
			klog.Warningf("will reconcile since failed to update ManagedClusterInfo status %v, %v", cluster.Name, err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(name, namespace string) error {
	err := r.client.Delete(context.Background(), &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

func (r *Reconciler) newClusterInfoByManagedCluster(cluster *clusterv1.ManagedCluster) *clusterinfov1beta1.ManagedClusterInfo {
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
		Status: clusterinfov1beta1.ClusterInfoStatus{
			Conditions: cluster.Status.Conditions,
		},
	}
}
