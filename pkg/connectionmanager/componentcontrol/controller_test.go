// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package componentcontrol

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func newController() *Controller {
	pod1 := &corev1.Pod{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
	}

	secret1 := &corev1.Secret{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "secret1",
			Namespace: "default",
		},
		Data: map[string][]byte{
			common.HubConfigSecretKey: []byte("abc"),
		},
		StringData: nil,
		Type:       "",
	}
	objects := []runtime.Object{pod1, secret1}

	kubeClient := k8sfake.NewSimpleClientset(objects...)

	options := NewControlOptions()
	options.AddFlags(pflag.CommandLine)
	options.KlusterletLabels = "owner=IBM"
	options.KlusterletSecret = "default/secret1"
	return options.ComponentControl(kubeClient)
}

func TestController(t *testing.T) {
	controller := newController()

	err := controller.RestartKlusterlet()
	if err != nil {
		t.Errorf("fail to restart klusterlet")
	}

	controller.klusterletSecretName = "secret"
	controller.klusterletSecretNamespace = "default"
	_, err = controller.UpdateKlusterletSecret([]byte{})
	if err != nil {
		t.Errorf("fail to create klusterlete secret")
	}
	_, err = controller.GetKlusterletSecret()
	if err != nil {
		t.Errorf("fail to get klusterlet secret")
	}

	controller.klusterletSecretName = "secret1"
	controller.klusterletSecretNamespace = "default"
	_, err = controller.UpdateKlusterletSecret([]byte("abc"))
	if err != nil {
		t.Errorf("fail to update klusterlete secret")
	}

	controller.klusterletSecretName = "secret3"
	_, err = controller.GetKlusterletSecret()
	if err != nil {
		t.Errorf("fail to get klusterlet secret")
	}
}
