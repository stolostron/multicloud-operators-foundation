// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	hcmclientset "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/clusterbootstrap"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog"
)

// Store is an interface to get/set client config
type Store interface {
	// Current returns the current client config with a client certificate
	Current() (*clientcmdapi.Config, error)
	// Update updates the current client config
	Update(config []byte) (bool, error)
}

// CertManager is able to rotate client certitication in client config before it expires
type CertManager interface {
	// Start starts cert rotation
	Start()
	// Stop stop cert rotation
	Stop()
	// Restart restarts cert rotation
	Restart()
}

// HcmClietFunc return the current hcmclient. Return nil if no hcmclient is available
type HcmClietFunc func() hcmclientset.Interface

type certManager struct {
	clusterNamespace  string
	clusterName       string
	hcmClietFunc      HcmClietFunc
	clientConfigStore Store
	stopRotation      context.CancelFunc
}

// NewCertManager creates a instance of CertManager
func NewCertManager(clusterNamespace, clusterName string, clientConfigStore Store, hcmClietFunc HcmClietFunc) CertManager {
	return &certManager{
		clusterNamespace:  clusterNamespace,
		clusterName:       clusterName,
		clientConfigStore: clientConfigStore,
		hcmClietFunc:      hcmClietFunc,
	}
}

func (m *certManager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.stopRotation = cancel

	go wait.Until(func() {
		clientConfig, err := m.clientConfigStore.Current()
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("unable to get current client config: %v", err))
			return
		}

		leaf, err := getCertLeafFromClientConfig(clientConfig)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("unable to get certificate leaf: %v", err))
			return
		}

		deadline := m.nextRotationDeadline(leaf)

		if sleepInterval := time.Until(deadline); sleepInterval > 0 {
			klog.Infof("Wait %v for next certificate rotation", sleepInterval.Round(time.Second))
			timer := time.NewTimer(sleepInterval)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				klog.Infof("Next certificate rotation is aborted")
				//return immediately once context is cancelled
				return
			case <-timer.C:
				// unblock when deadline expires
			}
		}

		var key []byte
		err = wait.PollImmediateUntil(30*time.Second, func() (bool, error) {
			server, err := getServerFromClientConfig(clientConfig)
			if err != nil {
				utilruntime.HandleError(err)
				return false, nil
			}

			// create a new private key
			if key == nil {
				key, err = certutil.MakeEllipticPrivateKeyPEM()
				if err != nil {
					utilruntime.HandleError(fmt.Errorf("unable to generate a private key: %v", err))
				}
			}

			if err = m.rotateCerts(ctx, server, key); err != nil {
				if ctx.Err() == nil {
					utilruntime.HandleError(fmt.Errorf("unable to rotate client certificate: %v", err))
				}
				return false, nil
			}

			return true, nil
		}, ctx.Done())

		if err == wait.ErrWaitTimeout {
			klog.Info("Certificate rotation is aborted")
		} else if err != nil {
			utilruntime.HandleError(fmt.Errorf("unable to rotate client certificate: %v", err))
		}
	}, time.Second, ctx.Done())
}

func (m *certManager) Restart() {
	m.Stop()
	m.Start()
}

func (m *certManager) Stop() {
	if m.stopRotation != nil {
		m.stopRotation()
	}
}

func (m *certManager) nextRotationDeadline(leaf *x509.Certificate) time.Time {
	notAfter := leaf.NotAfter
	totalDuration := float64(notAfter.Sub(leaf.NotBefore))
	deadline := leaf.NotBefore.Add(jitteryDuration(totalDuration))

	klog.V(4).Infof("Certificate expiration is %v, rotation deadline is %v", notAfter, deadline)
	return deadline
}

func (m *certManager) rotateCerts(ctx context.Context, server string, clientKeyData []byte) error {
	klog.Infof("Start rotating certificate")

	hcmClient := m.hcmClietFunc()
	if hcmClient == nil {
		return errors.New("HCM clientset is not initialized yet")
	}

	// renew cert on hub
	bootstrapper := clusterbootstrap.NewBootStrapper(hcmClient, server, m.clusterNamespace, m.clusterName, clientKeyData, nil)
	_, _, clientConfigData, err := bootstrapper.RenewClientCert(ctx)
	if err != nil {
		return fmt.Errorf("unable to renew client certificate: %v", err)
	}
	klog.Infof("A new client certificate is created and approved")

	// replace local cert with new one
	_, err = m.clientConfigStore.Update(clientConfigData)
	if err != nil {
		return fmt.Errorf("unable to update client certificate: %v", err)
	}

	klog.Infof("Client certificate is renewed")
	return nil
}

func jitteryDuration(totalDuration float64) time.Duration {
	return wait.Jitter(time.Duration(totalDuration), 0.2) - time.Duration(totalDuration*0.3)
}

func getCertLeafFromClientConfig(clientConfig *clientcmdapi.Config) (*x509.Certificate, error) {
	authInfos, ok := clientConfig.AuthInfos["default-auth"]
	if !ok {
		return nil, fmt.Errorf("unable to get authinfos from client config: default-auth")
	}

	keyPair, err := tls.X509KeyPair(authInfos.ClientCertificateData, authInfos.ClientKeyData)
	if err != nil {
		return nil, fmt.Errorf("unable to create x509 key pair: %v", err)
	}

	if keyPair.Certificate == nil || len(keyPair.Certificate) == 0 {
		return nil, fmt.Errorf("unable to get certificate from x509 key pair")
	}

	certs, err := x509.ParseCertificates(keyPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate data: %v", err)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("unable to get leaf from certificate")
	}

	return certs[0], nil
}

func getServerFromClientConfig(clientConfig *clientcmdapi.Config) (string, error) {
	clusterInfos, ok := clientConfig.Clusters["default-cluster"]
	if !ok {
		return "", fmt.Errorf("unable to get cluster infos from client config: default-cluster")
	}

	return clusterInfos.Server, nil
}
