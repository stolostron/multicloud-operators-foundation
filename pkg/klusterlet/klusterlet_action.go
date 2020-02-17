// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"encoding/json"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

func (k *Klusterlet) handleActionWork(work *v1beta1.Work) error {
	var err error
	var res runtime.RawExtension
	switch work.Spec.ActionType {
	case v1beta1.CreateActionType:
		res, err = k.handleCreateKubeActionWork(work)
	case v1beta1.DeleteActionType:
		res, err = k.handleDeleteKubeActionWork(work)
	case v1beta1.UpdateActionType:
		res, err = k.handleUpdateKubeActionWork(work)
	}
	if err != nil {
		return k.updateFailedStatus(work, err)
	}

	work.Status.Type = v1beta1.WorkCompleted
	work.Status.Result = res
	work.Status.LastUpdateTime = metav1.Now()
	_, err = k.hcmclientset.McmV1beta1().Works(k.config.ClusterNamespace).UpdateStatus(work)
	if err != nil {
		return err
	}

	return nil
}

//Create kube resource
func (k *Klusterlet) handleCreateKubeActionWork(work *v1beta1.Work) (runtime.RawExtension, error) {
	var err error
	if k.config.EnableImpersonation {
		klog.Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(work.Annotations["mcm.ibm.com/user-identity"])
		var userGroups = restutils.ParseUserGroup(work.Annotations["mcm.ibm.com/user-group"])
		_, err = k.kubeControl.Impersonate(userID, userGroups).Create(work.Spec.KubeWork.Namespace, work.Spec.KubeWork.ObjectTemplate, nil)
		k.kubeControl.UnsetImpersonate()
	} else {
		_, err = k.kubeControl.Create(work.Spec.KubeWork.Namespace, work.Spec.KubeWork.ObjectTemplate, nil)
	}
	return work.Spec.KubeWork.ObjectTemplate, err
}

//Update kube resource
func (k *Klusterlet) handleUpdateKubeActionWork(work *v1beta1.Work) (runtime.RawExtension, error) {
	var gvk schema.GroupVersionKind
	var err error

	patchType := types.MergePatchType
	obj := &unstructured.Unstructured{}
	if work.Spec.KubeWork.ObjectTemplate.Object != nil {
		gvk = work.Spec.KubeWork.ObjectTemplate.Object.GetObjectKind().GroupVersionKind()
	} else {
		err = json.Unmarshal(work.Spec.KubeWork.ObjectTemplate.Raw, obj)
		if err != nil {
			return runtime.RawExtension{}, err
		}
		gvk = obj.GroupVersionKind()
	}

	namespace := work.Spec.KubeWork.Namespace
	if namespace == "" {
		namespace = obj.GetNamespace()
	}
	name := obj.GetName()

	currentObj, err := k.kubeControl.Get(&gvk, "", namespace, name, false)
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

	patch, err := restutils.GeneratePatch(currentObj, work.Spec.KubeWork.ObjectTemplate, originRaw)
	if err != nil {
		return runtime.RawExtension{}, err
	}

	if string(patch) == "{}" {
		klog.V(5).Infof("Nothing to update")
		return work.Status.Result, nil
	}

	klog.V(5).Infof("%v in ns %v updates patch %v", name, namespace, string(patch))
	if k.config.EnableImpersonation {
		klog.Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(work.Annotations["mcm.ibm.com/user-identity"])
		var userGroups = restutils.ParseUserGroup(work.Annotations["mcm.ibm.com/user-group"])
		_, err = k.kubeControl.Impersonate(userID, userGroups).Patch(namespace, name, gvk, patchType, patch)
		k.kubeControl.UnsetImpersonate()
	} else {
		_, err = k.kubeControl.Patch(namespace, name, gvk, patchType, patch)
	}
	if err != nil {
		klog.V(5).Infof("Failed to patch object: %v", err)
		return runtime.RawExtension{}, err
	}

	return work.Spec.KubeWork.ObjectTemplate, err
}

//Delete kube resource
func (k *Klusterlet) handleDeleteKubeActionWork(work *v1beta1.Work) (runtime.RawExtension, error) {
	var err error
	if k.config.EnableImpersonation {
		klog.Info("enable impersonation")
		var userID = restutils.ParseUserIdentity(work.Annotations["mcm.ibm.com/user-identity"])
		var userGroups = restutils.ParseUserGroup(work.Annotations["mcm.ibm.com/user-group"])
		err = k.kubeControl.Impersonate(userID, userGroups).Delete(nil,
			work.Spec.KubeWork.Resource,
			work.Spec.KubeWork.Namespace,
			work.Spec.KubeWork.Name,
		)
		k.kubeControl.UnsetImpersonate()
	} else {
		err = k.kubeControl.Delete(
			nil,
			work.Spec.KubeWork.Resource,
			work.Spec.KubeWork.Namespace,
			work.Spec.KubeWork.Name,
		)
	}
	return runtime.RawExtension{}, err
}
