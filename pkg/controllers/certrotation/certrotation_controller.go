package certrotation

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"github.com/openshift/library-go/pkg/crypto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/cert"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

const (
	resyncInterval = time.Minute * 10

	signerName      = "ocm-klusterlet-logger"
	defaultValidity = time.Hour * 24 * 365
)

type Reconciler struct {
	client                 client.Client
	scheme                 *runtime.Scheme
	logCertSecretNamespace string
	logCertSecretName      string
}

func SetupWithManager(mgr manager.Manager, certSecret string) error {
	namespace, secretName, err := cache.SplitMetaNamespaceKey(certSecret)
	if err != nil {
		return err
	}
	if namespace == "" {
		namespace, err = utils.GetComponentNamespace()
		if err != nil {
			return err
		}
	}

	if err := add(mgr, newReconciler(mgr, namespace, secretName)); err != nil {
		klog.Errorf("Failed to create cert rotation controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, namespace, secretName string) reconcile.Reconciler {

	return &Reconciler{
		client:                 mgr.GetClient(),
		scheme:                 mgr.GetScheme(),
		logCertSecretNamespace: namespace,
		logCertSecretName:      secretName,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("certrotation-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	certSecret := &corev1.Secret{}

	err := r.client.Get(ctx, types.NamespacedName{Name: r.logCertSecretName, Namespace: r.logCertSecretNamespace}, certSecret)
	switch {
	case errors.IsNotFound(err):
		certSecret.Name = r.logCertSecretName
		certSecret.Namespace = r.logCertSecretNamespace
		certSecret.Type = corev1.SecretTypeTLS
		certSecret.Data = map[string][]byte{}
		if err := newCertSecret(certSecret, signerName, defaultValidity); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: resyncInterval}, r.client.Create(ctx, certSecret)

	case err != nil:
		return ctrl.Result{}, err
	}

	if validateCert(certSecret) == nil {
		return ctrl.Result{RequeueAfter: resyncInterval}, nil
	}

	klog.Infof("require a new cert secret.reason: %v", err)
	if err := newCertSecret(certSecret, signerName, defaultValidity); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: resyncInterval}, r.client.Update(ctx, certSecret)
}

func validateCert(secret *corev1.Secret) error {
	if secret == nil {
		return fmt.Errorf("invalid secret")
	}
	caBundle := secret.Data["ca.crt"]
	certData := secret.Data["tls.crt"]

	caCert, err := decodeCert(caBundle)
	if err != nil {
		return fmt.Errorf("failed to decode ca cert %v", err)
	}

	clientCert, err := decodeCert(certData)
	if err != nil {
		return fmt.Errorf("failed to decode client cert %v", err)
	}

	if clientCert.Issuer.CommonName == caCert.Subject.CommonName {
		return nil
	}

	return fmt.Errorf("issuer %q not in ca bundle", clientCert.Issuer.CommonName)
}

func decodeCert(certData []byte) (*x509.Certificate, error) {
	if len(certData) == 0 {
		return nil, fmt.Errorf("missing cert data")
	}
	certificates, err := cert.ParseCertsPEM(certData)
	if err != nil {
		return nil, fmt.Errorf("bad certificate")
	}

	if len(certificates) == 0 {
		return nil, fmt.Errorf("missing certificate")
	}
	clientCert := certificates[0]
	if time.Now().After(clientCert.NotAfter) {
		return nil, fmt.Errorf("already expired")
	}
	maxWait := clientCert.NotAfter.Sub(clientCert.NotBefore) / 5
	latestTime := clientCert.NotAfter.Add(-maxWait)
	if time.Now().After(latestTime) {
		return nil, fmt.Errorf("expired in %6.3f seconds", clientCert.NotAfter.Sub(time.Now()).Seconds())
	}

	return clientCert, nil
}

func newCertSecret(secret *corev1.Secret, signerName string, caLifetime time.Duration) error {
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}

	ca, err := crypto.MakeSelfSignedCAConfigForDuration(signerName, caLifetime)
	if err != nil {
		return err
	}

	caCertBytes := &bytes.Buffer{}
	caKeyBytes := &bytes.Buffer{}
	if err := ca.WriteCertConfig(caCertBytes, caKeyBytes); err != nil {
		return err
	}
	secret.Data["ca.crt"] = caCertBytes.Bytes()

	signingCertKeyPair, err := crypto.GetCAFromBytes(caCertBytes.Bytes(), caKeyBytes.Bytes())
	if err != nil {
		return err
	}
	clientUser := &user.DefaultInfo{
		Name: signerName,
	}
	certConfig, err := signingCertKeyPair.MakeClientCertificateForDuration(clientUser, caLifetime)
	if err != nil {
		return err
	}
	certBytes := &bytes.Buffer{}
	keyBytes := &bytes.Buffer{}
	if err := certConfig.WriteCertConfig(certBytes, keyBytes); err != nil {
		return err
	}
	secret.Data["tls.crt"] = certBytes.Bytes()
	secret.Data["tls.key"] = keyBytes.Bytes()
	return nil
}
