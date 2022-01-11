package agent

import (
	"crypto/tls"
	"encoding/base64"
	"testing"

	"github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var (
	kubeNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
	}

	kubeEndpoints = &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: "127.0.0.1",
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: 443,
					},
				},
			},
		},
	}

	kubeMonitoringService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}

	kubeMonitoringSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitoring",
			Namespace: "kube-system",
		},
		Data: map[string][]byte{
			"tls.crt": []byte("aaa"),
			"tls.key": []byte("aaa"),
		},
	}

	klusterletService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "klusterlet",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "10.0.0.1",
					},
				},
			},
		},
	}
)

func newFakeKlusterlet() *Agent {
	fakeKubeClient := kubefake.NewSimpleClientset(
		kubeNode, kubeEndpoints,
		kubeMonitoringService, kubeMonitoringSecret, klusterletService)
	return NewAgent("clusterName", fakeKubeClient)
}
func TestKlusterlet_ListenAndServe(t *testing.T) {
	fakeKlusterlet := newFakeKlusterlet()
	localaddress := []byte("127.0.0.1")

	tlsOps := &TLSOptions{
		Config: &tls.Config{},
	}
	str := "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURGekNDQWYrZ0F3SUJBZ0lSQVBwdXB" +
		"tTEpOL0pYQVJwOE9KNFJjcTR3RFFZSktvWklodmNOQVFFTEJRQXcKSlRFak1DRUdBMVVFQXhNYWJYVn" +
		"NkR2xqYkhWemRHVnlhSFZpTFd0c2RYTjBaWEpzWlhRd0hoY05NakF3TlRFMApNRGN6TVRFM1doY05Na" +
		"kV3TlRFME1EY3pNVEUzV2pBbE1TTXdJUVlEVlFRREV4cHRkV3gwYVdOc2RYTjBaWEpvCmRXSXRhMngxY" +
		"zNSbGNteGxkRENDQVNJd0RRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFNSFIKWVV3d" +
		"HpPYXVGKzBXYzh3em1Mdmlwdkl6MkhGWnlwYUNjeld5VmJtSmlZWTJCTzJsQk4ranUyS0IrV2FNckt1O" +
		"AppbWdNMU9WSWpqN0pleDB5NnlERE1BOW1EVlVFbFJzZUZuNGhxVjVzcVJraTkxMlo2WFZYNEZiQTY1M" +
		"GdsdDhjCitqWlUyeTdCUEpEeUdKSXJJQ0FzMmJSV1hWaHB6T012Y0p4UHFDV00zZjc0NUhNTEdwUXVpTE" +
		"pNTEswbkVnelQKbEloaWQrejJDNmJRTEdXVXh4YiszNjIzRFpSNXM3Z0x6bThVbkxXbFZYOHhJZXRLWFZ" +
		"oRisxUWFydVZaU0lmSwppWFg3elBkdFdJNFhzSms0dHEybjl3VGFCd3B3d09kK2tVcUNCdkhwUVlKWENpb" +
		"XEwUHlqRjBHR0tpNUQrUDVNCmcwVWpYUGFIQlRPZUxMTjNveHNDQXdFQUFhTkNNRUF3RGdZRFZSMFBBUUg" +
		"vQkFRREFnS2tNQjBHQTFVZEpRUVcKTUJRR0NDc0dBUVVGQndNQkJnZ3JCZ0VGQlFjREFqQVBCZ05WSFJNQ" +
		"kFmOEVCVEFEQVFIL01BMEdDU3FHU0liMwpEUUVCQ3dVQUE0SUJBUUI0WEd2YWhhYTlsUStSS3lrdGlyV2J" +
		"ZZVlSc1RuMTNWcW02VDNrSjFxQWp6YWx2UUNOCnAwVG1OTUhUcDd6OXlveVpwWmY3K2RLaWpQdkZqczQwN" +
		"0FVamZ4TzhKMlkveUd2aExPTzN3dStKZFp1TlJDYnAKM2YxVzZranV4SDVHdnpiOU0wMjFMTTA0RjJWMnF" +
		"IRTI2aFNlZFFUY3FQSURXYXlHS2RJQWZzc2d6TUM4RDN1Ngo1RzVvaWtETWRMb0I0V2doakxsc1NQOC9xR" +
		"lMxeGxjeHF0Mlp5bU9lL2lxWFkwbTAwQzRTVzB4UFpWbmN6ZEV4CjBHRWJmejkwWHpkVlQ1NTJhTTRYM09" +
		"QbUdWV3E2RTRRMFBxRFVOU2prY1JQN0VCRC9Jc2pkQ1lkcDB3cjVtNTMKUzdMM2JaUmRaU3V1VmhLb1pWek" +
		"1aWTdPaVpOdkVhQk5DdlhaCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

	data, _ := base64.StdEncoding.DecodeString(str)
	clusterinfo := v1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clusterinfo",
			Namespace: "kube-system",
		},
		Spec: v1beta1.ClusterInfoSpec{
			MasterEndpoint: "127.0.0.1",
			LoggingCA:      data,
		},
	}
	go func() {
		fakeKlusterlet.RunServer <- clusterinfo
	}()
	fakeKlusterlet.ListenAndServe(localaddress, 8080, tlsOps, nil, false)
	fakeKlusterlet.RefreshServerIfNeeded(&clusterinfo)
}
