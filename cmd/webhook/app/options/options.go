// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"crypto/tls"
	"sync"
	"time"

	"github.com/spf13/pflag"
)

// Config contains the server (the webhook) cert and key.
type Options struct {
	CertFile       string
	KeyFile        string
	KubeConfigFile string
	QPS            float32
	Burst          int
}

// NewOptions constructs a new set of default options for webhook.
func NewOptions() *Options {
	return &Options{
		KubeConfigFile: "",
		CertFile:       "",
		KeyFile:        "",
		QPS:            100.0,
		Burst:          200,
	}
}

func (c *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.CertFile, "tls-cert-file", c.CertFile, ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")
	fs.StringVar(&c.KeyFile, "tls-private-key-file", c.KeyFile, ""+
		"File containing the default x509 private key matching --tls-cert-file.")
	fs.StringVar(&c.KubeConfigFile, "kube-config-file", c.KubeConfigFile, ""+
		"Kube configuration file")
	fs.Float32Var(&c.QPS, "max-qps", c.QPS,
		"Maximum QPS to the hub server from this webhook.")
	fs.IntVar(&c.Burst, "max-burst", c.Burst,
		"Maximum burst for throttle.")
}

type certificateCacheEntry struct {
	cert  *tls.Certificate
	err   error
	birth time.Time
}

// isStale returns true when this cache entry is too old to be usable
func (c *certificateCacheEntry) isStale() bool {
	return time.Since(c.birth) > time.Second
}

func newCertificateCacheEntry(certFile, keyFile string) certificateCacheEntry {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	return certificateCacheEntry{cert: &cert, err: err, birth: time.Now()}
}

// cachingCertificateLoader ensures that we don't hammer the filesystem when opening many connections
// the underlying cert files are read at most once every second
func cachingCertificateLoader(certFile, keyFile string) func() (*tls.Certificate, error) {
	current := newCertificateCacheEntry(certFile, keyFile)
	var currentMtx sync.RWMutex

	return func() (*tls.Certificate, error) {
		currentMtx.RLock()
		if current.isStale() {
			currentMtx.RUnlock()

			currentMtx.Lock()
			defer currentMtx.Unlock()

			if current.isStale() {
				current = newCertificateCacheEntry(certFile, keyFile)
			}
		} else {
			defer currentMtx.RUnlock()
		}

		return current.cert, current.err
	}
}

func ConfigTLS(o *Options) *tls.Config {
	dynamicCertLoader := cachingCertificateLoader(o.CertFile, o.KeyFile)
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return dynamicCertLoader()
		},
	}
}
