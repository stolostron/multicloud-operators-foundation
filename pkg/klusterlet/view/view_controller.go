package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	viewv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	restutil "github.com/stolostron/multicloud-operators-foundation/pkg/utils/rest"
	"k8s.io/apimachinery/pkg/api/meta"
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
	Mapper                      meta.RESTMapper
}

const (
	DefaultUpdateInterval = 30 * time.Second
)

func (r *ViewReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	if condition := meta.FindStatusCondition(managedClusterView.Status.Conditions, viewv1beta1.ConditionViewProcessing); condition != nil {
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
		meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  metav1.ConditionFalse,
			Reason:  viewv1beta1.ReasonResourceNameInvalid,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	if scope.Resource == "" && (scope.Kind == "" || scope.Version == "") {
		err = fmt.Errorf("invalid resource type")
		meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  metav1.ConditionFalse,
			Reason:  viewv1beta1.ReasonResourceTypeInvalid,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	if scope.Resource == "" {
		gvk := schema.GroupVersionKind{Group: scope.Group, Kind: scope.Kind, Version: scope.Version}
		mapper, err := r.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
				Type:    viewv1beta1.ConditionViewProcessing,
				Status:  metav1.ConditionFalse,
				Reason:  viewv1beta1.ReasonResourceGVKInvalid,
				Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
			})
			return err
		}
		gvr = mapper.Resource
	} else {
		mapping, err := restutil.MappingFor(r.Mapper, scope.Resource)
		if err != nil {
			meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
				Type:    viewv1beta1.ConditionViewProcessing,
				Status:  metav1.ConditionFalse,
				Reason:  viewv1beta1.ReasonResourceTypeInvalid,
				Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
			})
			return err
		}
		gvr = mapping.Resource
	}

	// Do not return content of secrets
	if gvr.Resource == "secrets" {
		meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  metav1.ConditionFalse,
			Reason:  viewv1beta1.ReasonResourceTypeInvalid,
			Message: fmt.Errorf("viewing secrets is not allowed").Error(),
		})
		return nil
	}

	obj, err = r.ManagedClusterDynamicClient.Resource(gvr).Namespace(scope.Namespace).Get(context.TODO(), scope.Name, metav1.GetOptions{})
	if err != nil {
		meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
			Type:    viewv1beta1.ConditionViewProcessing,
			Status:  metav1.ConditionFalse,
			Reason:  viewv1beta1.ReasonGetResourceFailed,
			Message: fmt.Errorf("failed to get resource with err: %v", err).Error(),
		})
		return err
	}

	meta.SetStatusCondition(&managedClusterView.Status.Conditions, metav1.Condition{
		Type:    viewv1beta1.ConditionViewProcessing,
		Reason:  viewv1beta1.ReasonGetResource,
		Status:  metav1.ConditionTrue,
		Message: "Watching resources successfully",
	})

	objRaw, _ := json.Marshal(obj)
	if !bytes.Equal(managedClusterView.Status.Result.Raw, objRaw) {
		managedClusterView.Status.Result = runtime.RawExtension{Raw: objRaw, Object: obj}
	}

	return nil
}
