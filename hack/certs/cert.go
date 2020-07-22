package main

import (
	"crypto"
	"crypto/rand"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"flag"

	"math"
	"math/big"
	"os"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog"
)

var path string

func init() {
	flag.StringVar(&path, "path", "", "path is the file path")
	flag.Parse()
}

func main() {
	var err error
	if path == "" {
		path, err = os.Getwd()
		if err != nil {
			klog.Fatal(err)
		}
	}

	dnsNames := []string{"acm-proxyserver", "acm-proxyserver.open-cluster-management", "acm-proxyserver.open-cluster-management.svc"}
	if err := NewCerts(path, "acm-apiserver", dnsNames); err != nil {
		klog.Fatal(err)
	}
	if err := NewCerts(path, "acm-agent", nil); err != nil {
		klog.Fatal(err)
	}
}

func NewCerts(path string, commonName string, dnsNames []string) error {
	cakey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return err
	}
	config := certutil.Config{
		CommonName:   commonName,
		Organization: []string{"OpenShift ACM"},
	}
	caCert, err := certutil.NewSelfSignedCACert(config, cakey)
	if err != nil {
		return err
	}

	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return err
	}

	cert, err := NewSignedCert(key, caCert, cakey, commonName, dnsNames)
	if err != nil {
		return err
	}
	caData := utils.EncodeCertPEM(caCert)
	keyData := utils.EncodePrivateKeyPEM(key)
	certData := utils.EncodeCertPEM(cert)

	if err := certutil.WriteCert(path+"/"+commonName+"-client.pem", []byte(base64.StdEncoding.EncodeToString(certData))); err != nil {
		return err
	}
	if err := certutil.WriteCert(path+"/"+commonName+"-ca.pem", []byte(base64.StdEncoding.EncodeToString(caData))); err != nil {
		return err
	}
	if err := keyutil.WriteKey(path+"/"+commonName+"-key.pem", []byte(base64.StdEncoding.EncodeToString(keyData))); err != nil {
		return err
	}
	return nil
}

func NewSignedCert(key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, commonName string, dnsNames []string) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"OpenShift ACM"},
		},
		DNSNames:     dnsNames,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(time.Hour * 24 * 365 * 10).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
