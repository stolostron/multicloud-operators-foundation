// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func newSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"ca.crt":  []byte("abc"),
			"tls.crt": []byte("abc"),
			"tls.key": []byte("abc"),
		},
		StringData: nil,
		Type:       "",
	}
}

func Test_Getter(t *testing.T) {
	connectionOption := NewConnectionOption()
	connectionOption.Host = "https://test.sv"
	connectionOption.Secret = "default/finding-ca"

	secret1 := newSecret("default", "finding-ca")
	objects := []runtime.Object{
		secret1,
	}

	kubeClient := k8sfake.NewSimpleClientset(objects...)
	opt := NewConnectionInfoGetter(connectionOption, kubeClient, "test")
	opt.GetConnectionInfo(context.Background(), "test")
}
