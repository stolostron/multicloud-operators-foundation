package validating

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	"github.com/stolostron/cluster-lifecycle-api/helpers/localcluster"
	v1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clusterlisterv1 "open-cluster-management.io/api/client/cluster/listers/cluster/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stolostron/multicloud-operators-foundation/pkg/constants"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/serve"
)

type AdmissionHandler struct {
	HiveClient    client.Client
	KubeClient    kubernetes.Interface
	ClusterLister clusterlisterv1.ManagedClusterLister
}

var managedClustersGVR = metav1.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
}

var managedClusterSetsGVR = metav1.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1beta2",
	Resource: "managedclustersets",
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

var agentClusterOwnerRef = metav1.OwnerReference{
	Kind:       "AgentCluster",
	APIVersion: "capi-provider.agent-install.openshift.io/v1alpha1",
}

func (a *AdmissionHandler) ValidateResource(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	switch request.Resource {
	case managedClusterSetsGVR:
		return a.validateManagedClusterSet(request)
	case clusterDeploymentsGVR:
		return a.validateClusterDeployment(request)
	case clusterPoolsGVR:
		return a.validateClusterPool(request)
	case managedClustersGVR:
		return a.validateManagedCluster(request)
	default:
		return a.responseAllowed()
	}
}

// validateManagedClusterSet validates the managed cluster set,
// only allow creating/updating global clusterset and legacy clusterset
func (a *AdmissionHandler) validateManagedClusterSet(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	switch request.Operation {
	case v1.Create, v1.Update:
		clusterset := &clusterv1beta2.ManagedClusterSet{}
		if err := json.Unmarshal(request.Object.Raw, clusterset); err != nil {
			return a.responseNotAllowed(err.Error())
		}
		//allow creating/updating global clusterset
		if clusterset.Name == clustersetutils.GlobalSetName {
			if equality.Semantic.DeepEqual(clusterset.Spec, clustersetutils.GlobalSet.Spec) {
				return a.responseAllowed()
			}
			return a.responseNotAllowed("Do not allow to create/update global clusterset")
		}

		//allow creating/updating legacy clusterset
		if clusterset.Spec.ClusterSelector.SelectorType == clusterv1beta2.ExclusiveClusterSetLabel {
			return a.responseAllowed()
		}

		return a.responseNotAllowed("Do not allow to create/update this kind of clusterset")
	}

	return a.responseAllowed()
}

func (a *AdmissionHandler) validateClusterDeployment(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	switch request.Operation {
	case v1.Create:
		// ignore clusterDeployment created by clusterpool
		clusterDeployment := &hivev1.ClusterDeployment{}
		if err := json.Unmarshal(request.Object.Raw, clusterDeployment); err != nil {
			return a.responseNotAllowed(err.Error())
		}
		if clusterDeployment.Spec.ClusterPoolRef != nil {
			return a.responseAllowed()
		}

		//Requirment from AI integrate with hypershift:
		//  https://github.com/openshift/cluster-api-provider-agent/pull/38/files
		if hasOwnerRef(clusterDeployment.ObjectMeta.OwnerReferences, agentClusterOwnerRef) {
			return a.responseAllowed()
		}
		return a.validateClusterSetJoinPermission(request)
	case v1.Update:
		updateClusterSet, response, oldClusterSet, newClusterSet := a.validateUpdateClusterSet(request)
		if !response.Allowed || !updateClusterSet {
			return response
		}

		// The clusterdeployment is trying to update clusterset, user "could" update the clusterset label if
		// he/she has permission, but clusterdeployment controller will always sync the clusterdeployment's
		// clusterset label back.
		return a.validateAllowUpdateClusterSet(request.UserInfo, oldClusterSet, newClusterSet)
	case v1.Delete:
		return a.validatingIsHypershiftHostingCluster(request.Name)
	}

	return a.responseAllowed()
}

