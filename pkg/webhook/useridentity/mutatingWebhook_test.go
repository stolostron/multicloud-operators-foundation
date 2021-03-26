package useridentity

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

func newAdmissionHandler() *AdmissionHandler {
	var objs []runtime.Object
	client := kubefake.NewSimpleClientset(objs...)
	return &AdmissionHandler{
		Lister:        nil,
		KubeClient:    client,
		DynamicClient: nil,
	}
}

const (
	channelTest = `{"apiVersion":"apps.open-cluster-management.io/v1","kind":"Channel","metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"apps.open-cluster-management.io/v1\",\"kind\":\"Channel\",\"metadata\":{\"annotations\":{},\"name\":\"test\",\"namespace\":\"default\"},\"spec\":{\"pathname\":\"https://github.com/open-cluster-management/abc.git\",\"type\":\"Git\"}}\n"},"creationTimestamp":null,"managedFields":[{"apiVersion":"apps.open-cluster-management.io/v1","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:annotations":{".":{},"f:kubectl.kubernetes.io/last-applied-configuration":{}}},"f:spec":{".":{},"f:pathname":{},"f:type":{}}},"manager":"kubectl-client-side-apply","operation":"Update","time":"2021-03-26T07:23:28Z"}],"name":"test","namespace":"default"},"spec":{"pathname":"https://github.com/open-cluster-management/abc.git","type":"Git"}}`
)

func newAdmissionReview() *v1.AdmissionReview {
	return &v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Request: &v1.AdmissionRequest{
			UID: "3f6d1d1f-61a0-49ea-b458-05454ba42ab6",
			Kind: metav1.GroupVersionKind{
				Group:   "apps.open-cluster-management.io",
				Kind:    "Channel",
				Version: "v1",
			},
			Resource: metav1.GroupVersionResource{
				Group:    "apps.open-cluster-management.io",
				Version:  "v1",
				Resource: "channels",
			},
			SubResource: "",
			RequestKind: &metav1.GroupVersionKind{
				Group:   "apps.open-cluster-management.io",
				Kind:    "Channel",
				Version: "v1",
			},
			RequestResource: &metav1.GroupVersionResource{
				Group:    "apps.open-cluster-management.io",
				Version:  "v1",
				Resource: "channels",
			},
			RequestSubResource: "",
			Name:               "test",
			Namespace:          "default",
			Operation:          "CREATE",
			UserInfo: authenticationv1.UserInfo{
				Username: "system:admin",
			},
			Object: runtime.RawExtension{
				Raw: []byte(channelTest),
			},
		},
		Response: nil,
	}
}

func TestMutateResource(t *testing.T) {
	adHandler := newAdmissionHandler()
	rsp := adHandler.mutateResource(*newAdmissionReview())
	assert.True(t, true, len(rsp.Patch) != 0)
}
