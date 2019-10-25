// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"encoding/json"
	"fmt"

	v1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	helmutil "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/helm"
	restutils "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/utils/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

func (k *Klusterlet) handleActionWork(work *v1alpha1.Work) error {
	var err error
	var res runtime.RawExtension
	if work.Spec.HelmWork != nil && k.helmControl == nil {
		return fmt.Errorf("Failed to find helm client")
	}
	switch work.Spec.ActionType {
	case v1alpha1.CreateActionType:
		if work.Spec.KubeWork != nil {
			res, err = k.handleCreateKubeActionWork(work)
		} else if work.Spec.HelmWork != nil {
			res, err = k.handleCreateHelmActionWork(work, k.helmControl)
		}
		break
	case v1alpha1.DeleteActionType:
		if work.Spec.KubeWork != nil {
			res, err = k.handleDeleteKubeActionWork(work)
		} else if work.Spec.HelmWork != nil {
			res, err = k.handleDeleteHelmActionWork(work, k.helmControl)
		}
		break
	case v1alpha1.UpdateActionType:
		if work.Spec.KubeWork != nil {
			res, err = k.handleUpdateKubeActionWork(work)
		} else if work.Spec.HelmWork != nil {
			res, err = k.handleUpdateHelmActionWork(work, k.helmControl)
		}
		break
	}
	if err != nil {
		return k.updateFailedStatus(work, err)
	}

	work.Status.Type = v1alpha1.WorkCompleted
	work.Status.Result = res
	work.Status.LastUpdateTime = metav1.Now()
	_, err = k.hcmclientset.McmV1alpha1().Works(k.config.ClusterNamespace).UpdateStatus(work)
	if err != nil {
		return err
	}

	return nil
}

//Create helm release
func (k *Klusterlet) handleCreateHelmActionWork(work *v1alpha1.Work, helmcontrol helmutil.HelmControlInterface) (runtime.RawExtension, error) {
	rls, err := helmcontrol.CreateHelmRelease(work.Spec.HelmWork.ReleaseName, work.Namespace, *work.Spec.HelmWork)
	res := runtime.RawExtension{}
	if err != nil {
		return res, err
	}
	rl := helmutil.ConvertHelmReleaseFromRelease(rls.GetRelease())
	return runtime.RawExtension{Object: &rl}, nil
}

//Create kube resource
func (k *Klusterlet) handleCreateKubeActionWork(work *v1alpha1.Work) (runtime.RawExtension, error) {
	_, err := k.kubeControl.Create(work.Spec.KubeWork.Namespace, work.Spec.KubeWork.ObjectTemplate, nil)
	return work.Spec.KubeWork.ObjectTemplate, err
}

//Update helm release
func (k *Klusterlet) handleUpdateHelmActionWork(work *v1alpha1.Work, helmcontrol helmutil.HelmControlInterface) (runtime.RawExtension, error) {
	res := runtime.RawExtension{}
	rls, err := helmcontrol.UpdateHelmRelease(work.Spec.HelmWork.ReleaseName, work.Namespace, *work.Spec.HelmWork)
	if err != nil {
		return res, err
	}
	rl := helmutil.ConvertHelmReleaseFromRelease(rls.GetRelease())
	return runtime.RawExtension{Object: &rl}, nil
}

//Update kube resource
func (k *Klusterlet) handleUpdateKubeActionWork(work *v1alpha1.Work) (runtime.RawExtension, error) {
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
	_, err = k.kubeControl.Patch(namespace, name, gvk, patchType, patch)
	if err != nil {
		klog.V(5).Infof("Failed to patch object: %v", err)
		return runtime.RawExtension{}, err
	}

	return work.Spec.KubeWork.ObjectTemplate, err
}

//Delete kube resource
func (k *Klusterlet) handleDeleteKubeActionWork(work *v1alpha1.Work) (runtime.RawExtension, error) {
	err := k.kubeControl.Delete(
		nil,
		work.Spec.KubeWork.Resource,
		work.Spec.KubeWork.Namespace,
		work.Spec.KubeWork.Name,
	)
	return runtime.RawExtension{}, err
}

//Delete helm release
func (k *Klusterlet) handleDeleteHelmActionWork(work *v1alpha1.Work, helmControl helmutil.HelmControlInterface) (runtime.RawExtension, error) {
	res := runtime.RawExtension{}
	rls, err := helmControl.DeleteHelmRelease(work.Spec.HelmWork.ReleaseName)
	if err != nil {
		return res, err
	}
	rl := helmutil.ConvertHelmReleaseFromRelease(rls.GetRelease())
	return runtime.RawExtension{Object: &rl}, nil
}