func (a *AdmissionHandler) validateClusterPool(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	switch request.Operation {
	case v1.Create:
		return a.validateClusterSetJoinPermission(request)
	case v1.Update:
		updateClusterSet, response, _, _ := a.validateUpdateClusterSet(request)
		if !response.Allowed || !updateClusterSet {
			return response
		}

		// do not allow to update clusterpool's clusterset
		return a.responseNotAllowed("Do not allow to update clusterpool's clusterset label")
	}

	return a.responseAllowed()
}

func (a *AdmissionHandler) validateManagedCluster(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	switch request.Operation {
	case v1.Create:
		resp := a.validateLocalClusterCreate(request)
		if !resp.Allowed {
			return resp
		}
		return a.validateClusterSetJoinPermission(request)
	case v1.Update:
		resp := a.validateLocalClusterUpdate(request)
		if !resp.Allowed {
			return resp
		}
		updateClusterSet, response, oldClusterSet, newClusterSet := a.validateUpdateClusterSet(request)
		if !response.Allowed || !updateClusterSet {
			return response
		}

		// The managedClusters is trying to update clusterset, check if the managedcluster is claimed from clusterpool
		//   1. if the managedcluster is claimed from clusterpool, do not allow to update clusterset
		//   2. if the managedcluster is not claimed from clusterpool, check if user has permission to update clusterset
		//
		// Notes: clusterclaims-controller will auto create a mangedcluster with the clusterpool's clusterset label if
		//        there is a clusterclaim ref to this clusterpool.
		// https://github.com/stolostron/clusterclaims-controller/blob/main/controllers/clusterclaims/clusterclaims-controller.go
		managedCluster := &clusterv1.ManagedCluster{}
		if err := json.Unmarshal(request.Object.Raw, managedCluster); err != nil {
			return a.responseNotAllowed(err.Error())
		}

		// Get managedcluster's clusterdeployment
		// 1. No related clusterdeployment, check if user has permission to update this cluster.
		// 2. The related clusterdeployment is created from clusterpool, do not allow user to
		//    update the managedcluster.
		// 3. The related clusterdeployment is not created from clusterpool, check if user has
		//    permission to update the managedcluster.
		clusterDeployment := &hivev1.ClusterDeployment{}
		err := a.HiveClient.Get(context.TODO(), types.NamespacedName{Name: managedCluster.Name, Namespace: managedCluster.Name}, clusterDeployment)
		if err != nil {
			if errors.IsNotFound(err) {
				// Do not find clusterdeployment in managedcluster namespace. allow to update clusterset label.
				return a.validateAllowUpdateClusterSet(request.UserInfo, oldClusterSet, newClusterSet)
			}
			return a.responseNotAllowed(err.Error())
		}

		// the cluster is claimed from clusterpool, do not allow to update clustersetlabel.
		if clusterDeployment.Spec.ClusterPoolRef != nil {
			//For upgrade from 2.4 to 2.5, we need to move all clusters from empty set to "default" set.
			if newClusterSet == clustersetutils.DefaultSetName && len(oldClusterSet) == 0 {
				return a.validateAllowUpdateClusterSet(request.UserInfo, oldClusterSet, newClusterSet)
			}
			return a.responseNotAllowed("Do not allow to update claimed cluster's clusterset")
		}
		// the clusterdeployment is not claimed from clusterpool, allow to update clustersetlabel.
		return a.validateAllowUpdateClusterSet(request.UserInfo, oldClusterSet, newClusterSet)
	case v1.Delete:
		clusterName := request.Name
		clusters, err := a.ClusterLister.List(labels.Everything())
		if err != nil {
			return a.responseNotAllowed(err.Error())
		}

		var hostedClusters = make([]string, 0)
		for _, cluster := range clusters {
			if cluster.GetAnnotations()[apiconstants.AnnotationKlusterletDeployMode] != "Hosted" {
				continue
			}
			if cluster.GetAnnotations()[apiconstants.AnnotationKlusterletHostingClusterName] != clusterName {
				continue
			}
			hostedClusters = append(hostedClusters, cluster.Name)
		}

		if len(hostedClusters) > 0 {
			return a.responseNotAllowed(fmt.Sprintf(
				"Not allowed to delete, please delete the hosted clusters %v first", hostedClusters))
		}

		return a.validatingIsHypershiftHostingCluster(clusterName)
	}

	return a.responseAllowed()
}

