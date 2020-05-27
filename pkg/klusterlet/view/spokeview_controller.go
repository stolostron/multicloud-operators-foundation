package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
)

// SpokeViewReconciler reconciles a SpokeView object
type SpokeViewReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	SpokeDynamicClient dynamic.Interface
	Mapper             *restutils.Mapper
}

const (
	DefaultUpdateInterval = 30 * time.Second
)

func (r *SpokeViewReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("spokeview", req.NamespacedName)
	updateInterval := DefaultUpdateInterval
	spokeView := &viewv1beta1.SpokeView{}

	err := r.Get(ctx, req.NamespacedName, spokeView)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if spokeView.Spec.Scope.UpdateIntervalSeconds != 0 {
		updateInterval = time.Duration(spokeView.Spec.Scope.UpdateIntervalSeconds) * time.Second
	}

	if condition := conditions.FindStatusCondition(spokeView.Status.Conditions, viewv1beta1.ConditionViewProcessing); condition != nil {
		sub := time.Since(condition.LastTransitionTime.Time)
		if sub < updateInterval {
			return ctrl.Result{RequeueAfter: updateInterval - sub}, nil
		}
	}

	if err := r.queryResource(spokeView); err != nil {
		log.Error(err, "failed to query resource")
	}

	if err := r.Client.Status().Update(ctx, spokeView); err != nil {
		log.Error(err, "unable to update status of SpokeView")
		return ctrl.Result{RequeueAfter: updateInterval}, err
	}

	return ctrl.Result{RequeueAfter: updateInterval}, nil
}

func (r *SpokeViewReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&viewv1beta1.SpokeView{}).
		Complete(r)
}

func (r *SpokeViewReconciler) queryResource(spokeview *viewv1beta1.SpokeView) error {
	var obj runtime.Object
	var err error
	var gvr schema.GroupVersionResource
	scope := spokeview.Spec.Scope

	if scope.Name == "" {
		err = fmt.Errorf("invalid resource name")
		conditions.SetStatusCondition(&spokeview.Status.Conditions, conditions.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  corev1.ConditionFalse,
			Reason:  viewv1beta1.ReasonResourceNameInvalid,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	if scope.Resource == "" && (scope.Kind == "" || scope.Group == "" || scope.Version == "") {
		err = fmt.Errorf("invalid resource type")
		conditions.SetStatusCondition(&spokeview.Status.Conditions, conditions.Condition{
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
			conditions.SetStatusCondition(&spokeview.Status.Conditions, conditions.Condition{
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
			conditions.SetStatusCondition(&spokeview.Status.Conditions, conditions.Condition{
				Type:    viewv1beta1.ConditionViewProcessing,
				Status:  corev1.ConditionFalse,
				Reason:  viewv1beta1.ReasonResourceTypeInvalid,
				Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
			})
			return err
		}
		gvr = mapping.Resource
	}

	obj, err = r.SpokeDynamicClient.Resource(gvr).Namespace(scope.Namespace).Get(scope.Name, metav1.GetOptions{})
	if err != nil {
		conditions.SetStatusCondition(&spokeview.Status.Conditions, conditions.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  corev1.ConditionFalse,
			Reason:  viewv1beta1.ReasonGetResourceFailed,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	conditions.SetStatusCondition(&spokeview.Status.Conditions, conditions.Condition{
		Type:   viewv1beta1.ConditionViewProcessing,
		Status: corev1.ConditionTrue,
	})

	objRaw, _ := json.Marshal(obj)
	if !bytes.Equal(spokeview.Status.Result.Raw, objRaw) {
		spokeview.Status.Result = runtime.RawExtension{Raw: objRaw, Object: obj}
	}

	return nil
}
