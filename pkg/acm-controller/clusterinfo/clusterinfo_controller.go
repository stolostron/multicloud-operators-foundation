package clusterinfo

import (
	"context"
	"reflect"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/helpers"

	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterregistryv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterFinalizerName = "managedclusterinfo.finalizers.open-cluster-management.io"
)

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
	caData []byte
}

func SetupWithManager(mgr manager.Manager, caData []byte) error {
	if err := add(mgr, newReconciler(mgr, caData)); err != nil {
		klog.Errorf("Failed to create ManagedClusterInfo controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, caData []byte) reconcile.Reconciler {
	return &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		caData: caData,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterinfo-controller", mgr, controller.Options{Reconciler: r})
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
		if !helpers.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			klog.Info("Finalizer not found for cluster. Adding finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to add finalizer to cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if helpers.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			err := r.deleteExternalResources(cluster.Name, cluster.Namespace)
			if err != nil {
				return reconcile.Result{}, err
			}
			klog.Info("Removing Finalizer")
			cluster.ObjectMeta.Finalizers = helpers.RemoveString(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to remove finalizer from cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, clusterInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(ctx, r.newClusterInfoByCluster(cluster)); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	endpoint := ""
	if len(cluster.Spec.KubernetesAPIEndpoints.ServerEndpoints) != 0 {
		endpoint = cluster.Spec.KubernetesAPIEndpoints.ServerEndpoints[0].ServerAddress
	}

	if !reflect.DeepEqual(r.caData, clusterInfo.Spec.LoggingCA) ||
		clusterInfo.Spec.MasterEndpoint != endpoint {
		clusterInfo.Spec.LoggingCA = r.caData
		clusterInfo.Spec.MasterEndpoint = endpoint
		err = r.client.Update(ctx, clusterInfo)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if len(cluster.Status.Conditions) == 0 {
		return ctrl.Result{}, nil
	}
	condition := cluster.Status.Conditions[len(cluster.Status.Conditions)-1]
	if condition.Type == clusterregistryv1alpha1.ClusterOK {
		clusterInfo.Status.Conditions = []clusterv1.StatusCondition{
			{
				Type:    clusterv1.ManagedClusterConditionJoined,
				Status:  metav1.ConditionTrue,
				Reason:  "ClusterReady",
				Message: "cluster is posting ready status",
				LastTransitionTime: metav1.Time{
					Time: time.Now(),
				},
			},
		}
	} else {
		clusterInfo.Status.Conditions = []clusterv1.StatusCondition{
			{
				Type:    clusterv1.ManagedClusterConditionJoined,
				Status:  metav1.ConditionFalse,
				Reason:  "ClusterOffline",
				Message: "cluster is offline",
				LastTransitionTime: metav1.Time{
					Time: time.Now(),
				},
			},
		}
	}
	err = r.client.Status().Update(ctx, clusterInfo)
	if err != nil {
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
		if !helpers.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			klog.Info("Finalizer not found for cluster. Adding finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to add finalizer to cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if helpers.ContainsString(cluster.GetFinalizers(), clusterFinalizerName) {
			err := r.deleteExternalResources(cluster.Name, cluster.Name)
			if err != nil {
				return reconcile.Result{}, err
			}
			klog.Info("Removing Finalizer")
			cluster.ObjectMeta.Finalizers = helpers.RemoveString(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to remove finalizer from cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(ctx, r.newClusterInfoByManagedCluster(cluster)); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	endpoint := ""
	if len(cluster.Spec.ManagedClusterClientConfigs) != 0 {
		endpoint = cluster.Spec.ManagedClusterClientConfigs[0].URL
	}

	if !reflect.DeepEqual(r.caData, clusterInfo.Spec.LoggingCA) ||
		clusterInfo.Spec.MasterEndpoint != endpoint {
		clusterInfo.Spec.LoggingCA = r.caData
		clusterInfo.Spec.MasterEndpoint = endpoint
		err = r.client.Update(ctx, clusterInfo)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if reflect.DeepEqual(cluster.Status.Conditions, clusterInfo.Status.Conditions) {
		clusterInfo.Status.Conditions = cluster.Status.Conditions
		err = r.client.Status().Update(ctx, clusterInfo)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(name, namespace string) error {
	err := r.client.Delete(context.Background(), &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// TODO: Deprecate newClusterInfoByCluster
func (r *Reconciler) newClusterInfoByCluster(cluster *clusterregistryv1alpha1.Cluster) *clusterinfov1beta1.ManagedClusterInfo {
	endpoint := ""
	if len(cluster.Spec.KubernetesAPIEndpoints.ServerEndpoints) != 0 {
		endpoint = cluster.Spec.KubernetesAPIEndpoints.ServerEndpoints[0].ServerAddress
	}

	return &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			MasterEndpoint: endpoint,
			LoggingCA:      r.caData,
		},
	}
}

func (r *Reconciler) newClusterInfoByManagedCluster(cluster *clusterv1.ManagedCluster) *clusterinfov1beta1.ManagedClusterInfo {
	endpoint := ""
	if len(cluster.Spec.ManagedClusterClientConfigs) != 0 {
		endpoint = cluster.Spec.ManagedClusterClientConfigs[0].URL
	}

	return &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Name,
			Labels:    cluster.Labels,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			MasterEndpoint: endpoint,
			LoggingCA:      r.caData,
		},
	}
}
