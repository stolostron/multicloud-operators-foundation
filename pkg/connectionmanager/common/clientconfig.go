// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package common

import (
	"crypto/x509"
	"fmt"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	certutil "k8s.io/client-go/util/cert"
)

// GetHostCerKeyFromClientConfig return cert and key from client config
func GetHostCerKeyFromClientConfig(config []byte) (string, []byte, []byte, error) {
	clientConfigObj, err := runtime.Decode(clientcmdlatest.Codec, config)
	if err != nil {
		return "", nil, nil, err
	}

	clientConfig, ok := clientConfigObj.(*clientcmdapi.Config)
	if !ok {
		return "", nil, nil, fmt.Errorf("wrong type of client config data")
	}

	clusterInfos, ok := clientConfig.Clusters["default-cluster"]
	if !ok {
		return "", nil, nil, fmt.Errorf("cannot get cluster infos")
	}

	authInfos, ok := clientConfig.AuthInfos["default-auth"]
	if !ok {
		return "", nil, nil, fmt.Errorf("cannot get authinfos")
	}

	return clusterInfos.Server, authInfos.ClientKeyData, authInfos.ClientCertificateData, nil
}

// NewClientConfig returns a new client config
func NewClientConfig(host string, cert, key []byte) []byte {
	hcmconfig := clientcmdapi.Config{
		// Define a cluster stanza based on the bootstrap kubeconfig.
		Clusters: map[string]*clientcmdapi.Cluster{"default-cluster": {
			Server:                host,
			InsecureSkipTLSVerify: true,
		}},
		// Define auth based on the obtained client cert.
		AuthInfos: map[string]*clientcmdapi.AuthInfo{"default-auth": {
			ClientCertificateData: cert,
			ClientKeyData:         key,
		}},
		// Define a context that connects the auth info and cluster, and set it as the default
		Contexts: map[string]*clientcmdapi.Context{"default-context": {
			Cluster:   "default-cluster",
			AuthInfo:  "default-auth",
			Namespace: "default",
		}},
		CurrentContext: "default-context",
	}

	configData, _ := runtime.Encode(clientcmdlatest.Codec, &hcmconfig)
	return configData
}

// NewCertKey generate cert and key
func NewCertKey(caCommonName, certCommonName string) ([]byte, []byte, error) {
	signingKey, err := utils.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	signingCert, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: caCommonName}, signingKey)
	if err != nil {
		return nil, nil, err
	}

	key, err := utils.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	cert, err := utils.NewSignedCert(
		certutil.Config{
			CommonName: certCommonName,
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		return nil, nil, err
	}

	return utils.EncodePrivateKeyPEM(key), utils.EncodeCertPEM(cert), nil
}
