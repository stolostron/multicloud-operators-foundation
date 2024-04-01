package clusterinfo

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/onsi/gomega"
	clusterinfov1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cliScheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	cfg    *rest.Config
	scheme = runtime.NewScheme()
	c      client.Client
)

const (
	ManagedClusterName = "foo"
)

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "foundation", "hub", "crds")},
	}

	clusterinfov1beta1.AddToScheme(cliScheme.Scheme)
	clusterv1.AddToScheme(cliScheme.Scheme)
	corev1.AddToScheme(cliScheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		klog.Errorf("Failed to start, %v", err)
		os.Exit(1)
	}

	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, clusterv1.Install, clusterinfov1beta1.AddToScheme, corev1.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := clusterinfov1beta1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding cluster info to scheme, %v", err)
		os.Exit(1)
	}
	if err := clusterv1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
		os.Exit(1)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding core v1 to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

// StartTestManager adds recFn
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (context.CancelFunc, *sync.WaitGroup) {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		g.Expect(mgr.Start(ctx)).NotTo(gomega.HaveOccurred())
	}()

	return cancel, wg
}

func TestControllerReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{
		Metrics: metricserver.Options{BindAddress: "0"},
	})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	SetupWithManager(mgr, "open-cluster-management/ocm-klusterlet-self-signed-secrets")

	cancel, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		cancel()
		mgrStopped.Wait()
	}()

	time.Sleep(time.Second * 1)
}

func validateError(t *testing.T, err, expectedErrorType error) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}
}

func newTestClusterInfoReconciler(existingObjs []runtime.Object) *ClusterInfoReconciler {
	return &ClusterInfoReconciler{
		client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existingObjs...).Build(),
		scheme: scheme,
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name              string
		existingObjs      []runtime.Object
		expectedErrorType error
		req               reconcile.Request
		requeue           bool
	}{
		{
			name:         "ManagedClusterNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
		{
			name: "TerminatingManagedClusterHasFinalizerWithoutClusterInfo",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clusterFinalizerName,
						},
					},
					Spec: clusterv1.ManagedClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
		{
			name: "ManagedClusterNoFinalizerWithoutClusterInfo",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Finalizers: []string{
							clusterFinalizerName,
						},
					},
					Spec: clusterv1.ManagedClusterSpec{
						ManagedClusterClientConfigs: []clusterv1.ClientConfig{
							{
								URL: "",
							},
						},
						HubAcceptsClient:     false,
						LeaseDurationSeconds: 0,
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
			requeue: false,
		},
		{
			name: "ManagedClusterNoFinalizerWithClusterInfo",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
					},
					Spec: clusterv1.ManagedClusterSpec{
						ManagedClusterClientConfigs: []clusterv1.ClientConfig{
							{
								URL: "",
							},
						},
						HubAcceptsClient:     false,
						LeaseDurationSeconds: 0,
					},
					Status: clusterv1.ManagedClusterStatus{
						Conditions: []metav1.Condition{
							{
								Type:   clusterv1.ManagedClusterConditionAvailable,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&clusterinfov1beta1.ManagedClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ManagedClusterName,
						Namespace: ManagedClusterName,
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc := newTestClusterInfoReconciler(test.existingObjs)
			res, err := svrc.Reconcile(ctx, test.req)
			validateError(t, err, test.expectedErrorType)

			if test.requeue {
				assert.Equal(t, res.Requeue, true)
			} else {
				assert.Equal(t, res.Requeue, false)
			}
		})
	}
}

func Test_getLogCA(t *testing.T) {
	tests := []struct {
		name                   string
		logCertSecretNamespace string
		logCertSecretName      string
		existingObjs           []runtime.Object
		expectError            bool
		expectCAData           []byte
	}{
		{
			name:                   "get log ca",
			logCertSecretNamespace: "open-cluster-management",
			logCertSecretName:      "open-cluster-management",
			existingObjs: []runtime.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "open-cluster-management", Namespace: "open-cluster-management"},
				Data: map[string][]byte{
					"ca.crt": {123},
				},
			}},
			expectCAData: []byte{123},
		},
		{
			name:                   "no cert secret",
			logCertSecretNamespace: "open-cluster-management",
			logCertSecretName:      "open-cluster-management",
			existingObjs:           []runtime.Object{},
			expectError:            true,
			expectCAData:           nil,
		},
		{
			name:         "no cert secret is sepecified",
			existingObjs: []runtime.Object{},
			expectCAData: nil,
		},
		{
			name:                   "no ca data",
			logCertSecretNamespace: "open-cluster-management",
			logCertSecretName:      "open-cluster-management",
			existingObjs: []runtime.Object{&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "open-cluster-management", Namespace: "open-cluster-management"},
				Data:       nil,
			}},
			expectCAData: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logCertSecretNamespace = test.logCertSecretNamespace
			logCertSecretName = test.logCertSecretName
			svrc := newTestClusterInfoReconciler(test.existingObjs)
			caData, err := svrc.getLogCA()
			if !test.expectError && err != nil {
				t.Errorf("unexpected error %v", err)
			}

			assert.Equal(t, test.expectCAData, caData)
		})
	}
}
