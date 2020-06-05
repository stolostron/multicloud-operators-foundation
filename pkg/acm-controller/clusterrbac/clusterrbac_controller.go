package clusterrbac

import (
	"context"
	"reflect"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/helpers"

	"k8s.io/apimachinery/pkg/apis/meta/v1beta1"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistryv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterRBACFinalizerName = "managedclusterrbac.finalizers.open-cluster-management.io"
	namePrefix               = "acm-"
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

	// TODO: Deprecate clusterregistryv1alpha1.Cluster
	err = c.Watch(&source.Kind{Type: &clusterregistryv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{})
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
	// TODO: Deprecate ReconcileByCluster
	// TODO: return r.ReconcileByManagedCluster(req)
	return r.ReconcileByCluster(req)
}

// TODO: Deprecate ReconcileByCluster
func (r *Reconciler) ReconcileByCluster(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterregistryv1alpha1.Cluster{}

	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if cluster.GetDeletionTimestamp().IsZero() {
		if !helpers.ContainsString(cluster.GetFinalizers(), clusterRBACFinalizerName) {
			klog.Info("Finalizer not found for cluster. Adding finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to add finalizer to cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if helpers.ContainsString(cluster.GetFinalizers(), clusterRBACFinalizerName) {
			err := r.deleteExternalResources(roleName(cluster.Name))
			if err != nil {
				return reconcile.Result{}, err
			}
			klog.Info("Removing Finalizer from cluster")
			cluster.ObjectMeta.Finalizers = helpers.RemoveString(cluster.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to remove finalizer from cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}
	if len(cluster.Status.Conditions) == 0 {
		return reconcile.Result{}, nil
	}
	condition := cluster.Status.Conditions[len(cluster.Status.Conditions)-1]
	if condition.Type != clusterregistryv1alpha1.ClusterOK {
		return reconcile.Result{}, nil
	}

	if err = r.createOrUpdateRole(cluster.Name); err != nil {
		klog.Errorf("Failed to update cluster role: %v", err)
		return ctrl.Result{}, err
	}

	if err = r.createOrUpdateRoleBinding(cluster.Name); err != nil {
		klog.Errorf("Failed to update cluster rolebinding: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) ReconcileByManagedCluster(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterv1.ManagedCluster{}

	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if cluster.GetDeletionTimestamp().IsZero() {
		if !helpers.ContainsString(cluster.GetFinalizers(), clusterRBACFinalizerName) {
			klog.Info("Finalizer not found for cluster. Adding finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to add finalizer to cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if helpers.ContainsString(cluster.GetFinalizers(), clusterRBACFinalizerName) {
			err := r.deleteExternalResources(roleName(cluster.Name))
			if err != nil {
				return reconcile.Result{}, err
			}
			klog.Info("Removing Finalizer from cluster")
			cluster.ObjectMeta.Finalizers = helpers.RemoveString(cluster.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to remove finalizer from cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	conditionJoined := clusterv1.StatusCondition{}
	for _, condition := range cluster.Status.Conditions {
		if condition.Type == clusterv1.ManagedClusterConditionJoined {
			conditionJoined = condition
			break
		}
	}
	if conditionJoined.Status != v1beta1.ConditionTrue {
		return reconcile.Result{}, nil
	}

	if err = r.createOrUpdateRole(cluster.Name); err != nil {
		klog.Errorf("Failed to update cluster role: %v", err)
		return ctrl.Result{}, err
	}

	if err = r.createOrUpdateRoleBinding(cluster.Name); err != nil {
		klog.Errorf("Failed to update cluster rolebinding: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(clusterName string) error {
	err := r.kubeClient.RbacV1().RoleBindings(clusterName).Delete(roleName(clusterName), &metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	err = r.kubeClient.RbacV1().Roles(clusterName).Delete(roleName(clusterName), &metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// createOrUpdateRole create or update a role for a give cluster
func (r *Reconciler) createOrUpdateRole(clusterName string) error {
	role, err := r.kubeClient.RbacV1().Roles(clusterName).Get(roleName(clusterName), metav1.GetOptions{})
	rules := buildRoleRules()
	if err != nil {
		if errors.IsNotFound(err) {
			hcmRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName(clusterName),
					Namespace: clusterName,
				},
				Rules: rules,
			}
			_, err = r.kubeClient.RbacV1().Roles(clusterName).Create(hcmRole)
		}
		return err
	}

	if !reflect.DeepEqual(role.Rules, rules) {
		role.Rules = rules
		_, err := r.kubeClient.RbacV1().Roles(clusterName).Update(role)
		return err
	}

	return nil
}

// createOrUpdateRoleBinding create or update a role binding for a given cluster
func (r *Reconciler) createOrUpdateRoleBinding(clusterName string) error {
	hcmRoleBinding := NewRoleBinding(
		roleName(clusterName), clusterName).Users("hcm:clusters:" + clusterName + ":" + clusterName).BindingOrDie()

	binding, err := r.kubeClient.RbacV1().RoleBindings(clusterName).Get(roleName(clusterName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = r.kubeClient.RbacV1().RoleBindings(clusterName).Create(&hcmRoleBinding)
		}
		return err
	}

	needUpdate := false
	if !reflect.DeepEqual(hcmRoleBinding.RoleRef, binding.RoleRef) {
		needUpdate = true
		binding.RoleRef = hcmRoleBinding.RoleRef
	}
	if reflect.DeepEqual(hcmRoleBinding.Subjects, binding.Subjects) {
		needUpdate = true
		binding.Subjects = hcmRoleBinding.Subjects
	}
	if needUpdate {
		_, err = r.kubeClient.RbacV1().RoleBindings(clusterName).Update(binding)
		return err
	}

	return nil
}

func roleName(clusterName string) string {
	return namePrefix + clusterName
}
