// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterbootstrap

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
	"time"

	v1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/keyutil"

	certutil "k8s.io/client-go/util/cert"
)

func newCertKey() ([]byte, []byte, error) {
	signingKey, err := utils.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	signingCert, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "test.com"}, signingKey)
	if err != nil {
		return nil, nil, err
	}

	key, err := utils.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	cert, err := utils.NewSignedCert(
		certutil.Config{
			CommonName: "hcm",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		return nil, nil, err
	}

	return utils.EncodePrivateKeyPEM(key), utils.EncodeCertPEM(cert), nil
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
		Status: v1alpha1.ClusterJoinRequestStatus{
			Phase:       v1alpha1.JoinPhaseApproved,
			Certificate: cert,
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
		Status: v1alpha1.ClusterJoinRequestStatus{
			Phase:       v1alpha1.JoinPhaseApproved,
			Certificate: cert,
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
		Status: v1alpha1.ClusterJoinRequestStatus{
			Phase: v1alpha1.JoinPhaseDenied,
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
	key1, err := keyutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		t.Errorf("Failed to generate key: %v", err)
	}

	key2, err := keyutil.MakeEllipticPrivateKeyPEM()
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

func TestCertRenewal(t *testing.T) {
	fakehcmClient := hcmfake.NewSimpleClientset()
	bootstrapper := NewBootStrapper(fakehcmClient, "localhost", "test", "test", nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bootstrapper.RenewClientCert(ctx)

	var request *v1alpha1.ClusterJoinRequest
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		cjrs, err := fakehcmClient.McmV1alpha1().ClusterJoinRequests().List(metav1.ListOptions{})
		if err == nil && len(cjrs.Items) > 0 {
			request = &cjrs.Items[0]
			break
		}
	}

	if request == nil {
		t.Errorf("CJR for cert renewal is not created in 10 seconds")
	}

	if request.Annotations == nil {
		t.Errorf("CJR for cert renewal has no annotation")
	}

	renewal, ok := request.Annotations[common.RenewalAnnotation]

	if !ok {
		t.Errorf("CJR for cert renewal has no annotation: %s", common.RenewalAnnotation)
	}

	if renewal != "true" {
		t.Errorf("CJR for cert renewal invalid annotation: %s:%s", common.RenewalAnnotation, renewal)
	}
}

func TestCancelCertRenewal(t *testing.T) {
	key, _, err := newCertKey()
	if err != nil {
		t.Errorf("Failed to generate client key/cert: %v", err)
	}

	hcmjoinName := newRequestName(key)
	hcmjoin := &v1alpha1.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: hcmjoinName,
		},
	}

	bootstrapper := newTestBootStrap(hcmjoin)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, _, configData, err := bootstrapper.RenewClientCert(ctx)
	if err == nil {
		t.Errorf("Failed to cancel bootstrap")
	}

	if err != wait.ErrWaitTimeout {
		t.Errorf("Failed to cancel bootstrap: %v", err)
	}

	if configData != nil {
		t.Errorf("Failed to cancel bootstrap")
	}
}
