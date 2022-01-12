package autodetect

import (
	"context"
	"reflect"

	clusterinfov1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LabelCloudVendor = "cloud"
	LabelKubeVendor  = "vendor"
	LabelClusterID   = "clusterID"
	LabelManagedBy   = "managed-by"
	AutoDetect       = "auto-detect"
)

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create auto-detect controller, %v", err)
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
	c, err := controller.New("autodetect-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1.ManagedCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterinfov1beta1.ManagedClusterInfo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cluster := &clusterv1.ManagedCluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	labels := cluster.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	needUpdate := false
	if labels[LabelCloudVendor] == AutoDetect && clusterInfo.Status.CloudVendor != "" {
		labels[LabelCloudVendor] = string(clusterInfo.Status.CloudVendor)
		needUpdate = true
	}

	if labels[LabelKubeVendor] == AutoDetect && clusterInfo.Status.KubeVendor != "" {
		labels[LabelKubeVendor] = string(clusterInfo.Status.KubeVendor)
		// Backward Compatible for placementrrule
		if clusterInfo.Status.KubeVendor == clusterinfov1beta1.KubeVendorOSD {
			labels[LabelKubeVendor] = string(clusterinfov1beta1.KubeVendorOpenShift)
			labels[LabelManagedBy] = "platform"
		}
		needUpdate = true
	}

	if clusterInfo.Status.ClusterID != "" && labels[LabelClusterID] != clusterInfo.Status.ClusterID {
		labels[LabelClusterID] = clusterInfo.Status.ClusterID
		needUpdate = true
	}

	if needUpdate {
		cluster.SetLabels(labels)
		if err := r.client.Update(ctx, cluster); err != nil {
			klog.Warningf("will reconcile since failed to add labels to ManagedCluster %v, %v", cluster.Name, err)
			return reconcile.Result{}, err
		}
	}

	if len(labels) == 0 && len(clusterInfo.ObjectMeta.Labels) == 0 {
		return ctrl.Result{}, nil
	}

	if !reflect.DeepEqual(labels, clusterInfo.ObjectMeta.Labels) {
		clusterInfo.SetLabels(labels)
		if err := r.client.Update(ctx, clusterInfo); err != nil {
			klog.Warningf("will reconcile since failed to add labels to ManagedClusterInfo %v, %v", clusterInfo.Name, err)
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
