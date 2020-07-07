package autodetect

import (
	"context"
	"reflect"

	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
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

	if clusterInfo.Status.CloudVendor == "" || clusterInfo.Status.KubeVendor == "" {
		klog.Infof("will reconcile since ManagedClusterInfo is not updated %v", clusterInfo.Name)
		return ctrl.Result{}, nil
	}

	labels := cluster.ObjectMeta.Labels
	if len(labels) == 0 {
		return ctrl.Result{}, nil
	}

	needUpdate := false
	if labels[LabelCloudVendor] == AutoDetect {
		labels[LabelCloudVendor] = string(clusterInfo.Status.CloudVendor)
		needUpdate = true
	}
	if labels[LabelKubeVendor] == AutoDetect {
		labels[LabelKubeVendor] = string(clusterInfo.Status.KubeVendor)
		needUpdate = true
	}
	if needUpdate {
		if err := r.client.Update(ctx, cluster); err != nil {
			klog.Warningf("will reconcile since failed to add labels to ManagedCluster %v, %v", cluster.Name, err)
			return reconcile.Result{}, err
		}
	}

	if !reflect.DeepEqual(labels, clusterInfo.ObjectMeta.Labels) {
		clusterInfo.ObjectMeta.Labels = labels
		if err := r.client.Update(ctx, clusterInfo); err != nil {
			klog.Warningf("will reconcile since failed to add labels to ManagedClusterInfo %v, %v", clusterInfo.Name, err)
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}
