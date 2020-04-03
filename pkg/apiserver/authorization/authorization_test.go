package authorization

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestApplyTo(t *testing.T) {
	hcmAuthorization := HcmDelegatingAuthorizationOptions{
		DelegatingAuthorizationOptions: nil,
		QPS:                            500,
		Burst:                          100,
	}
	authorizationInfo := &server.AuthorizationInfo{}
	if err := hcmAuthorization.ApplyTo(authorizationInfo); err != nil {
		t.Errorf("case1 fail to applyTo authorization")
	}

	hcmAuthorization.DelegatingAuthorizationOptions = &genericoptions.DelegatingAuthorizationOptions{}

	objects := []runtime.Object{}
	kubeClient := k8sfake.NewSimpleClientset(objects...)
	_, err := hcmAuthorization.toAuthorizer(kubeClient)
	if err != nil {
		t.Errorf("case3 fail to authorizer")
	}
}
