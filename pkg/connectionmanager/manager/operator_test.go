// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package manager

import (
	"testing"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/common"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/componentcontrol"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/genericoptions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func newOperator(stopCh <-chan struct{}) *Operator {
	key, cert, _ := common.NewCertKey("test.com", "hcm")
	kconfig := common.NewClientConfig("localhost", cert, key)
	bootstrapSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bootstrap-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"kubeconfig": kconfig,
		},
	}
	klusterletSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "klusterlet-secrect",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"kubeconfig": kconfig,
		},
	}

	Kubeclient := kubefake.NewSimpleClientset(bootstrapSecret, klusterletSecret)

	controlOptions := componentcontrol.NewControlOptions()
	controlOptions.KlusterletSecret = "default/klusterlet-secrect"
	config := &genericoptions.GenericConfig{
		BootstrapSecret:  bootstrapSecret,
		Kubeclient:       Kubeclient,
		ComponentControl: controlOptions.ComponentControl(Kubeclient),
	}

	operator := NewOperator(config, "cluster1", "cluster1", nil)
	return operator
}

func TestReconnectToServer(t *testing.T) {
	stopCh := make(chan struct{})
	defer close(stopCh)

	operator := newOperator(stopCh)
	server := operator.server
	operator.bootstrapAll()

	if operator.server == server {
		t.Errorf("failed to reconnect")
	}
}
