package certificate

import (
	"crypto/x509"
	"fmt"
	"reflect"
	"time"

	"github.com/openshift/library-go/pkg/crypto"
	"k8s.io/client-go/util/cert"
)

// MergeCABundle adds the new certificate to the list of CABundles, eliminates duplicates,
// and prunes the list of expired certs
func MergeCABundle(oldCABytes, newCABytes []byte) ([]byte, error) {
	if len(oldCABytes) == 0 {
		return newCABytes, nil
	}
	if reflect.DeepEqual(oldCABytes, newCABytes) {
		return newCABytes, nil
	}

	certificates, err := cert.ParseCertsPEM(oldCABytes)
	if err != nil {
		return newCABytes, fmt.Errorf("failed to parse old ca bundle. err:%v", err)
	}

	newCertificates, err := cert.ParseCertsPEM(newCABytes)
	if err != nil {
		return oldCABytes, fmt.Errorf("failed to parse new ca bundle. err:%v", err)
	}

	certificates = append(certificates, newCertificates...)
	certificates = FilterExpiredCerts(certificates...)

	var finalCertificates []*x509.Certificate
	// now check for duplicates. n^2, but super simple
	for i := range certificates {
		found := false
		for j := range finalCertificates {
			if reflect.DeepEqual(certificates[i].Raw, finalCertificates[j].Raw) {
				found = true
				break
			}
		}
		if !found {
			finalCertificates = append(finalCertificates, certificates[i])
		}
	}

	caBytes, err := crypto.EncodeCertificates(finalCertificates...)
	if err != nil {
		return newCABytes, fmt.Errorf("failed to encode merged certificates. err: %v", err)
	}

	return caBytes, nil
}

// FilterExpiredCerts checks are all certificates in the bundle valid, i.e. they have not expired.
// The function returns new bundle with only valid certificates or error if no valid certificate is found.
func FilterExpiredCerts(certs ...*x509.Certificate) []*x509.Certificate {
	currentTime := time.Now()
	var validCerts []*x509.Certificate
	for _, c := range certs {
		if c.NotAfter.After(currentTime) {
			validCerts = append(validCerts, c)
		}
	}

	return validCerts
}
