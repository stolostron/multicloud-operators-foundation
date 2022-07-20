package serve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/stolostron/multicloud-operators-foundation/cmd/webhook/app/options"
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
func Serve(w http.ResponseWriter, r *http.Request, admit admitFunc) {
	var body []byte
	var errmsg string
	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := v1.AdmissionReview{}

	// The AdmissionReview that will be returned
	responseAdmissionReview := v1.AdmissionReview{}

	if r.Body == nil {
		errmsg = "Request Body is null"
		writerErrorResponse(errmsg, w)
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errmsg = fmt.Sprintf("Can not read request body, err: %v", err)
		writerErrorResponse(errmsg, w)
		return
	}

	body = data

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		errmsg = fmt.Sprintf("contentType=%s, expect application/json", contentType)
		writerErrorResponse(errmsg, w)
		return
	}

	klog.V(2).Info(fmt.Sprintf("handling request: %s", body))

	deserializer := options.Codecs.UniversalDeserializer()
	_, _, err = deserializer.Decode(body, nil, &requestedAdmissionReview)
	if err != nil {
		errmsg = fmt.Sprintf("Decode body error: %v", err)
		writerErrorResponse(errmsg, w)
		return
	} else {
		// pass to admitFunc
		responseAdmissionReview.Response = admit(requestedAdmissionReview.Request)
	}

	responseAdmissionReview.Kind = requestedAdmissionReview.Kind
	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	// Return the same UID
	if requestedAdmissionReview.Request == nil {
		errmsg = fmt.Sprintf("requestedAdmissionReview is nil")
		writerErrorResponse(errmsg, w)
		return
	}
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID

	klog.V(2).Info(fmt.Sprintf("sending response: %+v", responseAdmissionReview))

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		errmsg = fmt.Sprintf("Decode responseAdmissionReview error: %v", err)
		writerErrorResponse(errmsg, w)
		return
	}
	_, err = w.Write(respBytes)
	if err != nil {
		errmsg = fmt.Sprintf("Write responsebyte error: %v", err)
		writerErrorResponse(errmsg, w)
	}
	return
}

func writerErrorResponse(errmsg string, httpWriter http.ResponseWriter) {
	var buffer bytes.Buffer
	buffer.WriteString(errmsg)
	httpWriter.WriteHeader(http.StatusBadRequest)
	if _, err := httpWriter.Write(buffer.Bytes()); err != nil {
		klog.Warningf("%v", err)
	}
}
