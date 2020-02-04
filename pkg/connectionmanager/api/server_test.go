// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package api

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newSecrets() (map[string]*corev1.Secret, *corev1.Secret) {
	config := common.NewClientConfig("localhost", nil, nil)

	boot1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "boot-1",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"kubeconfig": config,
		},
	}

	boot2 := boot1.DeepCopy()
	boot2.Name = "boot-2"

	config3 := boot1.DeepCopy()
	config3.Name = "config-3"

	key, cert, _ := common.NewCertKey("test.com", "hcm")
	hub1config := common.NewClientConfig("hub1", cert, key)
	hub2config := common.NewClientConfig("hub2", cert, key)
	store := map[string][]byte{
		"hub1-kubeconfig": hub1config,
		"hub2-kubeconfig": hub2config,
		"hub2tls.key":     key,
		"hub2tls.crt":     cert,
		"hub3tls.key":     key,
		"hub3tls.crt":     cert,
	}

	secstore := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secstore",
			Namespace: "default",
		},
		Data: store,
	}

	return map[string]*corev1.Secret{"boot-1": boot1, "boot-2": boot2, "config-3": config3}, secstore
}

func TestLoadBootsrapServer(t *testing.T) {
	secrets, _ := newSecrets()

	secgetter := SecretGetterFunc(func(namespace string, name string) (*corev1.Secret, error) {
		return secrets[name], nil
	})
	host, server, _ := LoadBootstrapServerInfo(secrets["boot-1"], secgetter, "cluster1", "cluster1")
	if server == nil {
		t.Errorf("failed to load server info")
	}

	if host != "localhost" {
		t.Errorf("host is not correct")
	}
}
