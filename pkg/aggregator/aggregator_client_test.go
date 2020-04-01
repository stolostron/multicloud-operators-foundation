// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func Test_Getter(t *testing.T) {
	secret1 := newSecret("default", "finding-ca")
	objects := []runtime.Object{
		secret1,
	}

	kubeClient := k8sfake.NewSimpleClientset(objects...)
	getter := NewInfoGetters(kubeClient)
	opt := &ClientOptions{
		name:        "test",
		subResource: "sub",
		secret:      "default/finding-ca",
		path:        "/path",
		service:     "default/svr"}

	getter.AddAndUpdate(opt)
	getter.AddAndUpdate(&ClientOptions{name: "test", subResource: "test"})
	getter.Delete("test")

	if len(getter.nameToSubResource) != 0 {
		t.Errorf("fail to test getter")
	}

	connGetter := NewConnectionInfoGetter(opt, nil)
	_, _, err := connGetter.GetConnectionInfo(context.Background(), "test")
	if err != nil {
		t.Errorf("fail to test connGetter")
	}
}
