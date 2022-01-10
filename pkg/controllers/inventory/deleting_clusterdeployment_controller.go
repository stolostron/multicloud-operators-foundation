package inventory

import (
	"context"

	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	inventoryv1alpha1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// BareMetalAssetFinalizer is the finalizer used on BareMetalAsset resource
const BareMetalAssetFinalizer = "baremetalasset.inventory.open-cluster-management.io"

// newReconciler returns a new reconcile.Reconciler
func newCDReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileClusterDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func addCDReconciler(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("baremetalasset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BareMetalAsset
	err = c.Watch(&source.Kind{Type: &hivev1.ClusterDeployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &inventoryv1alpha1.BareMetalAsset{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				bma, ok := a.Object.(*inventoryv1alpha1.BareMetalAsset)
				if !ok {
					// not a Deployment, returning empty
					klog.Error("bma handler received bma object")
					return []reconcile.Request{}
				}
				var requests []reconcile.Request
				if bma.Spec.ClusterDeployment.Name != "" && bma.Spec.ClusterDeployment.Namespace != "" {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      bma.Spec.ClusterDeployment.Name,
							Namespace: bma.Spec.ClusterDeployment.Namespace,
						},
					})
				}
				return requests
			}),
		})

	return nil
}

// ReconcileClusterDeployment reconciles a ClusterDeployment object
type ReconcileClusterDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileClusterDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling ClusterDeployment")
	// Fetch the BareMetalAsset instance
	instance := &hivev1.ClusterDeployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if !instance.GetDeletionTimestamp().IsZero() {
		return reconcile.Result{}, r.removeClusterDeploymentFinalizer(instance)
	}

	bmas := &inventoryv1alpha1.BareMetalAssetList{}
	err = r.client.List(context.TODO(), bmas,
		client.MatchingLabels{
			ClusterDeploymentNameLabel:      instance.Name,
			ClusterDeploymentNamespaceLabel: instance.Namespace,
		})
	if err != nil {
		return reconcile.Result{}, err
	}

	// only add finalizer when there is ref bma
	if len(bmas.Items) == 0 {
		return reconcile.Result{}, nil
	}

	if !contains(instance.GetFinalizers(), BareMetalAssetFinalizer) {
		klog.Info("Finalizer not found for BareMetalAsset. Adding finalizer")
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, BareMetalAssetFinalizer)
		return reconcile.Result{}, r.client.Update(context.TODO(), instance)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileClusterDeployment) removeClusterDeploymentFinalizer(instance *hivev1.ClusterDeployment) error {
	if !contains(instance.GetFinalizers(), BareMetalAssetFinalizer) {
		return nil
	}

	bmas := &inventoryv1alpha1.BareMetalAssetList{}
	err := r.client.List(context.TODO(), bmas,
		client.MatchingLabels{
			ClusterDeploymentNameLabel:      instance.Name,
			ClusterDeploymentNamespaceLabel: instance.Namespace,
		})
	if err != nil {
		return err
	}

	errs := []error{}
	for _, bma := range bmas.Items {
		bmaCopy := bma.DeepCopy()
		if len(bmaCopy.Spec.ClusterDeployment.Name) == 0 && len(bmaCopy.Spec.ClusterDeployment.Namespace) == 0 {
			continue
		}
		bmaCopy.Spec.ClusterDeployment.Name = ""
		bmaCopy.Spec.ClusterDeployment.Namespace = ""
		err := r.client.Update(context.TODO(), bmaCopy)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return utils.NewMultiLineAggregate(errs)
	}

	instance.ObjectMeta.Finalizers = remove(instance.ObjectMeta.Finalizers, BareMetalAssetFinalizer)
	return r.client.Update(context.TODO(), instance)
}
