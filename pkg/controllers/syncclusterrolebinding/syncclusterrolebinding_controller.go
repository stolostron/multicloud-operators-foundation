package syncclusterrolebinding

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterSetFinalizerName = "clusterrolebinding.finalizers.open-cluster-management.io"
)

type Reconciler struct {
	client              client.Client
	scheme              *runtime.Scheme
	clustersetToSubject map[string][]rbacv1.Subject
}

func SetupWithManager(mgr manager.Manager, clustersetToSubject map[string][]rbacv1.Subject) error {
	if err := add(mgr, newReconciler(mgr, clustersetToSubject)); err != nil {
		klog.Errorf("Failed to create ClusterRoleBinding controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clustersetToSubject map[string][]rbacv1.Subject) reconcile.Reconciler {
	return &Reconciler{
		client:              mgr.GetClient(),
		scheme:              mgr.GetScheme(),
		clustersetToSubject: clustersetToSubject,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterinfo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterv1.ManagedCluster{}

	clusterRolebindingName := generateClusterRoleBindingName(cluster.Name)
	clusterRoleName := generateClusterRoleName(cluster.Name)

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// Check DeletionTimestamp to determine if object is under deletion
	if !cluster.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(cluster.GetFinalizers(), clusterSetFinalizerName) {
			klog.Infof("deleting ClusterRoleBinding %v", cluster.Name)
			err := r.DeleteClusterRoleBinding(ctx, clusterRolebindingName)
			if err != nil && !errors.IsNotFound(err) {
				klog.Warningf("will reconcile since failed to delete ClusterRoleBinding %v : %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
			klog.Infof("removing ClusterRoleBinding Finalizer in ManagedCluster %v", cluster.Name)
			cluster.ObjectMeta.Finalizers = utils.RemoveString(cluster.ObjectMeta.Finalizers, clusterSetFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedCluster %v, %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
		}

		return reconcile.Result{}, nil
	}

	if !utils.ContainsString(cluster.GetFinalizers(), clusterSetFinalizerName) {
		klog.Infof("adding ClusterRoleBinding Finalizer to ManagedCluster %v", cluster.Name)
		cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterSetFinalizerName)
		if err := r.client.Update(context.TODO(), cluster); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ManagedCluster %v, %v", cluster.Name, err)
			return reconcile.Result{}, err
		}
	}

	shouldExist := false
	if _, ok := r.clustersetToSubject[cluster.Name]; ok {
		shouldExist = true
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err = r.client.Get(ctx, types.NamespacedName{Name: clusterRolebindingName}, clusterRoleBinding)
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	//if clusterrolebinding do not exist, create it
	if errors.IsNotFound(err) && shouldExist {
		err = r.CreateClusterRoleBinding(ctx, clusterRolebindingName, clusterRoleName, r.clustersetToSubject[cluster.Spec.Clusterset])
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !shouldExist {
		r.DeleteClusterRoleBinding(ctx, clusterRolebindingName)
	}

	if shouldUpdate(clusterRoleBinding.Subjects, r.clustersetToSubject[cluster.Spec.Clusterset]) {
		err = r.UpdateClusterRoleBinding(ctx, clusterRolebindingName, clusterRoleName)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func shouldUpdate(subjects1, subjects2 []rbacv1.Subject) bool {
	return true
}

func generateClusterRoleName(clusterName string) string {
	return "****role" + clusterName
}
func generateClusterRoleBindingName(clusterName string) string {
	return "****rolebinding" + clusterName
}

func (r *Reconciler) CreateClusterRoleBinding(ctx context.Context, clusterRoleBindingName, clusterRoleName string, subjects []rbacv1.Subject) error {
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: subjects,
	}

	err := r.client.Create(ctx, rb)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) UpdateClusterRoleBinding(ctx context.Context, clusterRoleBindingName, clusterRoleName string, subjects []rbacv1.Subject) error {
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: subjects,
	}
	err := r.client.Update(ctx, rb)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) DeleteClusterRoleBinding(ctx context.Context, clusterRoleBindingName string) error {
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
	}
	err := r.client.Delete(ctx, rb)
	if err != nil {
		return err
	}
	return nil
}
