package managedserviceaccount

import (
	"context"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	msav1beta1 "open-cluster-management.io/managed-serviceaccount/apis/authentication/v1beta1"
	msav1beta1client "open-cluster-management.io/managed-serviceaccount/pkg/generated/clientset/versioned/typed/authentication/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Reconciler struct {
	client     client.Client
	msaClient  msav1beta1client.AuthenticationV1beta1Interface
	scheme     *runtime.Scheme
	logMcaName string
}

func SetupWithManager(mgr manager.Manager, msaClient msav1beta1client.AuthenticationV1beta1Interface) error {
	if err := add(mgr, newReconciler(mgr, msaClient)); err != nil {
		klog.Errorf("Failed to create addon install controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, msaClient msav1beta1client.AuthenticationV1beta1Interface) *Reconciler {
	return &Reconciler{
		msaClient:  msaClient,
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		logMcaName: helpers.LogManagedServiceAccountName,
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
	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1.ManagedCluster{},
		&handler.TypedEnqueueRequestForObject[*clusterv1.ManagedCluster]{}))
	if err != nil {
		return err
	}

	// watch log managedClusterAddon
	// do not watch ManagedServiceAccount API, because the CRD will not be found if msa is not installed.
	err = c.Watch(source.Kind(mgr.GetCache(), &addonv1alpha1.ManagedClusterAddOn{},
		handler.TypedEnqueueRequestsFromMapFunc[*addonv1alpha1.ManagedClusterAddOn](
			func(ctx context.Context, a *addonv1alpha1.ManagedClusterAddOn) []reconcile.Request {
				if a.GetName() != helpers.MsaAddonName {
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

	addon := &addonv1alpha1.ManagedClusterAddOn{}
	err = r.client.Get(ctx, types.NamespacedName{Name: helpers.MsaAddonName, Namespace: cluster.Name}, addon)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	if !addon.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	requiredLogMsa := r.newLogMsa(cluster.Name)
	logMsa, err := r.msaClient.ManagedServiceAccounts(cluster.Name).Get(ctx, helpers.LogManagedServiceAccountName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = r.msaClient.ManagedServiceAccounts(cluster.Name).Create(ctx, requiredLogMsa, metav1.CreateOptions{})
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// logMsa := &msav1beta1.ManagedServiceAccount{}
	// err = r.client.Get(ctx, types.NamespacedName{Name: r.logMcaName, Namespace: cluster.Name}, logMsa)
	// if errors.IsNotFound(err) {
	// 	return ctrl.Result{}, r.client.Create(ctx, requiredLogMsa)
	// }
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	if apiequality.Semantic.DeepEqual(requiredLogMsa.Spec, logMsa.Spec) {
		return ctrl.Result{}, nil
	}
	logMsa.Spec = requiredLogMsa.Spec
	_, err = r.msaClient.ManagedServiceAccounts(cluster.Name).Update(ctx, logMsa, metav1.UpdateOptions{})
	return ctrl.Result{}, err
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
				Validity: metav1.Duration{Duration: time.Minute * 60 * 24 * 7},
			},
		},
	}
}
