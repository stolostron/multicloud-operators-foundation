package controllers

import (
	"context"

	"github.com/go-logr/logr"
	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	restutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/rest"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ActionReconciler reconciles a Action object
type ActionReconciler struct {
	client.Client
	Log                 logr.Logger
	Scheme              *runtime.Scheme
	DynamicClient       dynamic.Interface
	KubeControl         restutils.KubeControlInterface
	EnableImpersonation bool
}

func NewActionReconciler(client client.Client,
	log logr.Logger, scheme *runtime.Scheme,
	dynamicClient dynamic.Interface,
	kubeControl restutils.KubeControlInterface,
	enableImpersonation bool) *ActionReconciler {
	return &ActionReconciler{
		Client:              client,
		Log:                 log,
		Scheme:              scheme,
		DynamicClient:       dynamicClient,
		KubeControl:         kubeControl,
		EnableImpersonation: enableImpersonation,
	}
}

func (r *ActionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ManagedClusterAction", req.NamespacedName)
	action := &actionv1beta1.ManagedClusterAction{}

	err := r.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if meta.IsStatusConditionTrue(action.Status.Conditions, actionv1beta1.ConditionActionCompleted) {
		return ctrl.Result{}, nil
	}

	if err := r.handleAction(action); err != nil {
		log.Error(err, "unable to handle ManagedClusterAction")
	}

	if err := r.Client.Status().Update(ctx, action); err != nil {
		log.Error(err, "unable to update status of ManagedClusterAction")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&actionv1beta1.ManagedClusterAction{}).
		Complete(r)
}
