package gc

import (
	"context"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"

	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

const (
	gcTimeout = 60 * time.Second
)

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create gc controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("gc-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource cluster
	err = c.Watch(&source.Kind{Type: &actionv1beta1.ManagedClusterAction{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	action := &actionv1beta1.ManagedClusterAction{}

	err := r.client.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if condition := conditions.FindStatusCondition(action.Status.Conditions, actionv1beta1.ConditionActionCompleted); condition != nil {
		sub := time.Since(condition.LastTransitionTime.Time)
		if sub < gcTimeout {
			return ctrl.Result{RequeueAfter: gcTimeout - sub}, nil
		}

		err := r.client.Delete(ctx, action)
		if err != nil {
			klog.Errorf("failed to delete cluster action %v in namespace %v", action.GetName(), action.GetNamespace())
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: gcTimeout}, nil
}
