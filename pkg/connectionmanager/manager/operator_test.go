// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package manager

import (
	"bytes"
	"testing"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/common"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/componentcontrol"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/genericoptions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func newOperator() *Operator {
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

	operator := NewOperator(config, "cluster1", "cluster1", "default", "klusterlet-secrect", nil)
	return operator
}

func TestReconnectToServer(t *testing.T) {
	operator := newOperator()
	server := operator.server
	operator.bootstrap()

	if operator.server == server {
		t.Errorf("failed to reconnect")
	}
}

func TestHandleKlusterletSecretChange(t *testing.T) {
	operator := newOperator()
	stopCh := make(chan struct{})
	defer close(stopCh)

	go operator.ksinformer.Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, operator.ksinformer.HasSynced); !ok {
		return
	}

	server := operator.server
	err := operator.handleKlusterletSecretChange("default/klusterlet-secrect")
	if err != nil {
		t.Errorf("failed to handle change of klusterlet secret: %+v", err)
	} else if operator.server == server {
		t.Errorf("failed to handle change of klusterlet secret")
	}
}

func TestSetupServer(t *testing.T) {
	operator := newOperator()

	key, cert, _ := common.NewCertKey("test.com", "hcm")
	kconfig := common.NewClientConfig("localhost", cert, key)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"kubeconfig": kconfig,
		},
	}

	server, err := operator.setupServer(secret)
	if err != nil {
		t.Errorf("failed to setup server with secret: %+v", err)
	} else if server.Host() != "localhost" {
		t.Errorf("failed to handle change of klusterlet secret")
	}
}

func TestHandleCertRotation(t *testing.T) {
	operator := newOperator()

	key, cert, _ := common.NewCertKey("test.com", "hcm")
	kconfig := common.NewClientConfig("server1", cert, key)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"kubeconfig": kconfig,
		},
	}
	server, err := operator.setupServer(secret)
	if err != nil {
		t.Errorf("failed to setup server with secret: %+v", err)
	} else if server.Host() != "server1" {
		t.Errorf("failed to handle change of klusterlet secret")
	}

	operator.server = server
	operator.handleCertRotation()

	updatedSecret, err := operator.kubeclient.CoreV1().Secrets("default").Get("klusterlet-secrect", metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to handle cert rotation: %+v", err)
	} else if equal := bytes.Equal(secret.Data[common.HubConfigSecretKey], updatedSecret.Data[common.HubConfigSecretKey]); !equal {
		t.Errorf("failed to handle cert rotation")
	}
}
