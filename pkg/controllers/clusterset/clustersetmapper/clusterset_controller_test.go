package clustersetmapper

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
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

const (
	ManagedClusterSetName = "foo"
	ManagedClusterName    = "c1"
)

func generateClustersetToClusters(ms map[string]sets.String) *helpers.ClusterSetMapper {
	clustersetToClusters := helpers.NewClusterSetMapper()
	for s, c := range ms {
		clustersetToClusters.UpdateClusterSetByObjects(s, c)
	}
	return clustersetToClusters
}

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
	SetupWithManager(mgr, kubeClient, nil, nil, nil)

	cancel, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		cancel()
		mgrStopped.Wait()
	}()

	time.Sleep(time.Second * 1)
}

func newTestReconciler(existingObjs, existingRoleOjb []runtime.Object, globalClusterSetClusterMapper, clusterSetClusterMapper, clusterSetNamespaceMapper *helpers.ClusterSetMapper) *Reconciler {
	return &Reconciler{
		client:                        fake.NewFakeClientWithScheme(scheme, existingObjs...),
		scheme:                        scheme,
		kubeClient:                    k8sfake.NewSimpleClientset(existingRoleOjb...),
		clusterSetClusterMapper:       clusterSetClusterMapper,
		clusterSetNamespaceMapper:     clusterSetNamespaceMapper,
		globalClusterSetClusterMapper: globalClusterSetClusterMapper,
	}
}

