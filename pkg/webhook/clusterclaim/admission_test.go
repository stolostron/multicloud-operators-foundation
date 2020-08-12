package clusterclaim

import (
	"encoding/json"
	"testing"

	clusterv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1alpha1"
	"github.com/stretchr/testify/assert"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func newManagedClusterClaim(clusterName string, selector *metav1.LabelSelector, labels map[string]string, bound bool) runtime.RawExtension {
	claim := &clusterv1alpha1.ManagedClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "claim1",
			Namespace: "default",
			Labels:    labels,
		},
		Spec: clusterv1alpha1.ManagedClusterClaimSpec{
			ClusterName: clusterName,
			Selector:    selector,
		},
	}

	if bound {
		claim.Status.ClusterName = "cluster1"
	}

	data, _ := json.Marshal(claim)
	return runtime.RawExtension{
		Raw: data,
	}
}

func TestDenyClaim(t *testing.T) {
	tests := []struct {
		name             string
		object           runtime.RawExtension
		oldObject        runtime.RawExtension
		operation        admissionv1beta1.Operation
		allowBindCluster bool
		expectedResult   bool
		expectedMessage  string
	}{
		{
			name:            "deny claim creation when neither clusterName nor selector is specified",
			object:          newManagedClusterClaim("", nil, nil, false),
			operation:       admissionv1beta1.Create,
			expectedResult:  true,
			expectedMessage: "Either ClusterName or Selector should be specified for ManagedClusterClaim",
		},
		{
			name:            "deny spec modification when claim is bound",
			object:          newManagedClusterClaim("cluster2", nil, nil, false),
			oldObject:       newManagedClusterClaim("cluster1", nil, nil, true),
			operation:       admissionv1beta1.Update,
			expectedResult:  true,
			expectedMessage: "Unable to update spec of ManagedClusterClaim default/claim1 because it has already been bound",
		},
		{
			name: "allow changing labels even after claim is bound",
			object: newManagedClusterClaim("cluster1", nil, map[string]string{
				"abc": "def",
			}, false),
			oldObject:      newManagedClusterClaim("cluster1", nil, nil, true),
			operation:      admissionv1beta1.Update,
			expectedResult: false,
		},
		{
			name:             "deny claim creation when user has no permission to bind claim to cluter",
			object:           newManagedClusterClaim("cluster1", nil, nil, false),
			operation:        admissionv1beta1.Create,
			allowBindCluster: false,
			expectedResult:   true,
			expectedMessage:  "User is not allowed to bind claim to cluster \"cluster1\"",
		},
		{
			name:             "allow claim creation",
			object:           newManagedClusterClaim("cluster1", nil, nil, false),
			operation:        admissionv1beta1.Create,
			allowBindCluster: true,
			expectedResult:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeClient := kubefake.NewSimpleClientset()
			kubeClient.PrependReactor(
				"create",
				"subjectaccessreviews",
				func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, &authorizationv1.SubjectAccessReview{
						Status: authorizationv1.SubjectAccessReviewStatus{
							Allowed: test.allowBindCluster,
						},
					}, nil
				},
			)

			request := &admissionv1beta1.AdmissionRequest{
				Object:    test.object,
				OldObject: test.oldObject,
				Operation: test.operation,
			}

			result, message := DenyClaim(request, kubeClient)
			assert.Equal(t, test.expectedResult, result)
			assert.Equal(t, test.expectedMessage, message)
		})
	}
}
