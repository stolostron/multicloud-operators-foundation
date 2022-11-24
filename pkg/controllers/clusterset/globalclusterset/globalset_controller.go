package globalclusterset

import (
	"context"

	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	v1 "k8s.io/api/core/v1"
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

const (
	globalNamespaceAnnotation = "open-cluster-management.io/ns-create"
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

	err = c.Watch(&source.Kind{Type: &clusterv1beta2.ManagedClusterSet{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				clusterset, ok := a.(*clusterv1beta2.ManagedClusterSet)
				if !ok {
					klog.Error("clusterset handler received non-clusterset object")
					return []reconcile.Request{}
				}
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
	err = c.Watch(&source.Kind{Type: &clusterv1beta2.ManagedClusterSetBinding{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				clustersetbinding, ok := a.(*clusterv1beta2.ManagedClusterSetBinding)
				if !ok {
					klog.Error("clustersetbinding handler received non-clustersetbinding object")
					return []reconcile.Request{}
				}
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

	if _, ok := clusterset.Annotations[globalNamespaceAnnotation]; ok {
		if clusterset.Annotations[globalNamespaceAnnotation] != "true" {
			return ctrl.Result{}, nil
		}
	}

	err = r.applyGlobalNsAndSetBinding()
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(clusterset.Annotations) == 0 {
		clusterset.Annotations = map[string]string{
			globalNamespaceAnnotation: "false",
		}
	} else {
		clusterset.Annotations[globalNamespaceAnnotation] = "false"
	}
	return ctrl.Result{}, r.client.Update(ctx, clusterset, &client.UpdateOptions{})
}

// The applyGlobalNsAndSetBinding func apply the clustersetutils.GlobalSetNameSpace and
// apply the ManagedClusterSetBinding which bind the global clusterset in the namespace
func (r *Reconciler) applyGlobalNsAndSetBinding() error {
	globalSetNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clustersetutils.GlobalSetNameSpace,
		},
	}
	//Apply GlobalSet Namespace
	_, err := r.kubeClient.CoreV1().Namespaces().Get(context.TODO(), clustersetutils.GlobalSetNameSpace, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_, err := r.kubeClient.CoreV1().Namespaces().Create(context.TODO(), globalSetNs, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	//Apply clusterset Binding
	globalSetBinding := &clusterv1beta2.ManagedClusterSetBinding{}

	err = r.client.Get(context.TODO(), types.NamespacedName{Name: clustersetutils.GlobalSetName, Namespace: clustersetutils.GlobalSetNameSpace}, globalSetBinding)
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
	return nil
}
