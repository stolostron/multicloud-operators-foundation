package certrotation

import (
	"context"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

var (
	scheme                 = runtime.NewScheme()
	logCertSecretNamespace = "open-cluster-management"
	logCertSecretName      = "ocm-klusterlet-self-signed-secrets"
)

func newTestReconciler(existingObjs ...runtime.Object) *Reconciler {
	s := kubescheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion)
	client := fake.NewFakeClientWithScheme(s, existingObjs...)
	return &Reconciler{
		client:                 client,
		scheme:                 scheme,
		logCertSecretNamespace: logCertSecretNamespace,
		logCertSecretName:      logCertSecretName,
	}
}

func newTestCertSecret(signerName string, caLifetime time.Duration) *corev1.Secret {
	certSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: logCertSecretNamespace,
			Name:      logCertSecretName,
		},
	}

	newCertSecret(certSecret, signerName, caLifetime)
	return certSecret
}

func TestReconciler(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		name           string
		existingSecret *corev1.Secret
		expectedCreate bool
		expectedUpdate bool
	}{
		{
			name: "no cert secret, create new one",
			existingSecret: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Namespace: "abc",
				Name:      "abc"},
			},
			expectedCreate: true,
			expectedUpdate: false,
		},
		{
			name:           "has cert secret, not expire，  not create/update",
			existingSecret: newTestCertSecret(signerName, defaultValidity),
			expectedCreate: false,
			expectedUpdate: false,
		},
		{
			name:           "has cert secret, expire，update old one",
			existingSecret: newTestCertSecret(signerName, -time.Minute*5),
			expectedCreate: false,
			expectedUpdate: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newTestReconciler(test.existingSecret)
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "abc", Name: "abc"}})
			if err != nil {
				t.Errorf("unexpected error :%v", err)
			}

			assert.Equal(t, res, reconcile.Result{RequeueAfter: resyncInterval})

			secret := &corev1.Secret{}
			err = r.client.Get(ctx, types.NamespacedName{Name: logCertSecretName, Namespace: logCertSecretNamespace}, secret)
			if err != nil {
				t.Errorf("unexpected error :%v", err)
			}

			switch {
			case test.expectedCreate:
				if len(secret.Data) == 0 {
					t.Errorf("should create cert secret")
				}
			case test.expectedUpdate:
				if apiequality.Semantic.DeepEqual(secret.Data, test.existingSecret.Data) {
					t.Errorf("should update cert secret")
				}
			default:
				if !apiequality.Semantic.DeepEqual(secret.Data, test.existingSecret.Data) {
					t.Errorf("should not update cert secret")
				}
			}
		})
	}
}
