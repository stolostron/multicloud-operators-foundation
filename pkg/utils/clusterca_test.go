package utils

import (
	"context"
	"reflect"
	"testing"

	ocinfrav1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func Test_getKubeAPIServerAddress(t *testing.T) {
	ctx := context.Background()
	infraConfig := &ocinfrav1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: ocinfrav1.InfrastructureSpec{},
		Status: ocinfrav1.InfrastructureStatus{
			APIServerURL: "http://127.0.0.1:6443",
		},
	}

	type args struct {
		client openshiftclientset.Interface
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "no error",
			args: args{
				client: configfake.NewSimpleClientset(infraConfig),
			},
			want:    "http://127.0.0.1:6443",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetKubeAPIServerAddress(ctx, tt.args.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKubeAPIServerAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getKubeAPIServerAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getKubeAPIServerSecretName(t *testing.T) {
	ctx := context.Background()
	apiserverConfig := &ocinfrav1.APIServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: ocinfrav1.APIServerSpec{
			ServingCerts: ocinfrav1.APIServerServingCerts{
				NamedCertificates: []ocinfrav1.APIServerNamedServingCert{
					{
						Names:              []string{"my-dns-name.com"},
						ServingCertificate: ocinfrav1.SecretNameReference{Name: "my-secret-name"},
					},
				},
			},
		},
	}

	type args struct {
		client openshiftclientset.Interface
		name   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "no name matches",
			args: args{
				client: configfake.NewSimpleClientset(apiserverConfig),
				name:   "fake-name",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				client: configfake.NewSimpleClientset(apiserverConfig),
				name:   "my-dns-name.com",
			},
			want:    "my-secret-name",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getKubeAPIServerSecretName(ctx, tt.args.client, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKubeAPIServerSecretName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getKubeAPIServerSecretName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getKubeAPIServerCertificate(t *testing.T) {
	ctx := context.Background()
	secretCorrect := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "openshift-config",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("fake-cert-data"),
			"tls.key": []byte("fake-key-data"),
		},
		Type: corev1.SecretTypeTLS,
	}
	secretWrongType := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "openshift-config",
		},
		Data: map[string][]byte{
			"token": []byte("fake-token"),
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	secretNoData := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "openshift-config",
		},
		Data: map[string][]byte{},
		Type: corev1.SecretTypeTLS,
	}

	type args struct {
		kubeClient kubernetes.Interface
		name       string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "no secret",
			args: args{
				kubeClient: kubefake.NewSimpleClientset([]runtime.Object{}...),
				name:       "test-secret",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "wrong type",
			args: args{
				kubeClient: kubefake.NewSimpleClientset(secretWrongType),
				name:       "test-secret",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty data",
			args: args{
				kubeClient: kubefake.NewSimpleClientset(secretNoData),
				name:       "test-secret",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "success",
			args: args{
				kubeClient: kubefake.NewSimpleClientset(secretCorrect),
				name:       "test-secret",
			},
			want:    []byte("fake-cert-data"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getKubeAPIServerCertificate(ctx, tt.args.kubeClient, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("getKubeAPIServerCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getKubeAPIServerCertificate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetCAFromServiceAccount(t *testing.T) {
	ctx := context.Background()
	secretSameNamespace := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-token-xxxx",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"ca.crt": []byte("ca data"),
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "kube-system",
		},
		Secrets: []corev1.ObjectReference{
			{
				Name: "default-token-xxxx",
			},
		},
	}

	tests := []struct {
		name       string
		kubeClient kubernetes.Interface
		want       []byte
		wantErr    bool
	}{
		{
			name:       "right crt",
			kubeClient: kubefake.NewSimpleClientset(serviceAccount, secretSameNamespace),
			want:       []byte("ca data"),
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCAFromServiceAccount(ctx, tt.kubeClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCAFromServiceAccount error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCAFromServiceAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}
