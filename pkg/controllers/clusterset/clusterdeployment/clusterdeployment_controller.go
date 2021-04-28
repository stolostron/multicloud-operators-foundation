package clusterdeployment

import (
	"context"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	clustersetutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/clusterset"

	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
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

// This controller sync the clusterdeployment's utils.ClusterSetLabel with releated clusterpool's utils.ClusterSetLabel
// if the clusterpool did not exist, do nothing.
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create ClusterDeployment controller, %v", err)
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
	c, err := controller.New("clusterset-clusterdeployment-mapper-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &hivev1.ClusterDeployment{}},
		&handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	clusterdeployment := &hivev1.ClusterDeployment{}
	klog.V(5).Infof("reconcile: %+v", req)
	err := r.client.Get(ctx, req.NamespacedName, clusterdeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//if the clusterdeployment is not created by clusterpool, do nothing
	if clusterdeployment.Spec.ClusterPoolRef == nil {
		return ctrl.Result{}, nil
	}

	clusterpool := &hivev1.ClusterPool{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: clusterdeployment.Spec.ClusterPoolRef.Namespace, Name: clusterdeployment.Spec.ClusterPoolRef.PoolName}, clusterpool)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	klog.V(5).Infof("Clusterdeployment's clusterpool: %+v", clusterpool)

	var isModified = false
	utils.SyncMapFiled(&isModified, &clusterdeployment.Labels, clusterpool.Labels, clustersetutils.ClusterSetLabel)

	if isModified {
		err = r.client.Update(ctx, clusterdeployment, &client.UpdateOptions{})
		if err != nil {
			klog.Errorf("Can not update clusterdeployment label. clusterdeployment: %v, error:%v", clusterdeployment.Name, err)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
