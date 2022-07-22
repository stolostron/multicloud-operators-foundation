package certificate

import (
	"crypto/x509"
	"reflect"
	"testing"
	"time"

	"github.com/openshift/library-go/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/cert"
)

func newCACert(singerName string, validity time.Duration) (*x509.Certificate, error) {
	ca, err := crypto.MakeSelfSignedCAConfigForDuration(singerName, validity)
	if err != nil {
		return nil, err
	}
	return ca.Certs[0], nil
}

func TestMergeCABundle(t *testing.T) {
	caCert1, err := newCACert("signer1", time.Hour*1)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}
	caCert2, err := newCACert("signer1", time.Hour*24)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	ca1Bytes, err := crypto.EncodeCertificates(caCert1)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}
	ca2Bytes, err := crypto.EncodeCertificates(caCert2)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}
	ca12Bytes, err := crypto.EncodeCertificates(caCert1, caCert2)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	testCases := []struct {
		name        string
		oldCA       []byte
		newCA       []byte
		expectedCA  []byte
		expectedErr bool
	}{
		{
			name:        "same CA",
			oldCA:       ca1Bytes,
			newCA:       ca1Bytes,
			expectedCA:  ca1Bytes,
			expectedErr: false,
		},
		{
			name:        "invalid oldCA",
			oldCA:       []byte("test"),
			newCA:       ca2Bytes,
			expectedCA:  ca2Bytes,
			expectedErr: true,
		},
		{
			name:        "invalid newCA",
			oldCA:       ca1Bytes,
			newCA:       []byte("test"),
			expectedCA:  ca1Bytes,
			expectedErr: true,
		},
		{
			name:        "merge CA",
			oldCA:       ca1Bytes,
			newCA:       ca2Bytes,
			expectedCA:  ca12Bytes,
			expectedErr: false,
		},
		{
			name:        "merge CA 2",
			oldCA:       ca12Bytes,
			newCA:       ca2Bytes,
			expectedCA:  ca12Bytes,
			expectedErr: false,
		},
	}

	for _, c := range testCases {

		t.Run(c.name, func(t *testing.T) {
			caBytes, err := MergeCABundle(c.oldCA, c.newCA)
			if err != nil && !c.expectedErr {
				t.Errorf("expect no error, got err %v", err)
			}
			if err == nil && c.expectedErr {
				t.Errorf("expect err %v, got no err", c.expectedErr)
			}

			assert.Equal(t, true, validateCA(c.expectedCA, caBytes))
		})
	}
}

func TestFilterExpiredCerts(t *testing.T) {
	caCert1, _ := newCACert("signer1", -1*time.Second)
	caCert2, _ := newCACert("signer1", 1*time.Hour)
	valid := FilterExpiredCerts(caCert1, caCert2)
	assert.Equal(t, 1, len(valid))
}

func validateCA(expected, got []byte) bool {
	expectedCerts, err := cert.ParseCertsPEM(expected)
	if err != nil {
		return false
	}

	gotCerts, err := cert.ParseCertsPEM(got)
	if err != nil {
		return false
	}

	if len(expectedCerts) != len(gotCerts) {
		return false
	}
	for i := range expectedCerts {
		found := false
		for j := range gotCerts {
			if reflect.DeepEqual(expectedCerts[i].Raw, gotCerts[j].Raw) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
