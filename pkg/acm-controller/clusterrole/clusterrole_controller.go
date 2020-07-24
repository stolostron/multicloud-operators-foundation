package clusterrole

import (
	"context"
	"reflect"

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
	clusterRoleFinalizerName = "open-cluster-management.io/managedclusterrole"
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
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterv1.ManagedCluster{}

	err := r.client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !cluster.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if helpers.ContainsString(cluster.GetFinalizers(), clusterRoleFinalizerName) {
			if klog.V(4) {
				klog.Infof("deleting ManagedClusterRole %v", cluster.Name)
			}
			err := r.deleteClusterRole(buildClusterRoleName(cluster.Name, "admin"))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole %v : %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
			err = r.deleteClusterRole(buildClusterRoleName(cluster.Name, "view"))
			if err != nil {
				klog.Warningf("will reconcile since failed to delete clusterrole %v : %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
			if klog.V(4) {
				klog.Infof("removing ManagedClusterInfo Finalizer in ManagedCluster %v", cluster.Name)
			}
			cluster.ObjectMeta.Finalizers = helpers.RemoveString(cluster.ObjectMeta.Finalizers, clusterRoleFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedCluster %v, %v", cluster.Name, err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if !helpers.ContainsString(cluster.GetFinalizers(), clusterRoleFinalizerName) {
		if klog.V(4) {
			klog.Infof("adding ManagedClusterRole Finalizer to ManagedCluster %v", cluster.Name)
		}
		cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterRoleFinalizerName)
		if err := r.client.Update(context.TODO(), cluster); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ManagedCluster %v, %v", cluster.Name, err)
			return reconcile.Result{}, err
		}
	}

	//add clusterrole
	adminRules := buildAdminRoleRules(cluster.Name)
	err = r.applyClusterRole(buildClusterRoleName(cluster.Name, "admin"), adminRules)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", cluster.Name, err)
		return ctrl.Result{}, err
	}
	viewRules := buildViewRoleRules(cluster.Name)
	err = r.applyClusterRole(buildClusterRoleName(cluster.Name, "view"), viewRules)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", cluster.Name, err)
		return ctrl.Result{}, err
	}

	//add label to clusternamespace
	clusterNamespace, err := r.kubeClient.CoreV1().Namespaces().Get(context.TODO(), cluster.Name, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("will reconcile since failed get clusternamespace %v, %v", cluster.Name, err)
		return ctrl.Result{}, err
	}

	var ClusterNameLabel = map[string]string{
		managedClusterKey: cluster.GetName(),
	}
	var modified = false
	utils.MergeMap(&modified, clusterNamespace.GetLabels(), ClusterNameLabel)

	if modified {
		_, err = r.kubeClient.CoreV1().Namespaces().Update(context.TODO(), clusterNamespace, metav1.UpdateOptions{})
		if err != nil {
			klog.Warningf("will reconcile since failed update clusternamespace %v, %v", cluster.Name, err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

//Delete cluster role
func (r *Reconciler) deleteClusterRole(clusterRoleName string) error {
	err := r.kubeClient.RbacV1().ClusterRoles().Delete(context.TODO(), clusterRoleName, metav1.DeleteOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

//apply cluster role
func (r *Reconciler) applyClusterRole(clusterRoleName string, rules []rbacv1.PolicyRule) error {
	clusterRole, err := r.kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), clusterRoleName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			clusterRole = &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterRoleName,
				},
				Rules: rules,
			}
			_, err = r.kubeClient.RbacV1().ClusterRoles().Create(context.TODO(), clusterRole, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if !reflect.DeepEqual(clusterRole.Rules, rules) {
		clusterRole.Rules = rules
		_, err := r.kubeClient.RbacV1().ClusterRoles().Update(context.TODO(), clusterRole, metav1.UpdateOptions{})
		return err
	}
	return nil
}

func buildClusterRoleName(clusterName, rule string) string {
	return "open-cluster-management:" + rule + ":" + clusterName
}
