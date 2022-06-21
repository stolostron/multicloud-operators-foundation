package controllers

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	restutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/rest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ActionReconciler) handleAction(action *actionv1beta1.ManagedClusterAction) error {
	var err error
	var res runtime.RawExtension
	var reason string
	switch action.Spec.ActionType {
	case actionv1beta1.CreateActionType:
		res, err = r.handleCreateAction(action)
		reason = actionv1beta1.ReasonCreateResourceFailed
	case actionv1beta1.DeleteActionType:
		res, err = r.handleDeleteAction(action)
		reason = actionv1beta1.ReasonDeleteResourceFailed
	case actionv1beta1.UpdateActionType:
		res, err = r.handleUpdateAction(action)
		reason = actionv1beta1.ReasonUpdateResourceFailed
	default:
		err = fmt.Errorf("invalid action type")
		reason = actionv1beta1.ReasonActionTypeInvalid
	}

	if err != nil {
		meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
			Type:    actionv1beta1.ConditionActionCompleted,
			Status:  metav1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Errorf("failed to handle %v action with err: %v", action.Spec.ActionType, err).Error(),
		})

		return err
	}

	meta.SetStatusCondition(&action.Status.Conditions, metav1.Condition{
		Type:    actionv1beta1.ConditionActionCompleted,
		Status:  metav1.ConditionTrue,
		Reason:  "ActionDone",
		Message: "Resource action is done.",
	})

	action.Status.Result = res
	return nil
}

// Create kube resource
func (r *ActionReconciler) handleCreateAction(action *actionv1beta1.ManagedClusterAction) (runtime.RawExtension, error) {
	var err error
	if r.EnableImpersonation {
		r.Log.V(5).Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(action.Annotations[actionv1beta1.UserIdentityAnnotation])
		var userGroups = restutils.ParseUserGroup(action.Annotations[actionv1beta1.UserGroupAnnotation])
		_, err = r.KubeControl.Impersonate(userID, userGroups).Create(action.Spec.KubeWork.Namespace, action.Spec.KubeWork.ObjectTemplate, nil)
		r.KubeControl.UnsetImpersonate()
	} else {
		_, err = r.KubeControl.Create(action.Spec.KubeWork.Namespace, action.Spec.KubeWork.ObjectTemplate, nil)
	}
	return action.Spec.KubeWork.ObjectTemplate, err
}

// Update kube resource
func (r *ActionReconciler) handleUpdateAction(action *actionv1beta1.ManagedClusterAction) (runtime.RawExtension, error) {
	var gvk schema.GroupVersionKind
	var err error

	patchType := types.MergePatchType
	obj := &unstructured.Unstructured{}
	if action.Spec.KubeWork.ObjectTemplate.Object != nil {
		gvk = action.Spec.KubeWork.ObjectTemplate.Object.GetObjectKind().GroupVersionKind()
	} else {
		err = json.Unmarshal(action.Spec.KubeWork.ObjectTemplate.Raw, obj)
		if err != nil {
			return runtime.RawExtension{}, err
		}
		gvk = obj.GroupVersionKind()
	}

	namespace := action.Spec.KubeWork.Namespace
	if namespace == "" {
		namespace = obj.GetNamespace()
	}
	name := obj.GetName()

	currentObj, err := r.KubeControl.Get(&gvk, "", namespace, name, false)
	if err != nil {
		return runtime.RawExtension{}, err
	}

	currentRaw, err := json.Marshal(currentObj)
	if err != nil {
		return runtime.RawExtension{}, err
	}

	originRaw := runtime.RawExtension{
		Raw: currentRaw,
	}

	patch, err := restutils.GeneratePatch(currentObj, action.Spec.KubeWork.ObjectTemplate, originRaw)
	if err != nil {
		return runtime.RawExtension{}, err
	}

	if string(patch) == "{}" {
		r.Log.V(5).Info("Nothing to update")
		return action.Status.Result, nil
	}

	r.Log.V(5).Info("resource update", "name", name, "namespace", namespace, "updates patch", string(patch))
	if r.EnableImpersonation {
		r.Log.V(5).Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(action.Annotations[actionv1beta1.UserIdentityAnnotation])
		var userGroups = restutils.ParseUserGroup(action.Annotations[actionv1beta1.UserGroupAnnotation])
		_, err = r.KubeControl.Impersonate(userID, userGroups).Patch(namespace, name, gvk, patchType, patch)
		r.KubeControl.UnsetImpersonate()
	} else {
		_, err = r.KubeControl.Patch(namespace, name, gvk, patchType, patch)
	}
	if err != nil {
		r.Log.Error(err, "Failed to patch object")
		return runtime.RawExtension{}, err
	}

	return action.Spec.KubeWork.ObjectTemplate, err
}

// Delete kube resource
func (r *ActionReconciler) handleDeleteAction(action *actionv1beta1.ManagedClusterAction) (runtime.RawExtension, error) {
	var err error
	if r.EnableImpersonation {
		r.Log.V(5).Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(action.Annotations[actionv1beta1.UserIdentityAnnotation])
		var userGroups = restutils.ParseUserGroup(action.Annotations[actionv1beta1.UserGroupAnnotation])
		err = r.KubeControl.Impersonate(userID, userGroups).Delete(nil,
			action.Spec.KubeWork.Resource,
			action.Spec.KubeWork.Namespace,
			action.Spec.KubeWork.Name,
		)
		r.KubeControl.UnsetImpersonate()
	} else {
		err = r.KubeControl.Delete(
			nil,
			action.Spec.KubeWork.Resource,
			action.Spec.KubeWork.Namespace,
			action.Spec.KubeWork.Name,
		)
	}
	return runtime.RawExtension{}, err
}
