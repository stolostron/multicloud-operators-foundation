package useridentity

import (
	"encoding/json"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/mattbaird/jsonpatch"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	serve "github.com/open-cluster-management/multicloud-operators-foundation/pkg/webhook/serve"
	v1 "k8s.io/api/admission/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/klog/v2"
)

type AdmissionHandler struct {
	Lister                rbaclisters.RoleBindingLister
	SkipOverwriteUserList []string
}

func (a *AdmissionHandler) mutateResource(ar *v1.AdmissionRequest) *v1.AdmissionResponse {
	klog.V(4).Info("mutating custom resource")

	obj := unstructured.Unstructured{}
	err := obj.UnmarshalJSON(ar.Object.Raw)
	if err != nil {
		klog.Error(err)
		return serve.ToAdmissionResponse(err)
	}

	annotations := obj.GetAnnotations()

	if utils.ContainsString(a.SkipOverwriteUserList, ar.UserInfo.Username) {
		klog.V(4).Infof("Skip add user and group for resource: %+v, name: %+v", ar.Resource.Resource, obj.GetName())
		reviewResponse := v1.AdmissionResponse{}
		reviewResponse.Allowed = true
		return &reviewResponse
	}

	resAnnotations := MergeUserIdentityToAnnotations(ar.UserInfo, annotations, obj.GetNamespace(), a.Lister)
	obj.SetAnnotations(resAnnotations)

	reviewResponse := v1.AdmissionResponse{}
	reviewResponse.Allowed = true

	updatedObj, err := obj.MarshalJSON()
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	res, err := jsonpatch.CreatePatch(ar.Object.Raw, updatedObj)
	if err != nil {
		klog.Errorf("Create patch error: %+v", err)
		return nil
	}
	klog.V(2).Infof("obj patch : %+v \n", res)

	resBytes, err := json.Marshal(res)
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	reviewResponse.Patch = resBytes
	pt := v1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt

	klog.V(2).Infof("Successfully Added user and group for resource: %+v, name: %+v", ar.Resource.Resource, obj.GetName())
	return &reviewResponse
}

func (a *AdmissionHandler) ServeMutateResource(w http.ResponseWriter, r *http.Request) {
	serve.Serve(w, r, a.mutateResource)
}
