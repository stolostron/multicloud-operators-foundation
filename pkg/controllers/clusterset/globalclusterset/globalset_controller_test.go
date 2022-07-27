package globalclusterset

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"k8s.io/client-go/rest"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme = runtime.NewScheme()
	cfg    *rest.Config
	c      client.Client
)

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "foundation", "hub", "resources", "crds")},
	}
	var err error
	if cfg, err = t.Start(); err != nil {
		klog.Errorf("Failed to start, %v", err)
	}
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, clusterv1beta1.Install, clusterv1.Install)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := clusterv1beta1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
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

func newTestReconciler(existingObjs, existingRoleOjb []runtime.Object) *Reconciler {
	return &Reconciler{
		client:     fake.NewFakeClientWithScheme(scheme, existingObjs...),
		scheme:     scheme,
		kubeClient: k8sfake.NewSimpleClientset(existingRoleOjb...),
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
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
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

func TestApplyGlobalNsAndSetBinding(t *testing.T) {
	tests := []struct {
		name                 string
		existingObjs         []runtime.Object
		setBindingAndNsExist bool
	}{
		{
			name: "Has Global set",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType:  clusterv1beta1.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			setBindingAndNsExist: true,
		},
		{
			name: "deleting Global set",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clustersetutils.ClustersetRoleFinalizerName,
						},
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType:  clusterv1beta1.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			setBindingAndNsExist: false,
		},
		{
			name: "global set has annotation",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: clustersetutils.GlobalSetName,
						Annotations: map[string]string{
							globalNamespaceAnnotation: "true",
						},
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType:  clusterv1beta1.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			setBindingAndNsExist: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newTestReconciler(test.existingObjs, nil)
			r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: clustersetutils.GlobalSetName,
				},
			})
			globalSetBinding := &clusterv1beta1.ManagedClusterSetBinding{}
			setBindingExist := true
			globalNsExist := true
			err := r.client.Get(context.TODO(), types.NamespacedName{Name: clustersetutils.GlobalSetName, Namespace: clustersetutils.GlobalSetNameSpace}, globalSetBinding)
			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get clustersetbinding, err:%v", err)
					return
				}
				setBindingExist = false
			}
			if setBindingExist != test.setBindingAndNsExist {
				t.Errorf("Expect setbinding exist:%v, but get:%v", test.setBindingAndNsExist, setBindingExist)
			}

			_, err = r.kubeClient.CoreV1().Namespaces().Get(context.TODO(), clustersetutils.GlobalSetNameSpace, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get clustersetbinding, err:%v", err)
				}
				globalNsExist = false
			}
			if globalNsExist != test.setBindingAndNsExist {
				t.Errorf("Expect setbinding exist:%v, but get:%v", test.setBindingAndNsExist, globalNsExist)
			}
		})
	}
}
