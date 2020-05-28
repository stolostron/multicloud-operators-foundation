// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package controllers

import (
	"encoding/json"
	"fmt"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"

	corev1 "k8s.io/api/core/v1"

	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ActionReconciler) handleClusterAction(clusterAction *actionv1beta1.ClusterAction) error {
	var err error
	var res runtime.RawExtension
	var reason string
	switch clusterAction.Spec.ActionType {
	case actionv1beta1.CreateActionType:
		res, err = r.handleCreateClusterAction(clusterAction)
		reason = actionv1beta1.ReasonCreateResourceFailed
	case actionv1beta1.DeleteActionType:
		res, err = r.handleDeleteClusterAction(clusterAction)
		reason = actionv1beta1.ReasonDeleteResourceFailed
	case actionv1beta1.UpdateActionType:
		res, err = r.handleUpdateClusterAction(clusterAction)
		reason = actionv1beta1.ReasonUpdateResourceFailed
	default:
		err = fmt.Errorf("invalid action type")
		reason = actionv1beta1.ReasonActionTypeInvalid
	}

	if err != nil {
		conditions.SetStatusCondition(&clusterAction.Status.Conditions, conditions.Condition{
			Type:    actionv1beta1.ConditionActionCompleted,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: fmt.Errorf("failed to handle %v action with err: %v", clusterAction.Spec.ActionType, err).Error(),
		})

		return err
	}

	conditions.SetStatusCondition(&clusterAction.Status.Conditions, conditions.Condition{
		Type:   actionv1beta1.ConditionActionCompleted,
		Status: corev1.ConditionTrue,
	})

	clusterAction.Status.Result = res
	return nil
}

// Create kube resource
func (r *ActionReconciler) handleCreateClusterAction(clusterAction *actionv1beta1.ClusterAction) (runtime.RawExtension, error) {
	var err error
	if r.EnableImpersonation {
		r.Log.V(5).Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(clusterAction.Annotations[actionv1beta1.UserIdentityAnnotation])
		var userGroups = restutils.ParseUserGroup(clusterAction.Annotations[actionv1beta1.UserGroupAnnotation])
		_, err = r.KubeControl.Impersonate(userID, userGroups).Create(clusterAction.Spec.KubeWork.Namespace, clusterAction.Spec.KubeWork.ObjectTemplate, nil)
		r.KubeControl.UnsetImpersonate()
	} else {
		_, err = r.KubeControl.Create(clusterAction.Spec.KubeWork.Namespace, clusterAction.Spec.KubeWork.ObjectTemplate, nil)
	}
	return clusterAction.Spec.KubeWork.ObjectTemplate, err
}

// Update kube resource
func (r *ActionReconciler) handleUpdateClusterAction(clusterAction *actionv1beta1.ClusterAction) (runtime.RawExtension, error) {
	var gvk schema.GroupVersionKind
	var err error

	patchType := types.MergePatchType
	obj := &unstructured.Unstructured{}
	if clusterAction.Spec.KubeWork.ObjectTemplate.Object != nil {
		gvk = clusterAction.Spec.KubeWork.ObjectTemplate.Object.GetObjectKind().GroupVersionKind()
	} else {
		err = json.Unmarshal(clusterAction.Spec.KubeWork.ObjectTemplate.Raw, obj)
		if err != nil {
			return runtime.RawExtension{}, err
		}
		gvk = obj.GroupVersionKind()
	}

	namespace := clusterAction.Spec.KubeWork.Namespace
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

	patch, err := restutils.GeneratePatch(currentObj, clusterAction.Spec.KubeWork.ObjectTemplate, originRaw)
	if err != nil {
		return runtime.RawExtension{}, err
	}

	if string(patch) == "{}" {
		r.Log.V(5).Info("Nothing to update")
		return clusterAction.Status.Result, nil
	}

	r.Log.V(5).Info("resource update", "name", name, "namespace", namespace, "updates patch", string(patch))
	if r.EnableImpersonation {
		r.Log.V(5).Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(clusterAction.Annotations[actionv1beta1.UserIdentityAnnotation])
		var userGroups = restutils.ParseUserGroup(clusterAction.Annotations[actionv1beta1.UserGroupAnnotation])
		_, err = r.KubeControl.Impersonate(userID, userGroups).Patch(namespace, name, gvk, patchType, patch)
		r.KubeControl.UnsetImpersonate()
	} else {
		_, err = r.KubeControl.Patch(namespace, name, gvk, patchType, patch)
	}
	if err != nil {
		r.Log.Error(err, "Failed to patch object")
		return runtime.RawExtension{}, err
	}

	return clusterAction.Spec.KubeWork.ObjectTemplate, err
}

// Delete kube resource
func (r *ActionReconciler) handleDeleteClusterAction(clusterAction *actionv1beta1.ClusterAction) (runtime.RawExtension, error) {
	var err error
	if r.EnableImpersonation {
		r.Log.V(5).Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(clusterAction.Annotations[actionv1beta1.UserIdentityAnnotation])
		var userGroups = restutils.ParseUserGroup(clusterAction.Annotations[actionv1beta1.UserGroupAnnotation])
		err = r.KubeControl.Impersonate(userID, userGroups).Delete(nil,
			clusterAction.Spec.KubeWork.Resource,
			clusterAction.Spec.KubeWork.Namespace,
			clusterAction.Spec.KubeWork.Name,
		)
		r.KubeControl.UnsetImpersonate()
	} else {
		err = r.KubeControl.Delete(
			nil,
			clusterAction.Spec.KubeWork.Resource,
			clusterAction.Spec.KubeWork.Namespace,
			clusterAction.Spec.KubeWork.Name,
		)
	}
	return runtime.RawExtension{}, err
}
