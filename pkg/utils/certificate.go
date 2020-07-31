package utils

import (
	"crypto"
	"crypto/rand"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	pathfilepath "path/filepath"
	"time"

	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate"
	"k8s.io/klog"
)

const (
	rsaKeySize             = 2048
	duration365d           = time.Hour * 24 * 365
	certificateBlockType   = "CERTIFICATE"
	rsaPrivateKeyBlockType = "RSA PRIVATE KEY"
)

// WriteKeyCertToFile write key/cert to a certain cert path
func WriteKeyCertToFile(certDir string, key, cert []byte) (string, error) {
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		direrr := os.MkdirAll(certDir, 0750)
		if direrr != nil {
			return "", direrr
		}
	}

	store, err := certificate.NewFileStore("mongo", certDir, "", "", "")
	if err != nil {
		return "", fmt.Errorf("unable to build mongo cert store")
	}

	if _, err := store.Update(cert, key); err != nil {
		return "", err
	}

	return store.CurrentPath(), nil
}

// GeneratePemFile generate a pem file that include key and cert
func GeneratePemFile(dir, certFile, keyFile string) (string, error) {
	cert, err := readAll(certFile)
	if err != nil {
		klog.Error("Could not read cert file: ", certFile)
		return "", err
	}
	key, err := readAll(keyFile)
	if err != nil {
		klog.Error("Could not read key file: ", keyFile)
		return "", err
	}
	curFilePath, err := WriteKeyCertToFile(dir, key, cert)
	if err != nil {
		klog.Error("Could not generate pem file: ", err)
	}
	return curFilePath, nil
}

// NewPrivateKey creates an RSA private key
func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg cert.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration365d).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// EncodePrivateKeyPEM returns PEM-encoded private key data
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  rsaPrivateKeyBlockType,
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}

func readAll(filePath string) ([]byte, error) {
	f, err := os.Open(pathfilepath.Clean(filePath))
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}