func (a *AdmissionHandler) validatingIsHypershiftHostingCluster(managedClusterName string) *v1.AdmissionResponse {
	managedCluster, err := a.ClusterLister.Get(managedClusterName)
	if err != nil {
		if errors.IsNotFound(err) {
			return a.responseAllowed()
		}
		return a.responseNotAllowed(err.Error())
	}

	if !managedCluster.DeletionTimestamp.IsZero() {
		return a.responseAllowed()
	}

	hasHostedClusters := false
	for _, claim := range managedCluster.Status.ClusterClaims {
		if claim.Name == constants.ClusterClaimHostedClusterCountZero && strings.EqualFold(claim.Value, "false") {
			hasHostedClusters = true
		}
	}

	// the ClusterClaimKeyHostedClusterCountZero is managed by the hypershift addon, so here we check if the addon
	// is available;
	// We only deny this delete request with 100% certainty, because this webhook on delete requests is dangerous
	if hasHostedClusters && meta.IsStatusConditionPresentAndEqual(managedCluster.Status.Conditions,
		clusterv1.ManagedClusterConditionAvailable, metav1.ConditionTrue) {
		if value, ok := managedCluster.GetLabels()[constants.LabelFeatureHypershiftAddon]; ok && value == "available" {
			return a.responseNotAllowed("Not allowed to delete, the cluster is hosting some hypershift clusters")
		}
	}

	return a.responseAllowed()
}

// validateCreateRequest validates:
// 1. the user has managedClusterSet/join <managedClusterSet> permission to create resource with a <managedClusterSet> label.
// 2. the user has managedClusterSet/join <all> permission to create resource without a managedClusterSet label.
func (a *AdmissionHandler) validateClusterSetJoinPermission(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	obj := unstructured.Unstructured{}
	err := obj.UnmarshalJSON(request.Object.Raw)
	if err != nil {
		return a.responseNotAllowed(err.Error())
	}

	clusterSetName := obj.GetLabels()[clusterv1beta2.ClusterSetLabel]
	return a.allowUpdateClusterSet(request.UserInfo, clusterSetName)
}

func (a *AdmissionHandler) validateAllowUpdateClusterSet(userInfo authenticationv1.UserInfo,
	originalClusterSetName, currentClusterSetName string) *v1.AdmissionResponse {
	if response := a.allowUpdateClusterSet(userInfo, originalClusterSetName); !response.Allowed {
		return response
	}
	return a.allowUpdateClusterSet(userInfo, currentClusterSetName)
}

// validateUpdateClusterSet check if the request is trying to update the clusterset
func (a *AdmissionHandler) validateUpdateClusterSet(
	request *v1.AdmissionRequest) (bool, *v1.AdmissionResponse, string, string) {
	oldObj := unstructured.Unstructured{}
	err := oldObj.UnmarshalJSON(request.OldObject.Raw)
	if err != nil {
		return false, a.responseNotAllowed(fmt.Sprintf("Unmarshal oldObject error: %v", err)), "", ""
	}

	newObj := unstructured.Unstructured{}
	err = newObj.UnmarshalJSON(request.Object.Raw)
	if err != nil {
		return false, a.responseNotAllowed(fmt.Sprintf("Unmarshal object error: %v", err)), "", ""
	}

	originalClusterSetName := oldObj.GetLabels()[clusterv1beta2.ClusterSetLabel]
	currentClusterSetName := newObj.GetLabels()[clusterv1beta2.ClusterSetLabel]

	// allow if managedClusterSet label is not updated
	if originalClusterSetName == currentClusterSetName {
		return false, a.responseAllowed(), "", ""
	}
	return true, a.responseAllowed(), originalClusterSetName, currentClusterSetName
}

