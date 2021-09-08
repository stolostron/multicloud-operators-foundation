package clusterset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app/options"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	v1 "k8s.io/api/admission/v1"

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

type AdmissionHandler struct {
	HiveClient hiveclient.Interface
	KubeClient kubernetes.Interface
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
var clusterPoolsGVR = metav1.GroupVersionResource{
	Group:    "hive.openshift.io",
	Version:  "v1",
	Resource: "clusterpools",
}
var clusterDeploymentsGVR = metav1.GroupVersionResource{
	Group:    "hive.openshift.io",
	Version:  "v1",
	Resource: "clusterdeployments",
}

const (
	clusterSetLabel = "cluster.open-cluster-management.io/clusterset"
)

// validateResource validate:
// 1. allow requests if the user has managedClusterSet/join all resources permission.
// 2. if user has permission to create/update resources.
func (a *AdmissionHandler) validateResource(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	status := &v1.AdmissionResponse{
		Allowed: true,
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

	if request.Resource == clusterDeploymentsGVR {
		// ignore clusterDeployment created by clusterpool
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
// 1. Allow requests if the clusterSet label is not changed.
// 2, If the user is try to update clusterset
//   2.1 For clusterpool, Deny the requests to update clusterSet label
//   2.2 For clusterdeployment, User "could" update the clusterset label if he/she has permission,
//       but clusterdeployment controller will always sync the clusterdeployment's clusterset label back.
//   2.3 For managedClusters, check if the managedcluster is claimed from clusterpool
//      2.3.1 if the managedcluster is claimed from clusterpool, do not allow to update clusterset.
//      2.3.2 if the managedcluster is not claimed from clusterpool, check if user has permission to update clusterset.
// Notes: clusterclaims-controller will auto create a mangedcluster with the clusterpool's clusterset label if there is a clusterclaim ref to this clusterpool.
// https://github.com/open-cluster-management/clusterclaims-controller/blob/main/controllers/clusterclaims/clusterclaims-controller.go

// Permission Check:
// 1. the user must have managedClusterSet/join <clusterSet-A/clusterSet-B> permission if update resource to change
//         the managedClusterSet label from clusterSet-A to clusterSet-B.
// 2. the user must have managedClusterSet/join <all> permission if update resource to add/remove the clusterSet label.
//

func (a *AdmissionHandler) validateUpdateRequest(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	allowStatus := &v1.AdmissionResponse{
		Allowed: true,
	}
	rejectStatus := &v1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: "",
		},
	}

	//check if there is a update clusterset label request
	isUpdateClusterSet, originalClusterSetName, currentClusterSetName, err := a.isUpdateClusterset(request)
	if err != nil {
		rejectStatus.Result.Message = err.Error()
		return rejectStatus
	}
	if !isUpdateClusterSet {
		return allowStatus
	}

	switch request.Resource {
	case managedClustersGVR:
		managedCluster := &clusterv1.ManagedCluster{}
		if err := json.Unmarshal(request.Object.Raw, managedCluster); err != nil {
			rejectStatus.Result.Message = err.Error()
			return rejectStatus
		}

		// Get managedcluster's clusterdeployment
		// 1. No related clusterdeployment, check if user has permission to update this cluster.
		// 2. The related clusterdeployment is created from clusterpool, do not allow user to update the managedcluster.
		// 3. The related clusterdeployment is not created from clusterpool, check if user has permission to update the managedcluster.
		clusterDeployment, err := a.HiveClient.HiveV1().ClusterDeployments(managedCluster.Name).Get(context.TODO(), managedCluster.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				rejectStatus.Result.Message = err.Error()
				return rejectStatus
			}
			// Do not find clusterdeployment in managedcluster namespace. allow to update clusterset label.
			return a.allowUpdate(request.UserInfo, originalClusterSetName, currentClusterSetName)
		}

		// the cluster is claimed from clusterpool, do not allow to update clustersetlabel.
		if clusterDeployment != nil && clusterDeployment.Spec.ClusterPoolRef != nil {
			rejectStatus.Result.Message = "Do not allow to update claimed cluster's clusterset."
			return rejectStatus
		}
		// the clusterdeployment is not claimed from clusterpool, allow to update clustersetlabel.
		return a.allowUpdate(request.UserInfo, originalClusterSetName, currentClusterSetName)
	case clusterPoolsGVR:
		// do not allow to update clusterpool's clusterset
		rejectStatus.Result.Message = "Do not allow to update clusterpool's clusterset label"
		return rejectStatus
	}

	//For clusterdeployment, we need to update clusterdeployment's clusterset, because the deployment
	// controller need to update the clusterset label.
	return a.allowUpdate(request.UserInfo, originalClusterSetName, currentClusterSetName)
}

func (a *AdmissionHandler) allowUpdate(userInfo authenticationv1.UserInfo, originalClusterSetName, currentClusterSetName string) *v1.AdmissionResponse {
	if status := a.allowUpdateClusterSet(userInfo, originalClusterSetName); !status.Allowed {
		return status
	}
	return a.allowUpdateClusterSet(userInfo, currentClusterSetName)
}
func (a *AdmissionHandler) isUpdateClusterset(request *v1.AdmissionRequest) (bool, string, string, error) {
	oldObj := unstructured.Unstructured{}
	err := oldObj.UnmarshalJSON(request.OldObject.Raw)
	if err != nil {
		return false, "", "", err
	}

	newObj := unstructured.Unstructured{}
	err = newObj.UnmarshalJSON(request.Object.Raw)
	if err != nil {
		return false, "", "", err
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
		return false, "", "", nil
	}
	return true, originalClusterSetName, currentClusterSetName, nil
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
