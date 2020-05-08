package clusterrbac

import (
	"context"
	"reflect"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"

	rbacv1helpers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common/rbac"
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
	clusterRBACFinalizerName = "clusterrbac.finalizers.open-cluster-management.io"
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

	// Watch for changes to primary resource cluster
	err = c.Watch(&source.Kind{Type: &clusterregistryv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.SpokeCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterregistryv1alpha1.Cluster{}

	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if cluster.GetDeletionTimestamp().IsZero() {
		if !containsString(cluster.GetFinalizers(), clusterRBACFinalizerName) {
			klog.Info("Finalizer not found for cluster. Adding finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to add finalizer to cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(cluster.GetFinalizers(), clusterRBACFinalizerName) {
			err := r.deleteExternalResources(cluster)
			if err != nil {
				return reconcile.Result{}, err
			}
			klog.Info("Removing Finalizer from cluster")
			cluster.ObjectMeta.Finalizers = removeString(cluster.ObjectMeta.Finalizers, clusterRBACFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to remove finalizer from cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if err = r.createOrUpdateRole(cluster.Name, cluster.Namespace); err != nil {
		klog.Errorf("Failed to update cluster role: %v", err)
		return ctrl.Result{}, err
	}

	if err = r.createOrUpdateRoleBinding(cluster.Name, cluster.Namespace); err != nil {
		klog.Errorf("Failed to update cluster rolebinding: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(cluster *clusterregistryv1alpha1.Cluster) error {
	err := r.kubeClient.RbacV1().RoleBindings(cluster.Namespace).Delete(roleName(cluster.Name), &metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	err = r.kubeClient.RbacV1().Roles(cluster.Namespace).Delete(roleName(cluster.Name), &metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// createOrUpdateRole create or update a role for a give cluster
func (r *Reconciler) createOrUpdateRole(clusterName, clusterNamespace string) error {
	role, err := r.kubeClient.RbacV1().Roles(clusterNamespace).Get(roleName(clusterName), metav1.GetOptions{})
	rules := buildRoleRules()
	if err != nil {
		if errors.IsNotFound(err) {
			hcmRole := &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:      roleName(clusterName),
					Namespace: clusterNamespace,
				},
				Rules: rules,
			}
			_, err = r.kubeClient.RbacV1().Roles(clusterNamespace).Create(hcmRole)
		}
		return err
	}

	if !reflect.DeepEqual(role.Rules, rules) {
		role.Rules = rules
		_, err := r.kubeClient.RbacV1().Roles(clusterNamespace).Update(role)
		return err
	}

	return nil
}

// createOrUpdateRoleBinding create or update a role binding for a given cluster
func (r *Reconciler) createOrUpdateRoleBinding(clusterName, clusterNamespace string) error {
	hcmRoleBinding := rbacv1helpers.NewRoleBinding(
		roleName(clusterName),
		clusterNamespace).Users("hcm:clusters:" + clusterNamespace + ":" + clusterName).BindingOrDie()

	binding, err := r.kubeClient.RbacV1().RoleBindings(clusterNamespace).Get(roleName(clusterName), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = r.kubeClient.RbacV1().RoleBindings(clusterNamespace).Create(&hcmRoleBinding)
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
		_, err = r.kubeClient.RbacV1().RoleBindings(clusterNamespace).Update(binding)
		return err
	}

	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func roleName(clusterName string) string {
	return namePrefix + clusterName
}
