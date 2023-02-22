package addoninstall

import (
	"context"
	"fmt"
	"strings"

	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	errorsutil "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
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

const (

	// AnnotationEnableHostedModeAddons is the key of annotation which indicates if the add-ons will be enabled
	// in hosted mode automatically for a managed cluster
	AnnotationEnableHostedModeAddons = "addon.open-cluster-management.io/enable-hosted-mode-addons"

	// AnnotationAddOnHostingClusterName is the annotation key of hosting cluster name for add-ons
	AnnotationAddOnHostingClusterName = "addon.open-cluster-management.io/hosting-cluster-name"

	// DefaultAddOnInstallNamespace is the default install namespace for addon in default mode
	DefaultAddOnInstallNamespace = "open-cluster-management-agent-addon"
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
	c, err := controller.New("addon-install-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// watch ManagedCluster as the primary resource
	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// watch ManagedClusterAddOn as additional resource
	err = c.Watch(&source.Kind{Type: &addonapiv1alpha1.ManagedClusterAddOn{}}, handler.EnqueueRequestsFromMapFunc(
		handler.MapFunc(func(a client.Object) []reconcile.Request {
			addOn, ok := a.(*addonapiv1alpha1.ManagedClusterAddOn)
			if !ok {
				klog.Error("invalid ManagedClusterAddOn object")
				return []reconcile.Request{}
			}

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

	// if cluster is deleting, do not install any addon
	if !cluster.DeletionTimestamp.IsZero() {
		klog.V(4).Infof("Cluster %q is deleting, skip addon deploy", cluster.Name)
		return ctrl.Result{}, nil
	}

	// do not install addon if addon automatic installation is disabled
	if value, ok := cluster.Annotations[constants.DisableAddonAutomaticInstallationAnnotationKey]; ok &&
		strings.EqualFold(value, "true") {
		klog.V(4).Infof("Cluster %q has annotation %q, skip addon deploy",
			cluster.Name, constants.DisableAddonAutomaticInstallationAnnotationKey)
		return ctrl.Result{}, nil
	}

	addOnHostingClusterName := getAddOnHostingClusterName(cluster)
	var errs []error
	for _, addonName := range r.addOnNames.List() {
		if err = r.applyAddOn(ctx, addonName, cluster.Name, addOnHostingClusterName); err != nil {
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

// applyAddOn only creates addon when it is missing, if user update the addon resource, it should not be reverted
func (r *Reconciler) applyAddOn(ctx context.Context, addonName, clusterName, hostingClusterName string) error {
	addOn := &addonapiv1alpha1.ManagedClusterAddOn{}
	err := r.client.Get(ctx, types.NamespacedName{Name: addonName, Namespace: clusterName}, addOn)

	if errors.IsNotFound(err) {
		addOn = &addonapiv1alpha1.ManagedClusterAddOn{
			ObjectMeta: metav1.ObjectMeta{
				Name:      addonName,
				Namespace: clusterName,
			},
			Spec: addonapiv1alpha1.ManagedClusterAddOnSpec{
				InstallNamespace: DefaultAddOnInstallNamespace,
			},
		}

		if len(hostingClusterName) > 0 {
			addOn.Spec.InstallNamespace = fmt.Sprintf("klusterlet-%s", clusterName)
			addOn.Annotations = map[string]string{
				AnnotationAddOnHostingClusterName: hostingClusterName,
			}
		}

		return r.client.Create(ctx, addOn)
	}

	return err
}

// getAddOnHostingClusterName returns the hosting cluster name for add-ons of the given managed cluster.
// An empty string is returned if the add-ons should be deployed in the default mode.
func getAddOnHostingClusterName(cluster *clusterv1.ManagedCluster) string {
	switch {
	case cluster == nil:
		return ""
	case cluster.Annotations[apiconstants.AnnotationKlusterletDeployMode] != "Hosted":
		return ""
	case !strings.EqualFold(cluster.Annotations[AnnotationEnableHostedModeAddons], "true"):
		return ""
	default:
		return cluster.Annotations[apiconstants.AnnotationKlusterletHostingClusterName]
	}
}
