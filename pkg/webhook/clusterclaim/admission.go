package clusterclaim

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	clusterv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

var claimGVR = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "managedclusterclaims",
}

// DenyClaim checks if the create/update operation on the managedclusterclaim should be denied or not
func DenyClaim(request *admissionv1beta1.AdmissionRequest, kubeClient kubernetes.Interface) (bool, string) {
	claim := &clusterv1alpha1.ManagedClusterClaim{}
	if err := json.Unmarshal(request.Object.Raw, claim); err != nil {
		return true, fmt.Sprintf("Unable to unmarshal ManagedClusterClaim: %v", err)
	}

	// deny claim neither ClusterName nor Selector is  specified
	if len(claim.Spec.ClusterName) == 0 && claim.Spec.Selector == nil {
		return true, "Either ClusterName or Selector should be specified for ManagedClusterClaim"
	}

	if request.Operation == admissionv1beta1.Update {
		oldClaim := &clusterv1alpha1.ManagedClusterClaim{}
		if err := json.Unmarshal(request.OldObject.Raw, oldClaim); err != nil {
			return true, fmt.Sprintf("Unable to unmarshal ManagedClusterClaim: %v", err)
		}

		// skip checking user permission if spec is not changed
		if reflect.DeepEqual(claim.Spec, oldClaim.Spec) {
			return false, ""
		}

		// it is not allowed to modify the spec of a bound claim
		if len(oldClaim.Status.ClusterName) != 0 {
			return true, fmt.Sprintf("Unable to update spec of ManagedClusterClaim %s/%s because it has already been bound", claim.Namespace, claim.Name)
		}
	}

	// check if user has get permission on bind subresource of managed cluster
	if allowed, err := allowBindingToCluster(claim.Spec.ClusterName, request.UserInfo, kubeClient); err != nil {
		return true, fmt.Sprintf("Unable to check user permission on managed cluster: %v", err)
	} else if allowed {
		return false, ""
	}

	message := "User is not allowed to bind claim to cluster"
	if len(claim.Spec.ClusterName) != 0 {
		message = fmt.Sprintf("%s %q", message, claim.Spec.ClusterName)
	}
	return true, message
}

// allowBindingToCluster checks if the user is able to bind a managed cluster to claim
func allowBindingToCluster(clusterName string, userInfo authenticationv1.UserInfo, kubeClient kubernetes.Interface) (bool, error) {
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
				Resource:    "managedclusters",
				Subresource: "bind",
				Verb:        "get",
				Name:        clusterName,
			},
		},
	}
	sar, err := kubeClient.AuthorizationV1().SubjectAccessReviews().Create(context.TODO(), sar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}

	return sar.Status.Allowed, nil
}
