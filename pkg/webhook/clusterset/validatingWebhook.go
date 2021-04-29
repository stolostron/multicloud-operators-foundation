package clusterset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app/options"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	v1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type AdmissionHandler struct {
	KubeClient kubernetes.Interface
	Enable     bool
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
type admitFunc func(request *v1.AdmissionRequest) *v1.AdmissionResponse

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

var managedClustersGVR = metav1.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
}
var clusterDeploymentsGVR = metav1.GroupVersionResource{
	Group:    "hive.openshift.io",
	Version:  "v1",
	Resource: "clusterdeployments",
}
var clusterPoolsGVR = metav1.GroupVersionResource{
	Group:    "hive.openshift.io",
	Version:  "v1",
	Resource: "clusterpools",
}

const (
	clusterSetLabel = "cluster.open-cluster-management.io/clusterset"
)

// validateResource validate:
// 1. allow requests if the user has managedClusterSet/join all resources permission.
// 2. allow clusterDeployment requests created by clusterClaim.
// 3. if user has permission to create/update resources.
func (a *AdmissionHandler) validateResource(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	status := &v1.AdmissionResponse{
		Allowed: true,
	}
	//TODO need improve later, may be in installer part to enable/disable it.
	if !a.Enable {
		return status
	}
	switch request.Resource {
	case managedClustersGVR, clusterPoolsGVR:
		break
	case clusterDeploymentsGVR:
		// ignore clusterDeployment created by clusterClaim
		clusterDeployment := &hivev1.ClusterDeployment{}
		if err := json.Unmarshal(request.Object.Raw, clusterDeployment); err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
		}

		if clusterDeployment.Spec.ClusterPoolRef != nil {
			return status
		}
	default:
		return status
	}

	switch request.Operation {
	case v1.Create:
		return a.validateCreateRequest(request)
	case v1.Update:
		return a.validateUpdateRequest(request)
	}

	return status
}

// validateCreateRequest validates:
// 1. the user has managedClusterSet/join <managedClusterSet> permission to create resource with a <managedClusterSet> label.
// 2. the user has managedClusterSet/join <all> permission to create resource without a managedClusterSet label.
func (a *AdmissionHandler) validateCreateRequest(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	status := &v1.AdmissionResponse{
		Allowed: true,
	}

	obj := unstructured.Unstructured{}
	err := obj.UnmarshalJSON(request.Object.Raw)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	labels := obj.GetLabels()
	clusterSetName := ""
	if len(labels) > 0 {
		clusterSetName = labels[clusterSetLabel]
	}

	return a.allowUpdateClusterSet(request.UserInfo, clusterSetName)
}

// validateUpdateRequest validates:
// 1. allow if the managedClusterSet label is not changed.
// 2. the user must have managedClusterSet/join <clusterSet-A/clusterSet-B> permission if update resource to change
// the managedClusterSet label from clusterSet-A to clusterSet-B.
// 3. the user must have managedClusterSet/join <all> permission if update resource to add/remove the managedClusterSet label.
func (a *AdmissionHandler) validateUpdateRequest(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	status := &v1.AdmissionResponse{
		Allowed: true,
	}

	oldObj := unstructured.Unstructured{}
	err := oldObj.UnmarshalJSON(request.OldObject.Raw)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	newObj := unstructured.Unstructured{}
	err = newObj.UnmarshalJSON(request.Object.Raw)
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: err.Error(),
		}
		return status
	}

	oldLabels := oldObj.GetLabels()
	newLabels := newObj.GetLabels()
	originalClusterSetName := ""
	currentClusterSetName := ""
	if len(oldLabels) > 0 {
		originalClusterSetName = oldLabels[clusterSetLabel]
	}
	if len(newLabels) > 0 {
		currentClusterSetName = newLabels[clusterSetLabel]
	}

	// allow if managedClusterSet label is not updated
	if originalClusterSetName == currentClusterSetName {
		return status
	}

	if status := a.allowUpdateClusterSet(request.UserInfo, originalClusterSetName); !status.Allowed {
		return status
	}

	return a.allowUpdateClusterSet(request.UserInfo, currentClusterSetName)
}

// allowUpdateClusterSet checks whether a request user has been authorized to add/remove a resource to/from the ManagedClusterSet.
// check if the user has clusterSet/join <all> permission when clusterSetName is null.
func (a *AdmissionHandler) allowUpdateClusterSet(userInfo authenticationv1.UserInfo, clusterSetName string) *v1.AdmissionResponse {
	status := &v1.AdmissionResponse{}

	extra := make(map[string]authorizationv1.ExtraValue)
	for k, v := range userInfo.Extra {
		extra[k] = authorizationv1.ExtraValue(v)
	}

	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			UID:    userInfo.UID,
			Groups: userInfo.Groups,
			Extra:  extra,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group:       "cluster.open-cluster-management.io",
				Resource:    "managedclustersets",
				Subresource: "join",
				Name:        clusterSetName,
				Verb:        "create",
			},
		},
	}
	sar, err := a.KubeClient.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: err.Error(),
		}
		return status
	}

	if !sar.Status.Allowed {
		status.Allowed = false
		status.Result = &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
			Message: fmt.Sprintf("user %q cannot add/remove the resource to/from ManagedClusterSet %q", userInfo.Username, clusterSetName),
		}
		return status
	}

	status.Allowed = true
	return status
}

func (a *AdmissionHandler) ServerValidateResource(w http.ResponseWriter, r *http.Request) {
	a.serve(w, r, a.validateResource)
}
