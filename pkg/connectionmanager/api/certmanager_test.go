// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package api

import (
	"context"
	"crypto"
	"crypto/rand"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	hcmv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmclientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	hcmfake "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/common"
	csrv1beta1 "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	certutil "k8s.io/client-go/util/cert"
)

type fakeStore struct {
	server string
	key    []byte
	cert   []byte
}

func newFakeStore(server string, key, cert []byte) Store {
	return &fakeStore{
		server: server,
		key:    key,
		cert:   cert,
	}
}

func (s *fakeStore) Current() (*clientcmdapi.Config, error) {
	config := common.NewClientConfig(s.server, s.cert, s.key)

	clientConfigObj, err := runtime.Decode(clientcmdlatest.Codec, config)
	if err != nil {
		return nil, err
	}

	clientConfig, ok := clientConfigObj.(*clientcmdapi.Config)
	if !ok {
		return nil, fmt.Errorf("wrong type of client config data")
	}

	return clientConfig, nil
}

// Update updates current client config
func (s *fakeStore) Update(config []byte) (bool, error) {
	clientConfigObj, err := runtime.Decode(clientcmdlatest.Codec, config)
	if err != nil {
		return false, err
	}

	clientConfig, ok := clientConfigObj.(*clientcmdapi.Config)
	if !ok {
		return false, fmt.Errorf("wrong type of client config data")
	}

	clusterInfos, ok := clientConfig.Clusters["default-cluster"]
	if !ok {
		return false, fmt.Errorf("unable to get cluster infos")
	}

	authInfos, ok := clientConfig.AuthInfos["default-auth"]
	if !ok {
		return false, fmt.Errorf("unable to get authinfos")
	}

	s.server = clusterInfos.Server
	s.cert = authInfos.ClientCertificateData
	s.key = authInfos.ClientKeyData

	return true, nil
}

func newCertKey(duration time.Duration) (*rsa.PrivateKey, *x509.Certificate, []byte, []byte, error) {
	caKey, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	caCert, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "test.com"}, caKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	key, err := certutil.NewPrivateKey()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	cert, err := newSignedCert(key, caCert, caKey, duration)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return caKey, caCert, certutil.EncodePrivateKeyPEM(key), certutil.EncodeCertPEM(cert), nil
}

func newSignedCert(key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, duration time.Duration) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName: "hcm",
		},
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// Sign signs a certificate request, and returns a DER encoded x509 certificate.
func sign(crDER []byte, caCert *x509.Certificate, caKey crypto.Signer, duration time.Duration) ([]byte, error) {
	cr, err := x509.ParseCertificateRequest(crDER)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate request: %v", err)
	}
	if err := cr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("unable to verify certificate request signature: %v", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber:       serial,
		Subject:            cr.Subject,
		DNSNames:           cr.DNSNames,
		IPAddresses:        cr.IPAddresses,
		EmailAddresses:     cr.EmailAddresses,
		URIs:               cr.URIs,
		PublicKeyAlgorithm: cr.PublicKeyAlgorithm,
		PublicKey:          cr.PublicKey,
		Extensions:         cr.Extensions,
		ExtraExtensions:    cr.ExtraExtensions,
		NotBefore:          time.Now().UTC(),
		NotAfter:           time.Now().Add(duration).UTC(),
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, cr.PublicKey, caKey)
	if err != nil {
		return nil, fmt.Errorf("unable to sign certificate: %v", err)
	}

	return der, nil
}

func approveCJR(hcmClient hcmclientset.Interface, cjr *hcmv1alpha1.ClusterJoinRequest,
	caCert *x509.Certificate, caKey crypto.Signer, duration time.Duration) ([]byte, error) {
	block, _ := pem.Decode(cjr.Spec.CSR.Request)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, fmt.Errorf("pem block type must be CERTIFICATE REQUEST")
	}
	x509cr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse csr: %v", err)
	}

	der, err := sign(x509cr.Raw, caCert, caKey, duration)
	if err != nil {
		return nil, err
	}

	cjr.Status.Phase = hcmv1alpha1.JoinApproved
	cjr.Status.CSRStatus = csrv1beta1.CertificateSigningRequestStatus{
		Conditions: []csrv1beta1.CertificateSigningRequestCondition{
			{
				Type:           csrv1beta1.CertificateApproved,
				Reason:         "ClusterJoinApprove",
				Message:        "This CSR was approved by cluster join controller.",
				LastUpdateTime: metav1.Now(),
			},
		},
		Certificate: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}

	_, err = hcmClient.McmV1alpha1().ClusterJoinRequests().UpdateStatus(cjr)
	if err != nil {
		return nil, err
	}

	return cjr.Status.CSRStatus.Certificate, nil
}

func TestRotateCert(t *testing.T) {
	// create a cert which expires in 3 seconds
	caKey, caCert, key, cert, err := newCertKey(3 * time.Second)
	if err != nil {
		t.Errorf("Unable to create key and cert: %v", err)
	}

	store := newFakeStore("localhost", key, cert)

	hcmClient := hcmfake.NewSimpleClientset()
	manager := NewCertManager("c1", "c1", store, func() hcmclientset.Interface {
		return hcmClient
	})

	manager.Start()
	defer manager.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// approve cluster join request once it is created
	var newCert []byte
	err = wait.PollImmediateUntil(1*time.Second, func() (bool, error) {
		cjrs, err := hcmClient.Mcm().ClusterJoinRequests().List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}

		if len(cjrs.Items) == 0 {
			return false, nil
		}

		value, ok := cjrs.Items[0].Annotations[common.RenewalAnnotation]
		if !ok {
			return false, fmt.Errorf("no renwal annotation found on cluster join request: %v", cjrs.Items[0].Name)
		}

		if value != "true" {
			return false, fmt.Errorf("invalid value of renwal annotation on cluster join request: %v, %v", value, cjrs.Items[0].Name)
		}

		newCert, err = approveCJR(hcmClient, &cjrs.Items[0], caCert, caKey, 1*time.Hour)
		if err != nil {
			return false, err
		}

		return true, nil
	}, ctx.Done())

	if err != nil {
		t.Errorf("Unable to approve cluster join request: %v", err)
	}

	// validate if cert is update or not
	err = wait.PollImmediateUntil(1*time.Second, func() (bool, error) {
		store, ok := store.(*fakeStore)
		if !ok {
			return false, fmt.Errorf("invalid store type")
		}

		if reflect.DeepEqual(store.cert, newCert) {
			return true, nil
		}

		return false, nil
	}, ctx.Done())

	if err != nil {
		t.Errorf("Unable to rotate cert: %v", err)
	}
}
