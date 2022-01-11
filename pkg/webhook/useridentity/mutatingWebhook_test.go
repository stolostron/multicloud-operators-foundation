package useridentity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func newAdmissionHandler() *AdmissionHandler {
	return &AdmissionHandler{
		Lister:                nil,
		SkipOverwriteUserList: []string{"system:serviceaccount:open-cluster-management-agent-addon:klusterlet-addon-appmgr"},
	}
}

const (
	channelTest = `{"apiVersion":"apps.open-cluster-management.io/v1","kind":"Channel","metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"apps.open-cluster-management.io/v1\",\"kind\":\"Channel\",\"metadata\":{\"annotations\":{},\"name\":\"test\",\"namespace\":\"default\"},\"spec\":{\"pathname\":\"https://github.com/stolostron/abc.git\",\"type\":\"Git\"}}\n"},"creationTimestamp":null,"managedFields":[{"apiVersion":"apps.open-cluster-management.io/v1","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:annotations":{".":{},"f:kubectl.kubernetes.io/last-applied-configuration":{}}},"f:spec":{".":{},"f:pathname":{},"f:type":{}}},"manager":"kubectl-client-side-apply","operation":"Update","time":"2021-03-26T07:23:28Z"}],"name":"test","namespace":"default"},"spec":{"pathname":"https://github.com/stolostron/abc.git","type":"Git"}}`
	appsubTest  = `{"apiVersion":"apps.open-cluster-management.io/v1","kind": "Subscription","metadata": {"name": "git-sub","namespace": "parentsub","annotations": {"apps.open-cluster-management.io/cluster-admin": "true","apps.open-cluster-management.io/github-path": "test/e2e/github/nestedSubscription"}},"spec": {"channel": "ch-git/git","placement": {"local": "true"}}}`
)

func newAdmissionRequest() *v1.AdmissionRequest {
	return &v1.AdmissionRequest{
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
	}
}

func appSubAdmissionRequest() *v1.AdmissionRequest {
	return &v1.AdmissionRequest{
		UID: "4d6d1d1f-61a0-49ea-b458-05454ba42ab6",
		Kind: metav1.GroupVersionKind{
			Group:   "apps.open-cluster-management.io",
			Kind:    "Subscription",
			Version: "v1",
		},
		Resource: metav1.GroupVersionResource{
			Group:    "apps.open-cluster-management.io",
			Version:  "v1",
			Resource: "subscriptions",
		},
		SubResource: "",
		RequestKind: &metav1.GroupVersionKind{
			Group:   "apps.open-cluster-management.io",
			Kind:    "Subscription",
			Version: "v1",
		},
		RequestResource: &metav1.GroupVersionResource{
			Group:    "apps.open-cluster-management.io",
			Version:  "v1",
			Resource: "subscriptions",
		},
		RequestSubResource: "",
		Name:               "subtest",
		Namespace:          "default",
		Operation:          "CREATE",
		UserInfo: authenticationv1.UserInfo{
			Username: "system:serviceaccount:open-cluster-management-agent-addon:klusterlet-addon-appmgr",
		},
		Object: runtime.RawExtension{
			Raw: []byte(appsubTest),
		},
	}
}

func TestMutateResource(t *testing.T) {
	adHandler := newAdmissionHandler()
	rsp := adHandler.mutateResource(newAdmissionRequest())
	assert.True(t, true, len(rsp.Patch) != 0)

	rsp = adHandler.mutateResource(appSubAdmissionRequest())
	assert.True(t, rsp.Patch == nil)
}
