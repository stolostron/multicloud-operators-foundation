// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package api

import (
	"crypto/x509/pkix"
	"reflect"
	"testing"

	v1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/clusterbootstrap"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	csrv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

func newRequestName(key []byte) string {
	subject := &pkix.Name{
		Organization: []string{"hcm:clusters"},
		CommonName:   "hcm:clusters:test:test",
	}

	return clusterbootstrap.DigestedName(key, subject)
}

func newConnection(clientConfig, clientCert, clientKey []byte) *ServerConnection {
	btConfig := common.NewClientConfig("localhost", nil, nil)
	serverConn, _, _ := NewServerConnection(btConfig, clientConfig, clientCert, clientKey, "cluster1", "cluster1", "")

	hcmjoin := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: newRequestName(clientKey),
		},
		Status: v1alpha1.ClusterJoinStatus{
			Phase: v1alpha1.JoinApproved,
			CSRStatus: csrv1beta1.CertificateSigningRequestStatus{
				Certificate: clientCert,
			},
		},
	}

	btclient := hcmfake.NewSimpleClientset(hcmjoin)
	serverConn.bootstrapper = clusterbootstrap.NewBootStrapper(btclient, "localhost", "cluster1", "cluster1", clientKey, nil)

	return serverConn
}

func TestBootStrap(t *testing.T) {
	key, cert, _ := common.NewCertKey("test.com", "hcm")
	server := newConnection(nil, cert, key)

	err := server.Bootstrap()
	if err != nil {
		t.Errorf("suppose to bootsrap: %v", err)
	}

	configData := server.ConnInfo()
	obj, err := runtime.Decode(clientcmdlatest.Codec, configData)
	if err != nil {
		t.Errorf("failed to decode config: %v", err)
	}

	clientConfig, ok := obj.(*clientcmdapi.Config)
	if !ok {
		t.Errorf("cannot format clientConfig")
	}

	if !reflect.DeepEqual(clientConfig.AuthInfos["default-auth"].ClientCertificateData, cert) {
		t.Errorf("cert data should be equal")
	}
}

func TestHandleJoinRequest(t *testing.T) {
	key, cert, _ := common.NewCertKey("test.com", "hcm")
	server := newConnection(nil, cert, key)
	err := server.Bootstrap()
	if err != nil {
		t.Errorf("suppose to bootsrap: %v", err)
	}

	_, cert2, _ := common.NewCertKey("test.com", "hcm")
	hcmjoin := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: v1alpha1.ClusterJoinStatus{
			Phase: v1alpha1.JoinApproved,
			CSRStatus: csrv1beta1.CertificateSigningRequestStatus{
				Certificate: cert2,
			},
		},
	}

	server.handleRequestUpdate(hcmjoin, nil)
	_, cert3 := server.KeyCert()
	if !reflect.DeepEqual(cert2, cert3) {
		t.Error("cert is supposed to be updated")
	}
}
