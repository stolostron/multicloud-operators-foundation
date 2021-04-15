package clusterrole

import (
	"context"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	clustersetRoleFinalizerName = "open-cluster-management.io/managedclustersetrole"
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
	c, err := controller.New("clusterset-clusterrole-controller", mgr, controller.Options{Reconciler: r})
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
	ctx := context.Background()
	clusterset := &clusterv1alpha1.ManagedClusterSet{}

	err := r.client.Get(ctx, req.NamespacedName, clusterset)
	if err != nil {
		if errors.IsNotFound(err) {
			err := utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(req.Name, "admin"))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", req.Name, err)
				return reconcile.Result{}, err
			}
			err = utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(req.Name, "view"))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", req.Name, err)
				return reconcile.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//add clusterrole
	adminRules := buildAdminRoleRules(clusterset.Name)
	err = utils.ApplyClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "admin"), adminRules)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", clusterset.Name, err)
		return ctrl.Result{}, err
	}
	viewRules := buildViewRoleRules(clusterset.Name)
	err = utils.ApplyClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "view"), viewRules)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", clusterset.Name, err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
