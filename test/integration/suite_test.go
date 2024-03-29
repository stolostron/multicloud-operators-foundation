package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	clusterlisterv1 "open-cluster-management.io/api/client/cluster/listers/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/stolostron/multicloud-operators-foundation/pkg/webhook/validating"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                 *rest.Config
	k8sClient           client.Client
	kubeClient          kubernetes.Interface
	hiveClient          hiveclient.Interface
	clusterClient       clusterv1client.Interface
	testEnv             *envtest.Environment
	testCtx, testCancel = context.WithCancel(context.Background())
)

const (
	validatingWebhookPath = "/validating"
)

func strPtr(s string) *string { return &s }

func TestIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Integration suite")
}

const (
	eventuallyTimeout  = 300
	eventuallyInterval = 2
)

var _ = ginkgo.BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(ginkgo.GinkgoWriter)))

	ginkgo.By("bootstrapping test environment")
	failPolicy := admissionv1.Fail
	equivalentPolicy := admissionv1.Equivalent
	sideEffects := admissionv1.SideEffectClassNone
	webhookInstallOptions := envtest.WebhookInstallOptions{
		ValidatingWebhooks: []*admissionv1.ValidatingWebhookConfiguration{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ocm-validating-webhook",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "ValidatingWebhookConfiguration",
					APIVersion: "admissionregistration.k8s.io/v1",
				},
				Webhooks: []admissionv1.ValidatingWebhook{
					{
						Name:                    "ocm.validating.webhook.admission.open-cluster-management.io",
						AdmissionReviewVersions: []string{"v1"},
						ClientConfig: admissionv1.WebhookClientConfig{
							Service: &admissionv1.ServiceReference{
								Path: strPtr(validatingWebhookPath),
							},
						},
						FailurePolicy: &failPolicy,
						MatchPolicy:   &equivalentPolicy,
						Rules: []admissionv1.RuleWithOperations{
							{
								Rule: admissionv1.Rule{
									APIGroups:   []string{"hive.openshift.io"},
									APIVersions: []string{"v1"},
									Resources:   []string{"clusterpools"},
								},
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
									admissionv1.Update,
								},
							},
							{
								Rule: admissionv1.Rule{
									APIGroups:   []string{"hive.openshift.io"},
									APIVersions: []string{"v1"},
									Resources:   []string{"clusterdeployments"},
								},
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
									admissionv1.Update,
									admissionv1.Delete,
								},
							},
							{
								Rule: admissionv1.Rule{
									APIGroups:   []string{"cluster.open-cluster-management.io"},
									APIVersions: []string{"*"},
									Resources:   []string{"managedclustersets"},
								},
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
									admissionv1.Update,
								},
							},
							{
								Rule: admissionv1.Rule{
									APIGroups:   []string{"cluster.open-cluster-management.io"},
									APIVersions: []string{"*"},
									Resources:   []string{"managedclusters"},
								},
								Operations: []admissionv1.OperationType{
									admissionv1.Create,
									admissionv1.Update,
									admissionv1.Delete,
								},
							},
						},
						SideEffects: &sideEffects,
					},
				},
			},
		},
	}

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(".", "vendor", "open-cluster-management.io", "api", "cluster", "v1"),
			filepath.Join(".", "vendor", "open-cluster-management.io", "api", "cluster", "v1beta2"),
			filepath.Join(".", "deploy", "foundation", "hub", "crds"),
		},
		WebhookInstallOptions: webhookInstallOptions,
	}

	var err error
	cfg, err = testEnv.Start()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(cfg).ToNot(gomega.BeNil())

	kubeClient, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(kubeClient).ToNot(gomega.BeNil())

	hiveClient, err = hiveclient.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(hiveClient).ToNot(gomega.BeNil())

	clusterClient, err = clusterv1client.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(clusterClient).ToNot(gomega.BeNil())

	clusterInformerFactory := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	clusterInformer := clusterInformerFactory.Cluster().V1().ManagedClusters()

	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(k8sClient).ToNot(gomega.BeNil())

	go clusterInformerFactory.Start(testCtx.Done())
	if ok := cache.WaitForCacheSync(testCtx.Done(), clusterInformer.Informer().HasSynced); !ok {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	ginkgo.By("running webhook server")
	go func() {
		err = runWebhookServer(testCtx, cfg, scheme, &testEnv.WebhookInstallOptions, clusterInformer.Lister())
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}()

	d := &net.Dialer{Timeout: time.Second}
	gomega.Eventually(func() error {
		serverURL := fmt.Sprintf("%s:%d",
			testEnv.WebhookInstallOptions.LocalServingHost,
			testEnv.WebhookInstallOptions.LocalServingPort)
		conn, err := tls.DialWithDialer(d, "tcp", serverURL, &tls.Config{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}, "120s", "1s").Should(gomega.Succeed())

	ginkgo.By("Start to run tests")
})

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("tearing down the test environment")
	testCancel()
	err := testEnv.Stop()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

func runWebhookServer(ctx context.Context, cfg *rest.Config, scheme *runtime.Scheme,
	opts *envtest.WebhookInstallOptions, clusterLister clusterlisterv1.ManagedClusterLister) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	ginkgo.By(fmt.Sprintf("start server on %d", opts.LocalServingPort))
	wh := webhook.NewServer(webhook.Options{
		Host:    opts.LocalServingHost,
		Port:    opts.LocalServingPort,
		CertDir: opts.LocalServingCertDir,
	})

	// +kubebuilder:scaffold:builder
	wh.Register(validatingWebhookPath, ValidatingHandler(clusterLister))

	if err := wh.Start(ctx); err != nil {
		return err
	}
	return nil
}

func ValidatingHandler(
	clusterLister clusterlisterv1.ManagedClusterLister) http.Handler {
	validatingAh := &validating.AdmissionHandler{
		KubeClient:    kubeClient,
		HiveClient:    hiveClient,
		ClusterLister: clusterLister,
	}
	return &webhook.Admission{
		Handler: &validatingHandler{
			validatingAh: validatingAh,
		},
	}
}

type validatingHandler struct {
	validatingAh *validating.AdmissionHandler
}

func (m *validatingHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	response := m.validatingAh.ValidateResource(&req.AdmissionRequest)
	return admission.Response{AdmissionResponse: *response}
}
