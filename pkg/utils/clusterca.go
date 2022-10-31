package utils

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ApiserverConfigName      = "cluster"
	OpenshiftConfigNamespace = "openshift-config"
	ServiceAccountNamespace  = "kube-system"
	ServiceAccountName       = "default"
	infrastructureConfigName = "cluster"
	ConfigmapNamespace       = "kube-public"
	CrtConfigmapName         = "kube-root-ca.crt"
	ClusterinfoConfigmap     = "cluster-info"
)

func GetKubeAPIServerAddress(ctx context.Context, openshiftClient openshiftclientset.Interface) (string, error) {
	infraConfig, err := openshiftClient.ConfigV1().Infrastructures().Get(ctx, infrastructureConfigName, v1.GetOptions{})
	if err != nil {
		return "", err
	}
	return infraConfig.Status.APIServerURL, nil
}

// getKubeAPIServerSecretName iterate through all namespacedCertificates
// returns the first one which has a name matches the given dnsName
func getKubeAPIServerSecretName(ctx context.Context, ocpClient openshiftclientset.Interface, dnsName string) (string, error) {
	apiserver, err := ocpClient.ConfigV1().APIServers().Get(ctx, ApiserverConfigName, v1.GetOptions{})
	if err != nil {
		return "", err
	}

	// iterate through all namedcertificates
	for _, namedCert := range apiserver.Spec.ServingCerts.NamedCertificates {
		for _, name := range namedCert.Names {
			if strings.EqualFold(name, dnsName) {
				return namedCert.ServingCertificate.Name, nil
			}
		}
	}

	return "", fmt.Errorf("Failed to get ServingCerts match name: %v", dnsName)
}

// getKubeAPIServerCertificate looks for secret in openshift-config namespace, and returns tls.crt
func getKubeAPIServerCertificate(ctx context.Context, kubeClient kubernetes.Interface, secretName string) ([]byte, error) {
	secret, err := kubeClient.CoreV1().Secrets(OpenshiftConfigNamespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if secret.Type != corev1.SecretTypeTLS {
		return nil, fmt.Errorf(
			"secret %s/%s should have type=kubernetes.io/tls",
			OpenshiftConfigNamespace,
			secretName,
		)
	}
	res, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, fmt.Errorf(
			"failed to find data[tls.crt] in secret %s/%s",
			OpenshiftConfigNamespace,
			secretName,
		)
	}
	return res, nil
}

func GetCAFromApiserver(ctx context.Context, ocpClient openshiftclientset.Interface, kubeClient kubernetes.Interface, kubeAPIServer string) ([]byte, error) {
	u, err := url.Parse(kubeAPIServer)
	if err != nil {
		return []byte{}, err
	}
	apiServerCertSecretName, err := getKubeAPIServerSecretName(ctx, ocpClient, u.Hostname())
	if err != nil {
		return nil, err
	}

	apiServerCert, err := getKubeAPIServerCertificate(ctx, kubeClient, apiServerCertSecretName)
	if err != nil {
		return nil, err
	}
	return apiServerCert, nil
}

// GetCACert returns the CA cert. It searches in the kube-root-ca.crt configmap in kube-public ns.
func GetCAFromConfigMap(ctx context.Context, kubeClient kubernetes.Interface) ([]byte, error) {
	cm, err := kubeClient.CoreV1().ConfigMaps(ConfigmapNamespace).Get(ctx, CrtConfigmapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return []byte(cm.Data["ca.crt"]), nil
}

func GetCAFromServiceAccount(ctx context.Context, kubeClient kubernetes.Interface) ([]byte, error) {
	defaultsa, err := kubeClient.CoreV1().ServiceAccounts(ServiceAccountNamespace).Get(ctx, ServiceAccountName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	for _, objectRef := range defaultsa.Secrets {
		defaultSecret, err := kubeClient.CoreV1().Secrets(ServiceAccountNamespace).Get(ctx, objectRef.Name, v1.GetOptions{})
		if err != nil || defaultSecret.Type != corev1.SecretTypeServiceAccountToken || defaultSecret == nil {
			continue
		}
		if _, ok := defaultSecret.Data["ca.crt"]; ok {
			return defaultSecret.Data["ca.crt"], nil
		}
	}
	return nil, fmt.Errorf("secret with type %s not found in service account %s/%s",
		corev1.SecretTypeServiceAccountToken,
		ServiceAccountNamespace,
		ServiceAccountName)
}
