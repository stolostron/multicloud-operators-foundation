package controllers

import (
	"context"

	"github.com/go-logr/logr"
	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
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
	SpokeDynamicClient  dynamic.Interface
	KubeControl         restutils.KubeControlInterface
	EnableImpersonation bool
}

func NewActionReconciler(client client.Client,
	log logr.Logger, scheme *runtime.Scheme,
	spokeDynamicClient dynamic.Interface,
	kubeControl restutils.KubeControlInterface,
	enableImpersonation bool) *ActionReconciler {
	return &ActionReconciler{
		Client:              client,
		Log:                 log,
		Scheme:              scheme,
		SpokeDynamicClient:  spokeDynamicClient,
		KubeControl:         kubeControl,
		EnableImpersonation: enableImpersonation,
	}
}

func (r *ActionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("action", req.NamespacedName)
	clusterAction := &actionv1beta1.ClusterAction{}

	err := r.Get(ctx, req.NamespacedName, clusterAction)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if conditions.IsStatusConditionTrue(clusterAction.Status.Conditions, actionv1beta1.ConditionActionCompleted) {
		return ctrl.Result{}, nil
	}

	if err := r.handleClusterAction(clusterAction); err != nil {
		log.Error(err, "unable to handle ClusterAction")
	}

	if err := r.Client.Status().Update(ctx, clusterAction); err != nil {
		log.Error(err, "unable to update status of ClusterAction")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ActionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&actionv1beta1.ClusterAction{}).
		Complete(r)
}