func newAdminRoleObjs() []runtime.Object {
	return []runtime.Object{
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: utils.GenerateClustersetClusterroleName(ManagedClusterSetName, "admin"),
			},
			Rules: nil,
		},
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	ms2 := map[string]sets.String{ManagedClusterSetName: sets.NewString(ManagedClusterName)}

	ctc1 := generateClustersetToClusters(ms2)

	tests := []struct {
		name                      string
		existingObjs              []runtime.Object
		existingRoleObjs          []runtime.Object
		clusterroleExist          bool
		req                       reconcile.Request
		clusterSetClusterMapper   *helpers.ClusterSetMapper
		clusterSetNamespaceMapper *helpers.ClusterSetMapper
		globalSetMapper           *helpers.ClusterSetMapper
		expectClusterSetCluster   map[string]sets.String
		expectClusterSetNamespace map[string]sets.String
	}{
		{
			name:         "ManagedClusterSetNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
			clusterroleExist:          false,
			clusterSetClusterMapper:   helpers.NewClusterSetMapper(),
			clusterSetNamespaceMapper: helpers.NewClusterSetMapper(),
			expectClusterSetCluster:   map[string]sets.String{},
			expectClusterSetNamespace: map[string]sets.String{},
		},
		{
			name: "ManagedClusterSetHasFinalizerWithoutClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterSetName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clustersetutils.ClustersetRoleFinalizerName,
						},
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType: clusterv1beta1.LegacyClusterSetLabel,
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
			clusterroleExist:          false,
			clusterSetClusterMapper:   ctc1,
			clusterSetNamespaceMapper: helpers.NewClusterSetMapper(),
			expectClusterSetCluster:   map[string]sets.String{},
			expectClusterSetNamespace: map[string]sets.String{},
		},
		{
			name: "ManagedClusterSetNoFinalizerWithoutClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterSetName,
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType: clusterv1beta1.LegacyClusterSetLabel,
						},
					},
				},
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						Labels: map[string]string{
							clusterv1beta1.ClusterSetLabel: ManagedClusterSetName,
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
			clusterroleExist:          true,
			clusterSetClusterMapper:   helpers.NewClusterSetMapper(),
			clusterSetNamespaceMapper: helpers.NewClusterSetMapper(),
			expectClusterSetCluster: map[string]sets.String{
				ManagedClusterSetName: sets.NewString(ManagedClusterName),
			},
			expectClusterSetNamespace: map[string]sets.String{},
		},
		{
			name: "ManagedClusterSetNoFinalizerWithClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterSetName,
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType: clusterv1beta1.LegacyClusterSetLabel,
						},
					},
				},
			},
			existingRoleObjs: newAdminRoleObjs(),
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
			clusterroleExist:          true,
			clusterSetClusterMapper:   helpers.NewClusterSetMapper(),
			clusterSetNamespaceMapper: helpers.NewClusterSetMapper(),
			expectClusterSetCluster:   map[string]sets.String{},
			expectClusterSetNamespace: map[string]sets.String{},
		},
		{
			name: "Sync global set",
			existingObjs: []runtime.Object{
				&clusterv1beta1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: "global",
					},
					Spec: clusterv1beta1.ManagedClusterSetSpec{
						ClusterSelector: clusterv1beta1.ManagedClusterSelector{
							SelectorType:  clusterv1beta1.LabelSelector,
							LabelSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "global",
				},
			},
			clusterroleExist:          false,
			clusterSetClusterMapper:   helpers.NewClusterSetMapper(),
			clusterSetNamespaceMapper: helpers.NewClusterSetMapper(),
			globalSetMapper:           helpers.NewClusterSetMapper(),
			expectClusterSetCluster:   map[string]sets.String{},
			expectClusterSetNamespace: map[string]sets.String{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := newTestReconciler(test.existingObjs, test.existingRoleObjs, test.globalSetMapper, test.clusterSetClusterMapper, test.clusterSetNamespaceMapper)
			r.Reconcile(ctx, test.req)

			clusterroleName := utils.GenerateClustersetClusterroleName(ManagedClusterSetName, "admin")
			clusterrole, _ := r.kubeClient.RbacV1().ClusterRoles().Get(context.TODO(), clusterroleName, metav1.GetOptions{})

			var clusterroleExist bool
			if clusterrole != nil {
				clusterroleExist = true
			}

			assert.Equal(t, test.clusterroleExist, clusterroleExist)

			if !reflect.DeepEqual(r.clusterSetClusterMapper.GetAllClusterSetToObjects(), test.expectClusterSetCluster) {
				t.Errorf("clusterSetClusterMapper is not equal. clusterSetCluster:%v, test.expectClusterSetCluster:%v", r.clusterSetClusterMapper.GetAllClusterSetToObjects(), test.expectClusterSetCluster)
			}

			if !reflect.DeepEqual(r.clusterSetNamespaceMapper.GetAllClusterSetToObjects(), test.expectClusterSetNamespace) {
				t.Errorf("clusterSetNamespaceMapper is not equal. clusterSetCluster:%v, test.expectClusterSetNamespace:%v", r.clusterSetNamespaceMapper.GetAllClusterSetToObjects(), test.expectClusterSetNamespace)
			}

		})
	}
}

func TestGetRequiredClusterSet(t *testing.T) {
	tests := []struct {
		name             string
		labels           map[string]string
		clusterSetMap    map[string]sets.String
		expectRequestLen int
	}{
		{
			name:             "No clusterset",
			labels:           map[string]string{},
			expectRequestLen: 0,
		},
		{
			name: "Only have set label",
			labels: map[string]string{
				clusterv1beta1.ClusterSetLabel: "set1",
			},
			expectRequestLen: 1,
		},
		{
			name: "Only have set label",
			labels: map[string]string{
				clusterv1beta1.ClusterSetLabel: "set1",
			},
			clusterSetMap:    map[string]sets.String{"set2": sets.NewString("t1-ns", "t1-ns1")},
			expectRequestLen: 2,
		},
	}

	for _, test := range tests {
		clustersetMapper := generateClustersetToClusters(test.clusterSetMap)
		t.Run(test.name, func(t *testing.T) {
			reqs := getRequiredClusterSet(test.labels, clustersetMapper, "t1-ns")
			if len(reqs) != test.expectRequestLen {
				t.Errorf("expect requests is not right. reqs: %v", reqs)
			}

		})
	}
}
