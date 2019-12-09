// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterbootstrap

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	v1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmfake "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	csrv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	certutil "k8s.io/client-go/util/cert"
)

func newCertKey() ([]byte, []byte, error) {
	signingKey, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	signingCert, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "test.com"}, signingKey)
	if err != nil {
		return nil, nil, err
	}

	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	cert, err := certutil.NewSignedCert(
		certutil.Config{
			CommonName: "hcm",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		return nil, nil, err
	}

	return certutil.EncodePrivateKeyPEM(key), certutil.EncodeCertPEM(cert), nil
}

func newRequestName(key []byte) string {
	subject := &pkix.Name{
		Organization: []string{"hcm:clusters"},
		CommonName:   "hcm:clusters:test:test",
	}

	return DigestedName(key, subject)
}

func newTestBootStrap(request *v1alpha1.ClusterJoinRequest) *BootStrapper {
	fakehcmClient := hcmfake.NewSimpleClientset(request)

	return NewBootStrapper(fakehcmClient, "localhost", "test", "test", nil, nil)
}

func TestBootStrapWithNoSecret(t *testing.T) {
	key, cert, err := newCertKey()
	if err != nil {
		t.Errorf("Failed to generate client key/cert: %v", err)
	}

	hcmjoinName := newRequestName(key)
	hcmjoin := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: hcmjoinName,
		},
		Status: v1alpha1.ClusterJoinStatus{
			Phase: v1alpha1.JoinApproved,
			CSRStatus: csrv1beta1.CertificateSigningRequestStatus{
				Certificate: cert,
			},
		},
	}

	bootstrapper := newTestBootStrap(hcmjoin)
	clientKey, clientCert, configData, err := bootstrapper.LoadClientCert()
	if err != nil {
		t.Errorf("Failed to load config: %v", err)
	}
	if clientKey == nil || clientCert == nil || configData == nil {
		t.Errorf("Failed to load config")
	}
}

func TestBootStrapWithSecret(t *testing.T) {
	key, cert, err := newCertKey()
	if err != nil {
		t.Errorf("Failed to generate client key/cert: %v", err)
	}

	hcmjoinName := newRequestName(key)
	hcmjoin := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: hcmjoinName,
		},
		Status: v1alpha1.ClusterJoinStatus{
			Phase: v1alpha1.JoinApproved,
			CSRStatus: csrv1beta1.CertificateSigningRequestStatus{
				Certificate: cert,
			},
		},
	}

	bootstrapper := newTestBootStrap(hcmjoin)
	clientKey, clientCert, configData, err := bootstrapper.LoadClientCert()
	if err != nil {
		t.Errorf("Failed to load config: %v", err)
	}
	if clientKey == nil || clientCert == nil || configData == nil {
		t.Errorf("Failed to load config")
	}
}

func TestBootStrapWithDeny(t *testing.T) {
	key, _, err := newCertKey()
	if err != nil {
		t.Errorf("Failed to generate client key/cert: %v", err)
	}

	hcmjoinName := newRequestName(key)
	hcmjoin := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: hcmjoinName,
		},
		Status: v1alpha1.ClusterJoinStatus{
			Phase: v1alpha1.JoinDenied,
		},
	}

	bootstrapper := newTestBootStrap(hcmjoin)
	clientKey, clientCert, configData, err := bootstrapper.LoadClientCert()
	if err == nil {
		t.Errorf("Request is not denied: %v", err)
	}
	if clientKey != nil || clientCert != nil || configData != nil {
		t.Errorf("Request is not denied")
	}
}

func TestDigestNames(t *testing.T) {
	key1, err := certutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		t.Errorf("Failed to generate key: %v", err)
	}

	key2, err := certutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		t.Errorf("Failed to generate key: %v", err)
	}

	subjects := &pkix.Name{
		Organization: []string{"test"},
		CommonName:   "test",
	}

	name1 := DigestedName(key1, subjects)
	name2 := DigestedName(key1, subjects)
	if name1 != name2 {
		t.Errorf("name with the same key should be same")
	}

	name2 = DigestedName(key2, subjects)
	if name1 == name2 {
		t.Errorf("name with the different key should be different")
	}
}
