package hosteannotation

import (
	"context"

	"github.com/stolostron/multicloud-operators-foundation/pkg/addon"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
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
	scheme     *runtime.Scheme
	addOnNames sets.String
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr, "work-manager")); err != nil {
		klog.Errorf("Failed to create addon install controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, addOnNames ...string) *Reconciler {

	return &Reconciler{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		addOnNames: sets.NewString(addOnNames...),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *Reconciler) error {
	// Create a new controller
	c, err := controller.New("addon-hosted-annotation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// watch ManagedCluster as the primary resource
	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1.ManagedCluster{},
		&handler.TypedEnqueueRequestForObject[*clusterv1.ManagedCluster]{}))
	if err != nil {
		return err
	}

	// watch ManagedClusterAddOn as additional resource
	err = c.Watch(source.Kind(mgr.GetCache(), &addonapiv1alpha1.ManagedClusterAddOn{},
		handler.TypedEnqueueRequestsFromMapFunc[*addonapiv1alpha1.ManagedClusterAddOn](
			func(ctx context.Context, addOn *addonapiv1alpha1.ManagedClusterAddOn) []reconcile.Request {
				if !r.filterAddOn(addOn) {
					return []reconcile.Request{}
				}

				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: addOn.Namespace,
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

	var errs []error
	for _, addonName := range r.addOnNames.List() {
		if err = r.applyAddOn(ctx, addonName, cluster); err != nil {
			errs = append(errs, err)
		}
	}

	return ctrl.Result{}, errorsutil.NewAggregate(errs)
}

// filterAddOn returns true if the contoller should take care of the installation of the given add-on.
func (r *Reconciler) filterAddOn(addOn *addonapiv1alpha1.ManagedClusterAddOn) bool {
	if addOn == nil {
		return false
	}

	return r.addOnNames.Has(addOn.Name)
}

// applyAddOn only update addon annotation it is missing, if user update the addon resource, it should not be reverted
func (r *Reconciler) applyAddOn(ctx context.Context, addonName string, cluster *clusterv1.ManagedCluster) error {
	addOn := &addonapiv1alpha1.ManagedClusterAddOn{}
	err := r.client.Get(ctx, types.NamespacedName{Name: addonName, Namespace: cluster.Name}, addOn)

	if errors.IsNotFound(err) {
		return nil
	}

	_, hostingClusterName := addon.HostedClusterInfo(addOn, cluster)

	if len(hostingClusterName) == 0 {
		return nil
	}

	if _, ok := addOn.Annotations[addonapiv1alpha1.HostingClusterNameAnnotationKey]; ok {
		return nil
	}
	if addOn.Annotations == nil {
		addOn.Annotations = map[string]string{}
	}
	addOn.Annotations[addonapiv1alpha1.HostingClusterNameAnnotationKey] = hostingClusterName
	return r.client.Update(ctx, addOn)
}
