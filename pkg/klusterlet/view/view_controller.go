package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ViewReconciler reconciles a ManagedClusterView object
type ViewReconciler struct {
	client.Client
	Log                         logr.Logger
	Scheme                      *runtime.Scheme
	ManagedClusterDynamicClient dynamic.Interface
	Mapper                      *restutils.Mapper
}

const (
	DefaultUpdateInterval = 30 * time.Second
)

func (r *ViewReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("ManagedClusterView", req.NamespacedName)
	updateInterval := DefaultUpdateInterval
	managedClusterView := &viewv1beta1.ManagedClusterView{}

	err := r.Get(ctx, req.NamespacedName, managedClusterView)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if managedClusterView.Spec.Scope.UpdateIntervalSeconds != 0 {
		updateInterval = time.Duration(managedClusterView.Spec.Scope.UpdateIntervalSeconds) * time.Second
	}

	if condition := helpers.FindStatusCondition(managedClusterView.Status.Conditions, viewv1beta1.ConditionViewProcessing); condition != nil {
		sub := time.Since(condition.LastTransitionTime.Time)
		if sub < updateInterval {
			return ctrl.Result{RequeueAfter: updateInterval - sub}, nil
		}
	}

	if err := r.queryResource(managedClusterView); err != nil {
		log.Error(err, "failed to query resource")
	}

	if err := r.Client.Status().Update(ctx, managedClusterView); err != nil {
		log.Error(err, "unable to update status of ManagedClusterView")
		return ctrl.Result{RequeueAfter: updateInterval}, err
	}

	return ctrl.Result{RequeueAfter: updateInterval}, nil
}

func (r *ViewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&viewv1beta1.ManagedClusterView{}).
		Complete(r)
}

func (r *ViewReconciler) queryResource(managedClusterView *viewv1beta1.ManagedClusterView) error {
	var obj runtime.Object
	var err error
	var gvr schema.GroupVersionResource
	scope := managedClusterView.Spec.Scope

	if scope.Name == "" {
		err = fmt.Errorf("invalid resource name")
		helpers.SetStatusCondition(&managedClusterView.Status.Conditions, conditions.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  corev1.ConditionFalse,
			Reason:  viewv1beta1.ReasonResourceNameInvalid,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	if scope.Resource == "" && (scope.Kind == "" || scope.Version == "") {
		err = fmt.Errorf("invalid resource type")
		helpers.SetStatusCondition(&managedClusterView.Status.Conditions, conditions.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  corev1.ConditionFalse,
			Reason:  viewv1beta1.ReasonResourceTypeInvalid,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	if scope.Resource == "" {
		gvk := schema.GroupVersionKind{Group: scope.Group, Kind: scope.Kind, Version: scope.Version}
		mapper, err := r.Mapper.MappingForGVK(gvk)
		if err != nil {
			helpers.SetStatusCondition(&managedClusterView.Status.Conditions, conditions.Condition{
				Type:    viewv1beta1.ConditionViewProcessing,
				Status:  corev1.ConditionFalse,
				Reason:  viewv1beta1.ReasonResourceGVKInvalid,
				Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
			})
			return err
		}
		gvr = mapper.Resource
	} else {
		mapping, err := r.Mapper.MappingFor(scope.Resource)
		if err != nil {
			helpers.SetStatusCondition(&managedClusterView.Status.Conditions, conditions.Condition{
				Type:    viewv1beta1.ConditionViewProcessing,
				Status:  corev1.ConditionFalse,
				Reason:  viewv1beta1.ReasonResourceTypeInvalid,
				Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
			})
			return err
		}
		gvr = mapping.Resource
	}

	obj, err = r.ManagedClusterDynamicClient.Resource(gvr).Namespace(scope.Namespace).Get(context.TODO(), scope.Name, metav1.GetOptions{})
	if err != nil {
		helpers.SetStatusCondition(&managedClusterView.Status.Conditions, conditions.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  corev1.ConditionFalse,
			Reason:  viewv1beta1.ReasonGetResourceFailed,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	helpers.SetStatusCondition(&managedClusterView.Status.Conditions, conditions.Condition{
		Type:   viewv1beta1.ConditionViewProcessing,
		Status: corev1.ConditionTrue,
	})

	objRaw, _ := json.Marshal(obj)
	if !bytes.Equal(managedClusterView.Status.Result.Raw, objRaw) {
		managedClusterView.Status.Result = runtime.RawExtension{Raw: objRaw, Object: obj}
	}

	return nil
}