// allowUpdateClusterSet checks whether a request user has been authorized to add/remove a resource to/from the
// ManagedClusterSet. check if the user has clusterSet/join <all> permission when clusterSetName is null.
func (a *AdmissionHandler) allowUpdateClusterSet(
	userInfo authenticationv1.UserInfo,
	clusterSetName string) *v1.AdmissionResponse {

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
	sar, err := a.KubeClient.AuthorizationV1().SubjectAccessReviews().Create(
		context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return a.responseNotAllowed(err.Error())
	}

	if !sar.Status.Allowed {
		return a.responseNotAllowed(fmt.Sprintf(
			"user %q cannot add/remove the resource to/from ManagedClusterSet %q",
			userInfo.Username, clusterSetName))
	}

	return a.responseAllowed()
}

func (a *AdmissionHandler) ServeValidateResource(w http.ResponseWriter, r *http.Request) {
	serve.Serve(w, r, a.ValidateResource)
}

func (a *AdmissionHandler) responseAllowed() *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: true,
	}
}

func (a *AdmissionHandler) responseNotAllowed(msg string) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
			Message: msg,
		},
	}
}

// validateLocalClusterCreate check if the cluster can be created as the local cluster. Only one cluster in
// the hub can be created as the local cluster.
func (a *AdmissionHandler) validateLocalClusterCreate(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	newCluster := &clusterv1.ManagedCluster{}
	err := json.Unmarshal(request.Object.Raw, newCluster)
	if err != nil {
		return a.responseNotAllowed(err.Error())
	}

	if localcluster.IsClusterSelfManaged(newCluster) {
		clusters, err := a.ClusterLister.List(labels.Everything())
		if err != nil {
			return a.responseNotAllowed(err.Error())
		}
		for _, cluster := range clusters {
			if localcluster.IsClusterSelfManaged(cluster) {
				return &v1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
						Message: fmt.Sprintf("cluster %s is already local cluster, cannot create another local cluster", cluster.Name),
					},
				}
			}
		}
	}

	return a.responseAllowed()
}

func (a *AdmissionHandler) validateLocalClusterUpdate(request *v1.AdmissionRequest) *v1.AdmissionResponse {
	oldCluster := &clusterv1.ManagedCluster{}
	newCluster := &clusterv1.ManagedCluster{}

	err := json.Unmarshal(request.Object.Raw, newCluster)
	if err != nil {
		return a.responseNotAllowed(err.Error())
	}

	err = json.Unmarshal(request.OldObject.Raw, oldCluster)
	if err != nil {
		return a.responseNotAllowed(err.Error())
	}

	if (localcluster.IsClusterSelfManaged(newCluster) && !localcluster.IsClusterSelfManaged(oldCluster)) ||
		(!localcluster.IsClusterSelfManaged(newCluster) && localcluster.IsClusterSelfManaged(oldCluster)) {
		return &v1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: fmt.Sprintf("cluster %s is not allowed to update local-cluster label", newCluster.Name),
			},
		}
	}

	return a.responseAllowed()
}

func hasOwnerRef(ownerRefs []metav1.OwnerReference, wantRef metav1.OwnerReference) bool {
	if len(ownerRefs) <= 0 {
		return false
	}
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind != wantRef.Kind {
			continue
		}

		curGroupVersion, _ := schema.ParseGroupVersion(ownerRef.APIVersion)
		wantGroupVersion, _ := schema.ParseGroupVersion(wantRef.APIVersion)
		if curGroupVersion.Group != wantGroupVersion.Group {
			continue
		}
		return true
	}
	return false
}
