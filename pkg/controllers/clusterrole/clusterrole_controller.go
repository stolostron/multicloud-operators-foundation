package clusterrole

import (
	"context"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
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

const (
	clusterRoleFinalizerName = "open-cluster-management.io/managedclusterrole"
)

type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	kubeClient kubernetes.Interface
}

func SetupWithManager(mgr manager.Manager, kubeClient kubernetes.Interface) error {
	if err := add(mgr, newReconciler(mgr, kubeClient)); err != nil {
		klog.Errorf("Failed to create clusterrole controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, kubeClient kubernetes.Interface) reconcile.Reconciler {
	return &Reconciler{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		kubeClient: kubeClient,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterrole-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1.ManagedCluster{},
		&handler.TypedEnqueueRequestForObject[*clusterv1.ManagedCluster]{}))
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1.ManagedCluster{}

	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !cluster.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(cluster.GetFinalizers(), clusterRoleFinalizerName) {
			if klog.V(4) {
				klog.Infof("deleting ManagedClusterRole %v", cluster.Name)
			}
			err := utils.DeleteClusterRole(r.kubeClient, utils.GenerateClusterRoleName(cluster.Name, "admin"))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole %v : %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
			err = utils.DeleteClusterRole(r.kubeClient, utils.GenerateClusterRoleName(cluster.Name, "view"))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole %v : %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
			if klog.V(4) {
				klog.Infof("removing ManagedClusterInfo Finalizer in ManagedCluster %v", cluster.Name)
			}
			cluster.ObjectMeta.Finalizers = utils.RemoveString(cluster.ObjectMeta.Finalizers, clusterRoleFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedCluster %v, %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if !utils.ContainsString(cluster.GetFinalizers(), clusterRoleFinalizerName) {
		if klog.V(4) {
			klog.Infof("adding ManagedClusterRole Finalizer to ManagedCluster %v", cluster.Name)
		}
		cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterRoleFinalizerName)
		if err := r.client.Update(context.TODO(), cluster); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ManagedCluster %v, %v", cluster.Name, err)
			return reconcile.Result{}, err
		}
	}

	// add clusterrole
	adminRole := buildAdminRole(cluster.Name)
	err = utils.ApplyClusterRole(r.kubeClient, adminRole)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", cluster.Name, err)
		return ctrl.Result{}, err
	}

	viewRole := buildViewRole(cluster.Name)
	err = utils.ApplyClusterRole(r.kubeClient, viewRole)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", cluster.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
