package app

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mattbaird/jsonpatch"
	"github.com/stolostron/multicloud-operators-foundation/cmd/webhook/app/options"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/denynamespace"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/useridentity"
	"k8s.io/api/admission/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/klog"
)

var namespaceGVR = metav1.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

type admissionHandler struct {
	lister        rbaclisters.RoleBindingLister
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

// toAdmissionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func toAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

// admitFunc is the type we use for all of our validators and mutators
type admitFunc func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

// serve handles the http portion of a request prior to handing to an admit
// function
func (a *admissionHandler) serve(w io.Writer, r *http.Request, admit admitFunc) {
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
	requestedAdmissionReview := v1beta1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := v1beta1.AdmissionReview{}

	deserializer := options.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Error(err)
		responseAdmissionReview.Response = toAdmissionResponse(err)
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview)
	}

	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	klog.V(2).Info(fmt.Sprintf("sending response: %v", responseAdmissionReview.Response))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}

func (a *admissionHandler) mutateResource(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	klog.V(2).Info("mutating custom resource")
	raw := ar.Request.Object.Raw
	crd := apiextensionsv1beta1.CustomResourceDefinition{}
	deserializer := options.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &crd); err != nil {
		klog.Error(err)
		return toAdmissionResponse(err)
	}
	ori, err := json.Marshal(crd)
	if err != nil {
		klog.Error(err)
		return toAdmissionResponse(err)
	}
	annotations := crd.GetAnnotations()

	resAnnotations := useridentity.MergeUserIdentityToAnnotations(ar.Request.UserInfo, annotations, crd.GetNamespace(), a.lister)
	crd.SetAnnotations(resAnnotations)
	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	crBytes, err := json.Marshal(crd)
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	res, err := jsonpatch.CreatePatch(ori, crBytes)
	if err != nil {
		klog.Errorf("Create patch error: %+v", err)
		return nil
	}
	resBytes, err := json.Marshal(res)
	if err != nil {
		klog.Errorf("marshal json error: %+v", err)
		return nil
	}
	reviewResponse.Patch = resBytes
	pt := v1beta1.PatchTypeJSONPatch
	reviewResponse.PatchType = &pt
	klog.V(2).Infof("Successfully Added user and group for resource: %+v, name: %+v", ar.Request.Resource.Resource, crd.GetName())
	return &reviewResponse
}

func (a *admissionHandler) validateResource(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	reviewResponse := v1beta1.AdmissionResponse{
		Allowed: true,
	}

	switch {
	case ar.Request.Resource == namespaceGVR:
		klog.V(2).Info("validating namespace deletion")
		allowedDeny, msg := denynamespace.ShouldDenyDeleteNamespace(ar.Request.Namespace, a.dynamicClient)
		reviewResponse.Allowed = !allowedDeny
		if allowedDeny {
			reviewResponse.Result = &metav1.Status{Message: msg}
		}
		klog.V(2).Infof("reviewResponse %v", reviewResponse)
	}

	return &reviewResponse
}

func (a *admissionHandler) serveMutateResource(w http.ResponseWriter, r *http.Request) {
	a.serve(w, r, a.mutateResource)
}

func (a *admissionHandler) serverValidateResource(w http.ResponseWriter, r *http.Request) {
	a.serve(w, r, a.validateResource)
}
