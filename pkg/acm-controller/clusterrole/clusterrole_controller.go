package clusterrole

import (
	"context"
	"reflect"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/helpers"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
	clusterRoleFinalizerName = "managedclusterrole.finalizers.open-cluster-management.io"
	managedClusterKey        = "cluster.open-cluster-management.io/managedCluster"
)

type Reconciler struct {
	client     client.Client
	scheme     *runtime.Scheme
	kubeClient kubernetes.Interface
}

func SetupWithManager(mgr manager.Manager, kubeClient kubernetes.Interface) error {
	if err := add(mgr, newReconciler(mgr, kubeClient)); err != nil {
		klog.Errorf("Failed to create clusterrole controller, %v", err)
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
	c, err := controller.New("clusterrole-controller", mgr, controller.Options{Reconciler: r})
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
		if helpers.ContainsString(clusterInfo.GetFinalizers(), clusterRoleFinalizerName) {
			klog.Infof("deleting clusterrole for ManagedCluster agent %v", clusterInfo.Name)
			err := r.deleteExternalResources(clusterInfo.Name)
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole, %v", clusterInfo.Name, err)
				return reconcile.Result{}, err
			}
			klog.Infof("removing clusterrole Finalizer in ManagedClusterInfo %v", clusterInfo.Name)
			clusterInfo.ObjectMeta.Finalizers = helpers.RemoveString(clusterInfo.ObjectMeta.Finalizers, clusterRoleFinalizerName)
			if err := r.client.Update(context.TODO(), clusterInfo); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedClusterInfo %v, %v", clusterInfo.Name, err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if clusterInfo.GetDeletionTimestamp().IsZero() {
		if !helpers.ContainsString(clusterInfo.GetFinalizers(), clusterRoleFinalizerName) {
			klog.Infof("adding clusterrole Finalizer to ManagedClusterInfo %v", clusterInfo.Name)
			clusterInfo.ObjectMeta.Finalizers = append(clusterInfo.ObjectMeta.Finalizers, clusterRoleFinalizerName)
			if err := r.client.Update(context.TODO(), clusterInfo); err != nil {
				klog.Warningf("will reconcile since failed to add finalizer to ManagedClusterInfo %v, %v", clusterInfo.Name, err)
				return reconcile.Result{}, err
			}
		}
	}
	//add cluster label
	managedCluster := &clusterv1.ManagedCluster{}
	err = r.client.Get(ctx, req.NamespacedName, managedCluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var managedClusterNameLabel = map[string]string{
		managedClusterKey: managedCluster.GetName(),
	}
	labelSelector := &metav1.LabelSelector{
		MatchLabels: managedClusterNameLabel,
	}
	if !utils.MatchLabelForLabelSelector(managedCluster.GetLabels(), labelSelector) {
		labels := utils.AddLabel(managedCluster.GetLabels(), managedClusterKey, managedCluster.GetName())
		managedCluster.SetLabels(labels)
	}
	err = r.client.Update(ctx, managedCluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	//add clusterrole
	if err = r.createOrUpdateClusterRole(clusterInfo.Name); err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", clusterInfo.Name, err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(clusterName string) error {
	err := r.kubeClient.RbacV1().ClusterRoles().Delete(clusterAdminRoleName(clusterName), &metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	err = r.kubeClient.RbacV1().ClusterRoles().Delete(clusterViewRoleName(clusterName), &metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

// createOrUpdateClusterRole create or update a clusterrole for a give cluster
func (r *Reconciler) createOrUpdateClusterRole(clusterName string) error {
	clusterAdminRole, err := r.kubeClient.RbacV1().ClusterRoles().Get(clusterAdminRoleName(clusterName), metav1.GetOptions{})
	adminRules := buildAdminRoleRules(clusterName)
	if err != nil {
		if errors.IsNotFound(err) {
			acmRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterAdminRoleName(clusterName),
				},
				Rules: adminRules,
			}
			_, err = r.kubeClient.RbacV1().ClusterRoles().Create(acmRole)
		}
		return err
	}

	if !reflect.DeepEqual(clusterAdminRole.Rules, adminRules) {
		clusterAdminRole.Rules = adminRules
		_, err := r.kubeClient.RbacV1().ClusterRoles().Update(clusterAdminRole)
		return err
	}

	clusterViewRole, err := r.kubeClient.RbacV1().ClusterRoles().Get(clusterViewRoleName(clusterName), metav1.GetOptions{})
	viewRules := buildViewRoleRules(clusterName)
	if err != nil {
		if errors.IsNotFound(err) {
			acmRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterViewRoleName(clusterName),
				},
				Rules: viewRules,
			}
			_, err = r.kubeClient.RbacV1().ClusterRoles().Create(acmRole)
		}
		return err
	}

	if !reflect.DeepEqual(clusterViewRole.Rules, viewRules) {
		clusterViewRole.Rules = viewRules
		_, err := r.kubeClient.RbacV1().ClusterRoles().Update(clusterViewRole)
		return err
	}
	return nil
}

func clusterAdminRoleName(clusterName string) string {
	return "open-cluster-management:admin:managed-cluster-" + clusterName
}

func clusterViewRoleName(clusterName string) string {
	return "open-cluster-management:view:managed-cluster-" + clusterName
}
