package globalclusterset

import (
	"context"

	"github.com/stolostron/multicloud-operators-foundation/pkg/constants"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	v1 "k8s.io/api/core/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// This controller apply a namespace and clustersetbinding for global set
type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	kubeClient kubernetes.Interface
}

func SetupWithManager(mgr manager.Manager, kubeClient kubernetes.Interface) error {
	if err := add(mgr, newReconciler(mgr, kubeClient)); err != nil {
		klog.Errorf("Failed to create global clusterset controller, %v", err)
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
	c, err := controller.New("global-clusterset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta2.ManagedClusterSet{},
		handler.TypedEnqueueRequestsFromMapFunc[*clusterv1beta2.ManagedClusterSet](
			func(ctx context.Context, clusterset *clusterv1beta2.ManagedClusterSet) []reconcile.Request {
				if clusterset.Spec.ClusterSelector.SelectorType != clusterv1beta2.LabelSelector {
					return []reconcile.Request{}
				}
				if clusterset.Name != clustersetutils.GlobalSetName {
					return []reconcile.Request{}
				}
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: clustersetutils.GlobalSetName,
						},
					},
				}
			}),
	),
	)
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta2.ManagedClusterSetBinding{},
		handler.TypedEnqueueRequestsFromMapFunc[*clusterv1beta2.ManagedClusterSetBinding](
			func(ctx context.Context, clustersetbinding *clusterv1beta2.ManagedClusterSetBinding) []reconcile.Request {
				if clustersetbinding.Namespace != clustersetutils.GlobalSetNameSpace {
					return []reconcile.Request{}
				}
				if clustersetbinding.Name != clustersetutils.GlobalSetName {
					return []reconcile.Request{}
				}
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name:      clustersetutils.GlobalSetName,
							Namespace: clustersetutils.GlobalSetNameSpace,
						},
					},
				}
			}),
	),
	)
	if err != nil {
		return err
	}

	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta1.Placement{},
		handler.TypedEnqueueRequestsFromMapFunc[*clusterv1beta1.Placement](
			func(ctx context.Context, placement *clusterv1beta1.Placement) []reconcile.Request {
				if placement.Namespace != clustersetutils.GlobalSetNameSpace {
					return []reconcile.Request{}
				}
				if placement.Name != clustersetutils.GlobalPlacementName {
					return []reconcile.Request{}
				}
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Name: clustersetutils.GlobalSetName,
						},
					},
				}
			}),
	),
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterset := &clusterv1beta2.ManagedClusterSet{}

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, clusterset)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !clusterset.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	_, ok := clusterset.Annotations[constants.GlobalNamespaceAnnotation]
	if !ok {
		// this is to prevent the namespace can not be deleted when uninstall the mce
		// issue: https://github.com/stolostron/backlog/issues/24532
		err = r.applyNamespace()
		if err != nil {
			return ctrl.Result{}, err
		}

		if clusterset.Annotations == nil {
			clusterset.Annotations = map[string]string{}
		}
		clusterset.Annotations[constants.GlobalNamespaceAnnotation] = "true"
		err = r.client.Update(ctx, clusterset, &client.UpdateOptions{})
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.applyBindingAndPlacement()
}

// applyNamespace creates the clustersetutils.GlobalSetNameSpace if it does not exist
func (r *Reconciler) applyNamespace() error {
	globalSetNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clustersetutils.GlobalSetNameSpace,
		},
	}
	// Apply GlobalSet Namespace
	_, err := r.kubeClient.CoreV1().Namespaces().Get(
		context.TODO(), clustersetutils.GlobalSetNameSpace, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_, err := r.kubeClient.CoreV1().Namespaces().Create(context.TODO(), globalSetNs, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

// applyBindingAndPlacement func will apply the following resources if the namespace exists:
//  1. apply the ManagedClusterSetBinding which bind the global clusterset in the namespace
//  2. apply a placement which select all clusters
func (r *Reconciler) applyBindingAndPlacement() error {
	// Apply GlobalSet Namespace
	_, err := r.kubeClient.CoreV1().Namespaces().Get(
		context.TODO(), clustersetutils.GlobalSetNameSpace, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		klog.Infof("GlobalSet Namespace %s does not exist, skip applying binding and placement",
			clustersetutils.GlobalSetNameSpace)
		return nil
	}

	// Apply clusterset Binding
	globalSetBinding := &clusterv1beta2.ManagedClusterSetBinding{}

	err = r.client.Get(context.TODO(),
		types.NamespacedName{
			Name:      clustersetutils.GlobalSetName,
			Namespace: clustersetutils.GlobalSetNameSpace},
		globalSetBinding)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		setBinding := &clusterv1beta2.ManagedClusterSetBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clustersetutils.GlobalSetName,
				Namespace: clustersetutils.GlobalSetNameSpace,
			},
			Spec: clusterv1beta2.ManagedClusterSetBindingSpec{
				ClusterSet: clustersetutils.GlobalSetName,
			},
		}
		err := r.client.Create(context.TODO(), setBinding, &client.CreateOptions{})
		if err != nil {
			return err
		}
	}

	// Apply global placement
	globalPlacement := &clusterv1beta1.Placement{}

	err = r.client.Get(context.TODO(),
		types.NamespacedName{
			Name:      clustersetutils.GlobalPlacementName,
			Namespace: clustersetutils.GlobalSetNameSpace},
		globalPlacement)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		globalPlacement = &clusterv1beta1.Placement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clustersetutils.GlobalPlacementName,
				Namespace: clustersetutils.GlobalSetNameSpace,
			},
			Spec: clusterv1beta1.PlacementSpec{
				ClusterSets: []string{clustersetutils.GlobalSetName},
				Tolerations: []clusterv1beta1.Toleration{
					{
						Key: clusterv1.ManagedClusterTaintUnreachable,
					},
					{
						Key: clusterv1.ManagedClusterTaintUnavailable,
					},
				},
			},
		}
		err := r.client.Create(context.TODO(), globalPlacement, &client.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
