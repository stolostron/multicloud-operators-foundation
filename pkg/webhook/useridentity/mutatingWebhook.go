package useridentity

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/mattbaird/jsonpatch"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app/options"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/klog/v2"
)

type AdmissionHandler struct {
	Lister rbaclisters.RoleBindingLister
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func toAdmissionResponse(err error) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

// admitFunc is the type we use for all of our validators and mutators
type admitFunc func(v1.AdmissionReview) *v1.AdmissionResponse

// AppMgr service account, we do not overwrite the user, as the subscription will handle this.
const UserIDforAppMgr = "system:serviceaccount:open-cluster-management-agent-addon:klusterlet-addon-appmgr"

// serve handles the http portion of a request prior to handing to an admit
// function
func (a *AdmissionHandler) serve(w io.Writer, r *http.Request, admit admitFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	klog.V(2).Info(fmt.Sprintf("handling request: %s", body))

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := v1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := v1.AdmissionReview{}

	deserializer := options.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Error(err)
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview)
	}

	responseAdmissionReview.Kind = requestedAdmissionReview.Kind
	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	klog.V(0).Info(fmt.Sprintf("sending response: %+v", responseAdmissionReview))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}

func (a *AdmissionHandler) mutateResource(ar v1.AdmissionReview) *v1.AdmissionResponse {
	klog.V(0).Info("mutating custom resource")
	obj := unstructured.Unstructured{}
	err := obj.UnmarshalJSON(ar.Request.Object.Raw)
	if err != nil {
		klog.Error(err)
		return toAdmissionResponse(err)
	}

	annotations := obj.GetAnnotations()

	// Do not change the userInfo if the object is being created by the AppMgr.
	// This value is stamped onto the Subscription object by Appmgr (Only for subscriptions)
	klog.V(0).Infof("Object kind:%s, User request:%s, expected:%s", obj.GetKind(), ar.Request.UserInfo.Username, UserIDforAppMgr)
	if obj.GetKind() == "Subscription" && ar.Request.UserInfo.Username == UserIDforAppMgr {
		klog.V(0).Infof("Skip add user and group for resource: %+v, name: %+v", ar.Request.Resource.Resource, obj.GetName())
		reviewResponse := v1.AdmissionResponse{}
		reviewResponse.Allowed = true
		return &reviewResponse
	}

	resAnnotations := MergeUserIdentityToAnnotations(ar.Request.UserInfo, annotations, obj.GetNamespace(), a.Lister)
	obj.SetAnnotations(resAnnotations)

	reviewResponse := v1.AdmissionResponse{}
	reviewResponse.Allowed = true

	updatedObj, err := obj.MarshalJSON()
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	res, err := jsonpatch.CreatePatch(ar.Request.Object.Raw, updatedObj)
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

	klog.V(2).Infof("Successfully Added user and group for resource: %+v, name: %+v", ar.Request.Resource.Resource, obj.GetName())
	return &reviewResponse
}

func (a *AdmissionHandler) ServeMutateResource(w http.ResponseWriter, r *http.Request) {
	a.serve(w, r, a.mutateResource)
}
