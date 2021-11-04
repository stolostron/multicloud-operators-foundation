package serve

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app/options"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// toAdmissionResponse is a helper function to create an AdmissionResponse
// with an embedded error
func ToAdmissionResponse(err error) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
			Reason:  metav1.StatusReasonBadRequest,
		},
	}
}

// admitFunc is the type we use for all of our validators and mutators
type admitFunc func(request *v1.AdmissionRequest) *v1.AdmissionResponse

// serve handles the http portion of a request prior to handing to an admit
// function
func Serve(w io.Writer, r *http.Request, admit admitFunc) {
	var body []byte
	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := v1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := v1.AdmissionReview{}

	if r.Body == nil {
		klog.Errorf("Request Body is null, request: %v", r)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		klog.Errorf("Can not read request body, err: %v", err)
		return
	}

	body = data

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	klog.V(2).Info(fmt.Sprintf("handling request: %s", body))

	deserializer := options.Codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		klog.Error(err)
		return
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview.Request)
	}

	responseAdmissionReview.Kind = requestedAdmissionReview.Kind
	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	// Return the same UID
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	klog.V(2).Info(fmt.Sprintf("sending response: %+v", responseAdmissionReview))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}
