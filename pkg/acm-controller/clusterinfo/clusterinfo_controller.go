package clusterinfo

import (
	"context"
	"reflect"

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
	clusterFinalizerName = "clusterinfo.finalizers.open-cluster-management.io"
)

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
	caData []byte
}

func SetupWithManager(mgr manager.Manager, caData []byte) error {
	if err := add(mgr, newReconciler(mgr, caData)); err != nil {
		klog.Errorf("Failed to create clusterinfo controller, %v", err)
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

	// Watch for changes to primary resource cluster
	err = c.Watch(&source.Kind{Type: &clusterregistryv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.SpokeCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterinfov1beta1.ClusterInfo{}}, &handler.EnqueueRequestForObject{})
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
		if !containsString(cluster.GetFinalizers(), clusterFinalizerName) {
			klog.Info("Finalizer not found for cluster. Adding finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to add finalizer to cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(cluster.GetFinalizers(), clusterFinalizerName) {
			err := r.deleteExternalResources(cluster)
			if err != nil {
				return reconcile.Result{}, err
			}
			klog.Info("Removing Finalizer")
			cluster.ObjectMeta.Finalizers = removeString(cluster.ObjectMeta.Finalizers, clusterFinalizerName)
			if err := r.client.Update(context.TODO(), cluster); err != nil {
				klog.Errorf("Failed to remove finalizer from cluster, %v", err)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	clusterInfo := &clusterinfov1beta1.ClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, clusterInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(ctx, r.newClusterInfo(cluster)); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(r.caData, clusterInfo.Spec.KlusterletCA) {
		return ctrl.Result{}, nil
	}

	clusterInfo.Spec.KlusterletCA = r.caData
	err = r.client.Update(ctx, clusterInfo)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(cluster *clusterregistryv1alpha1.Cluster) error {
	err := r.client.Delete(context.Background(), &clusterinfov1beta1.ClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
	})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

func (r *Reconciler) newClusterInfo(cluster *clusterregistryv1alpha1.Cluster) *clusterinfov1beta1.ClusterInfo {
	return &clusterinfov1beta1.ClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			KlusterletCA: r.caData,
		},
	}
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
