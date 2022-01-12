package clusterrbac

import (
	"context"
	"reflect"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	clusterRBACFinalizerName = "managedclusterrbac.finalizers.open-cluster-management.io"
	subjectPrefix            = "system:open-cluster-management:"
)

type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	kubeClient kubernetes.Interface
}

func SetupWithManager(mgr manager.Manager, kubeClient kubernetes.Interface) error {
	if err := add(mgr, newReconciler(mgr, kubeClient)); err != nil {
		klog.Errorf("Failed to create clusterrbac controller, %v", err)
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
	c, err := controller.New("clusterrbac-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1beta1.ManagedClusterInfo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}

	err := r.client.Get(ctx, req.NamespacedName, clusterInfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !clusterInfo.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(clusterInfo.GetFinalizers(), clusterRBACFinalizerName) {
			klog.Infof("deleting rbac for ManagedCluster agent %v", clusterInfo.Name)
			err := r.deleteExternalResources(roleName(clusterInfo.Name))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete role and rolebinding %v, %v", roleName(clusterInfo.Name), err)
				return reconcile.Result{}, err
			}
			klog.Infof("removing rbac Finalizer in ManagedClusterInfo %v", clusterInfo.Name)
			clusterInfo.ObjectMeta.Finalizers = utils.RemoveString(clusterInfo.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), clusterInfo); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedClusterInfo %v, %v", clusterInfo.Name, err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if clusterInfo.GetDeletionTimestamp().IsZero() {
		if !utils.ContainsString(clusterInfo.GetFinalizers(), clusterRBACFinalizerName) {
			klog.Infof("adding rbac Finalizer to ManagedClusterInfo %v", clusterInfo.Name)
			clusterInfo.ObjectMeta.Finalizers = append(clusterInfo.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), clusterInfo); err != nil {
				klog.Warningf("will reconcile since failed to add finalizer to ManagedClusterInfo %v, %v", clusterInfo.Name, err)
				return reconcile.Result{}, err
			}
		}
	}

	if err = r.createOrUpdateRole(clusterInfo.Name); err != nil {
		klog.Warningf("will reconcile since failed to create/update role %v, %v", roleName(clusterInfo.Name), err)
		return ctrl.Result{}, err
	}

	if err = r.createOrUpdateRoleBinding(clusterInfo.Name); err != nil {
		klog.Warningf("will reconcile since failed to create/update rolebinding %v, %v", roleName(clusterInfo.Name), err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(clusterName string) error {
	err := r.kubeClient.RbacV1().RoleBindings(clusterName).Delete(context.TODO(), roleName(clusterName), metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	err = r.kubeClient.RbacV1().Roles(clusterName).Delete(context.TODO(), roleName(clusterName), metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// createOrUpdateRole create or update a role for a give cluster
func (r *Reconciler) createOrUpdateRole(clusterName string) error {
	role, err := r.kubeClient.RbacV1().Roles(clusterName).Get(context.TODO(), roleName(clusterName), metav1.GetOptions{})
	rules := buildRoleRules()
	if err != nil {
		if errors.IsNotFound(err) {
			acmRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName(clusterName),
					Namespace: clusterName,
				},
				Rules: rules,
			}
			_, err = r.kubeClient.RbacV1().Roles(clusterName).Create(context.TODO(), acmRole, metav1.CreateOptions{})
		}
		return err
	}

	if !reflect.DeepEqual(role.Rules, rules) {
		role.Rules = rules
		_, err := r.kubeClient.RbacV1().Roles(clusterName).Update(context.TODO(), role, metav1.UpdateOptions{})
		return err
	}

	return nil
}

// createOrUpdateRoleBinding create or update a role binding for a given cluster
func (r *Reconciler) createOrUpdateRoleBinding(clusterName string) error {
	roleName := roleName(clusterName)
	acmRoleBinding := NewRoleBinding(roleName, clusterName).Groups(subjectPrefix + clusterName).BindingOrDie()

	// role and rolebinding have the same name
	binding, err := r.kubeClient.RbacV1().RoleBindings(clusterName).Get(context.TODO(), roleName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = r.kubeClient.RbacV1().RoleBindings(clusterName).Create(context.TODO(), &acmRoleBinding, metav1.CreateOptions{})
		}
		return err
	}

	needUpdate := false
	if !reflect.DeepEqual(acmRoleBinding.RoleRef, binding.RoleRef) {
		needUpdate = true
		binding.RoleRef = acmRoleBinding.RoleRef
	}
	if !reflect.DeepEqual(acmRoleBinding.Subjects, binding.Subjects) {
		needUpdate = true
		binding.Subjects = acmRoleBinding.Subjects
	}
	if needUpdate {
		_, err = r.kubeClient.RbacV1().RoleBindings(clusterName).Update(context.TODO(), binding, metav1.UpdateOptions{})
		return err
	}

	return nil
}

func roleName(clusterName string) string {
	return clusterName + ":managed-cluster-foundation"
}
