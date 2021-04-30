package clusterclaim

import (
	"context"

	utils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	clustersetutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/clusterset"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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

// This controller sync the clusterclaim's utils.ClusterSetLabel with releated clusterpool's utils.ClusterSetLabel
// if the clusterpool did not exist, do nothing.
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create ClusterSetMapper controller, %v", err)
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
	c, err := controller.New("clusterset-clusterclaim-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &hivev1.ClusterClaim{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterclaim := &hivev1.ClusterClaim{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterclaim)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//update clusterclaim label by clusterpool's clusterset label
	clusterpool := &hivev1.ClusterPool{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: clusterclaim.Namespace, Name: clusterclaim.Spec.ClusterPoolName}, clusterpool)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.V(5).Infof("Clusterclaim's clusterpool: %+v", clusterpool)

	var isModified = false
	utils.SyncMapFiled(&isModified, &clusterclaim.Labels, clusterpool.Labels, clustersetutils.ClusterSetLabel)

	if isModified {
		err = r.client.Update(ctx, clusterclaim, &client.UpdateOptions{})
		if err != nil {
			klog.Errorf("Can not update clusterclaim label: %+v", clusterclaim.Name)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
