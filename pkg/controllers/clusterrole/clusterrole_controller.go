package clusterrole

import (
	"context"
	"fmt"
	"strings"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	if err := add(mgr, newReconciler(mgr, kubeClient), "clusterrole-controller"); err != nil {
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

var _ handler.EventHandler = &enqueueRequestForClusterRole{}

type enqueueRequestForClusterRole struct{}

func (e *enqueueRequestForClusterRole) enqueue(clusterroleName string, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	clusterName, err := getClusterNameFromClusterRoleName(clusterroleName)
	if err != nil {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}})
}

func (e *enqueueRequestForClusterRole) Create(ctx context.Context,
	evt event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
}

func (e *enqueueRequestForClusterRole) Update(ctx context.Context,
	evt event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.enqueue(evt.ObjectNew.GetName(), q)
}

func (e *enqueueRequestForClusterRole) Delete(ctx context.Context,
	evt event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	e.enqueue(evt.Object.GetName(), q)
}

func (e *enqueueRequestForClusterRole) Generic(ctx context.Context,
	evt event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, controllerName string) error {
	return ctrl.NewControllerManagedBy(mgr).Named(controllerName).Watches(
		&clusterv1.ManagedCluster{},
		&handler.EnqueueRequestForObject{},
	).WatchesMetadata(
		&rbacv1.ClusterRole{},
		&enqueueRequestForClusterRole{},
		builder.WithPredicates(predicate.Funcs{
			GenericFunc: func(e event.GenericEvent) bool { return false },
			DeleteFunc: func(e event.DeleteEvent) bool {
				return isValiedClusterRole(e.Object.GetName())
			},
			CreateFunc: func(e event.CreateEvent) bool { return false },
			UpdateFunc: func(e event.UpdateEvent) bool {
				return isValiedClusterRole(e.ObjectNew.GetName())
			},
		}),
	).Complete(r)
}

// isValiedClusterRole tells if the clusterrole is an OCM clusterrole
// The format of the clusterrole name is "open-cluster-management:admin:cluster-name" or "open-cluster-management:view:cluster-name"
func isValiedClusterRole(clusterRoleName string) bool {
	parts := strings.Split(clusterRoleName, ":")
	if len(parts) != 3 {
		return false
	}
	return parts[0] == "open-cluster-management" && (parts[1] == "admin" || parts[1] == "view")
}

// getClusterNameFromClusterRoleName get the cluster name from the clusterrole name
// The format of the clusterrole name is "open-cluster-management:admin:cluster-name" or "open-cluster-management:view:cluster-name"
func getClusterNameFromClusterRoleName(clusterRoleName string) (string, error) {
	parts := strings.Split(clusterRoleName, ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid clusterrole name: %s", clusterRoleName)
	}
	return parts[2], nil
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
