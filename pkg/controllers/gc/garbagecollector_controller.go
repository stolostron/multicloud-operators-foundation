package gc

import (
	"context"
	"time"

	actionv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

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

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

const (
	gcTimeout                      = 60 * time.Second
	ClustersetFinalizerName string = "open-cluster-management.io/clusterset"
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

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	action := &actionv1beta1.ManagedClusterAction{}

	err := r.client.Get(ctx, req.NamespacedName, action)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if condition := meta.FindStatusCondition(action.Status.Conditions, actionv1beta1.ConditionActionCompleted); condition != nil {
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

// In 2.2.X, we add the finalier for some clusterrole(have admin permission to clusterset)
// In 2.3, the finalizer is not needed, so we need to clean these finalizers to handle upgrade case.
type CleanGarbageFinalizer struct {
	kubeClient kubernetes.Interface
}

func NewCleanGarbageFinalizer(kubeClient kubernetes.Interface) *CleanGarbageFinalizer {
	return &CleanGarbageFinalizer{
		kubeClient: kubeClient,
	}
}

// start a routine to sync the clusterrolebinding periodically.
func (c *CleanGarbageFinalizer) Run(stopCh <-chan struct{}) {
	utilwait.PollImmediateUntil(60*time.Second, c.clean, stopCh)
}

func (c *CleanGarbageFinalizer) clean() (bool, error) {
	clusterRolebindingList, err := c.kubeClient.RbacV1().ClusterRoles().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return false, nil
	}
	allFinalizerRmoved := true
	for _, clusterrole := range clusterRolebindingList.Items {
		if !utils.ContainsString(clusterrole.GetFinalizers(), ClustersetFinalizerName) {
			continue
		}

		klog.V(4).Infof("removing ClusterRoleBinding Finalizer in ClusterRole %v", clusterrole.Name)
		clusterrole.ObjectMeta.Finalizers = utils.RemoveString(clusterrole.ObjectMeta.Finalizers, ClustersetFinalizerName)
		_, err = c.kubeClient.RbacV1().ClusterRoles().Update(context.Background(), &clusterrole, v1.UpdateOptions{})
		if err != nil {
			allFinalizerRmoved = false
		}
	}
	if allFinalizerRmoved {
		return true, nil
	}
	return false, nil
}
