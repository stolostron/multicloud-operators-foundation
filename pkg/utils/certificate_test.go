package utils

import (
	"crypto/x509"
	"fmt"
	"os"
	"testing"

	certutil "k8s.io/client-go/util/cert"
)

func TestWriteKeyCertToFile(t *testing.T) {
	certDir := "/tmp/tmp-cert"
	defer os.RemoveAll(certDir)

	_, err := WriteKeyCertToFile(certDir, []byte("aaa"), []byte("aaa"))
	if err == nil {
		t.Errorf("Write key/cert file shoud fail")
	}

	cert, key, err := certutil.GenerateSelfSignedCertKey("test.com", nil, nil)
	if err != nil {
		t.Errorf("Failed to generate key: %v", err)
	}

	path, err := WriteKeyCertToFile(certDir, key, cert)
	if err != nil {
		t.Errorf("Faile to write key/cert: %v", err)
	}

	fmt.Printf("%v", path)
}

func TestGeneratePemFile(t *testing.T) {
	certDir := "/tmp/tmp-cert"
	certFile := certDir + "/cert"
	keyFile := certDir + "/key"
	defer os.RemoveAll(certDir)

	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		direrr := os.MkdirAll(certDir, 0750)
		if direrr != nil {
			t.Errorf("Failed to generate cert dir: %v", direrr)
		}
	}

	cert, key, err := certutil.GenerateSelfSignedCertKey("test.com", nil, nil)
	if err != nil {
		t.Errorf("Failed to generate key: %v", err)
	}

	os.WriteFile(certFile, cert, 0777)
	os.WriteFile(keyFile, key, 0777)

	_, err = GeneratePemFile(certDir, "xxx", keyFile)
	if err == nil {
		t.Errorf("Generate pem file should fail")
	}
	_, err = GeneratePemFile(certDir, certFile, "xxx")
	if err == nil {
		t.Errorf("Generate pem file should fail")
	}
	curFilePath, err := GeneratePemFile(certDir, certFile, keyFile)
	if err != nil {
		t.Errorf("Faile to generate pem file: %v", err)
	}

	fmt.Printf("%v", curFilePath)
}

func TestNewSignedCert(t *testing.T) {
	key, err := NewPrivateKey()
	if err != nil {
		t.Errorf("Faile to NewPrivateKey: %v", err)
	}

	caKey, err := NewPrivateKey()
	if err != nil {
		t.Errorf("Faile to NewPrivateKey: %v", err)
	}

	config := certutil.Config{CommonName: "CommonName", Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}
	caCert, err := certutil.NewSelfSignedCACert(config, caKey)
	if err != nil {
		t.Errorf("Faile to NewSelfSignedCACert: %v", err)
	}

	config.Usages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	_, err = NewSignedCert(config, key, caCert, caKey)
	if err != nil {
		t.Errorf("Faile to NewPrivateKey: %v", err)
	}

}

func TestNewPrivateKey(t *testing.T) {
	_, err := NewPrivateKey()
	if err != nil {
		t.Errorf("Faile to NewPrivateKey: %v", err)
	}
}

func TestEncodeCertPEM(t *testing.T) {
	cert := &x509.Certificate{}
	EncodeCertPEM(cert)
}

func TestEncodePrivateKeyPEM(t *testing.T) {
	key, _ := NewPrivateKey()
	EncodePrivateKeyPEM(key)
}
