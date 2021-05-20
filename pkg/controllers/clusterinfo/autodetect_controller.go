package clusterinfo

import (
	"context"
	"reflect"

	clusterinfov1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	clusterclaims "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
	OCPVersion       = "openshiftVersion"
)

// AutoDetectReconciler auto detects platform related labels and sync to managedcluster
type AutoDetectReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

// newAutoDetectReconciler returns a new reconcile.Reconciler
func newAutoDetectReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &AutoDetectReconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

func (r *AutoDetectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &clusterv1.ManagedCluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, cluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		return reconcile.Result{}, nil
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = r.client.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Name}, clusterInfo)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, nil
	case err != nil:
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

	for _, claim := range cluster.Status.ClusterClaims {
		if claim.Name == clusterclaims.ClaimOpenshiftVersion && labels[OCPVersion] != claim.Value {
			labels[OCPVersion] = claim.Value
			needUpdate = true
		}
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
