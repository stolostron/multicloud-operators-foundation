package managedserviceaccount

import (
	"context"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	msav1beta1 "open-cluster-management.io/managed-serviceaccount/apis/authentication/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	LogManagedServiceAccountName = "klusterlet-addon-workmgr-log"
)

type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	logMcaName string
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create addon install controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *Reconciler {
	return &Reconciler{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		logMcaName: LogManagedServiceAccountName,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *Reconciler) error {
	// Create a new controller
	c, err := controller.New("log-managedServiceAccount-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// watch ManagedCluster as the primary resource
	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1.ManagedCluster{}), &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// watch log ManagedServiceAccount
	err = c.Watch(source.Kind(mgr.GetCache(), &msav1beta1.ManagedServiceAccount{}), handler.EnqueueRequestsFromMapFunc(
		handler.MapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
			if a.GetName() != LogManagedServiceAccountName {
				return []reconcile.Request{}
			}

			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name: a.GetNamespace(),
					},
				},
			}
		}),
	))
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1.ManagedCluster{}
	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if !cluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	labels := cluster.GetLabels()
	if len(labels) == 0 {
		return ctrl.Result{}, nil
	}

	if _, ok := labels[helpers.MsaAddOnFeatureLabel]; !ok {
		return ctrl.Result{}, nil
	}

	requiredLogMsa := r.newLogMsa(cluster.Name)
	logMsa := &msav1beta1.ManagedServiceAccount{}
	err = r.client.Get(ctx, types.NamespacedName{Name: r.logMcaName, Namespace: cluster.Name}, logMsa)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, r.client.Create(ctx, requiredLogMsa)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if apiequality.Semantic.DeepEqual(requiredLogMsa.Spec, logMsa.Spec) {
		return ctrl.Result{}, nil
	}
	logMsa.Spec = requiredLogMsa.Spec
	return ctrl.Result{}, r.client.Update(ctx, logMsa)
}

func (r *Reconciler) newLogMsa(clusterName string) *msav1beta1.ManagedServiceAccount {
	return &msav1beta1.ManagedServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.logMcaName,
			Namespace: clusterName,
		},
		Spec: msav1beta1.ManagedServiceAccountSpec{
			Rotation: msav1beta1.ManagedServiceAccountRotation{
				Enabled:  true,
				Validity: metav1.Duration{Duration: time.Minute * 365 * 24 * 60},
			},
		},
	}
}
