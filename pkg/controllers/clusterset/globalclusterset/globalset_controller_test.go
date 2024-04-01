package globalclusterset

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"

	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
)

var (
	scheme = runtime.NewScheme()
	cfg    *rest.Config
	c      client.Client
)

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "foundation", "hub", "crds")},
	}
	var err error
	if cfg, err = t.Start(); err != nil {
		klog.Errorf("Failed to start, %v", err)
		os.Exit(1)
	}
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, clusterv1beta1.Install, clusterv1beta2.Install, clusterv1.Install)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := hivev1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding hive to scheme, %v", err)
		os.Exit(1)
	}

	if err := rbacv1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding hive to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func newTestReconciler(existingObjs, existingKubeOjb []runtime.Object) *Reconciler {
	return &Reconciler{
		client:     fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existingObjs...).Build(),
		scheme:     scheme,
		kubeClient: k8sfake.NewSimpleClientset(existingKubeOjb...),
	}
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

	kubeClient := k8sfake.NewSimpleClientset()
	SetupWithManager(mgr, kubeClient)

	cancel, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		cancel()
		mgrStopped.Wait()
	}()

	time.Sleep(time.Second * 1)
}

func TestApplyGlobalResources(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	tests := []struct {
		name             string
		existingObjs     []runtime.Object
		existingKubeObjs []runtime.Object
		resourcesExist   bool
	}{
		{
			name: "global set exists",
			existingObjs: []runtime.Object{
				&clusterv1beta2.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
					},
					Spec: clusterv1beta2.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta2.ManagedClusterSelector{
							SelectorType:  clusterv1beta2.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			resourcesExist: true,
		},
		{
			name: "namespace exists",
			existingObjs: []runtime.Object{
				&clusterv1beta2.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
						Annotations: map[string]string{
							globalNamespaceAnnotation: "true",
						},
					},
					Spec: clusterv1beta2.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta2.ManagedClusterSelector{
							SelectorType:  clusterv1beta2.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			existingKubeObjs: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetNameSpace,
					},
				},
			},
			resourcesExist: true,
		},
		{
			name: "deleting Global set",
			existingObjs: []runtime.Object{
				&clusterv1beta2.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clustersetutils.ClustersetRoleFinalizerName,
						},
					},
					Spec: clusterv1beta2.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta2.ManagedClusterSelector{
							SelectorType:  clusterv1beta2.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			resourcesExist: false,
		},
		{
			name: "global set has annotation",
			existingObjs: []runtime.Object{
				&clusterv1beta2.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
						Annotations: map[string]string{
							globalNamespaceAnnotation: "true",
						},
					},
					Spec: clusterv1beta2.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta2.ManagedClusterSelector{
							SelectorType:  clusterv1beta2.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			resourcesExist: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newTestReconciler(test.existingObjs, test.existingKubeObjs)
			_, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: clustersetutils.GlobalSetName,
				},
			})
			g.Expect(err).NotTo(gomega.HaveOccurred())

			globalNsExist := true
			setBindingExist := true
			globalPlacementExist := true

			_, err = r.kubeClient.CoreV1().Namespaces().Get(
				context.TODO(), clustersetutils.GlobalSetNameSpace, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get global namespace, err: %v", err)
				}
				globalNsExist = false
			}
			if globalNsExist != test.resourcesExist {
				t.Errorf("Expect global namespace exist: %v, but get: %v", test.resourcesExist, globalNsExist)
			}

			globalSetBinding := &clusterv1beta2.ManagedClusterSetBinding{}
			err = r.client.Get(context.TODO(),
				types.NamespacedName{
					Name:      clustersetutils.GlobalSetName,
					Namespace: clustersetutils.GlobalSetNameSpace},
				globalSetBinding)
			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get clustersetbinding, err:%v", err)
					return
				}
				setBindingExist = false
			}
			if setBindingExist != test.resourcesExist {
				t.Errorf("Expect setbinding exist: %v, but get: %v", test.resourcesExist, setBindingExist)
			}

			globalPlacement := &clusterv1beta1.Placement{}
			err = r.client.Get(context.TODO(),
				types.NamespacedName{
					Name:      clustersetutils.GlobalPlacementName,
					Namespace: clustersetutils.GlobalSetNameSpace},
				globalPlacement)
			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get global placement, err: %v", err)
				}
				globalPlacementExist = false
			} else {
				if len(globalPlacement.Spec.Tolerations) != 2 {
					t.Errorf("Expect global placement has 2 tolerations, but get: %v",
						len(globalPlacement.Spec.Tolerations))
				}
				if globalPlacement.Spec.Tolerations[0].Key != clusterv1.ManagedClusterTaintUnreachable {
					t.Errorf("Expect global placement has taint %s, but get: %s",
						clusterv1.ManagedClusterTaintUnreachable, globalPlacement.Spec.Tolerations[0].Key)
				}
				if globalPlacement.Spec.Tolerations[1].Key != clusterv1.ManagedClusterTaintUnavailable {
					t.Errorf("Expect global placement has taint %s, but get: %s",
						clusterv1.ManagedClusterTaintUnavailable, globalPlacement.Spec.Tolerations[1].Key)
				}
			}
			if globalPlacementExist != test.resourcesExist {
				t.Errorf("Expect global placement exist: %v, but get: %v", test.resourcesExist, globalPlacementExist)
			}
		})
	}
}
