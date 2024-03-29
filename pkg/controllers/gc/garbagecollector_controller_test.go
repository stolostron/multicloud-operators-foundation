package gc

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/onsi/gomega"
	actionv1beta1 "github.com/stolostron/cluster-lifecycle-api/action/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cliScheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	actionName       = "foo"
	clusterNamespace = "bar"
)

var (
	cfg    *rest.Config
	scheme = runtime.NewScheme()
	c      client.Client
)

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "foundation", "hub", "crds")},
	}

	actionv1beta1.AddToScheme(cliScheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		klog.Errorf("Failed to start, %v", err)
		os.Exit(1)
	}

	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, actionv1beta1.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := actionv1beta1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding ManagedClusterAction to scheme, %v", err)
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

	SetupWithManager(mgr)

	cancel, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		cancel()
		mgrStopped.Wait()
	}()

	time.Sleep(time.Second * 1)
}

func newTestReconciler(existingObjs []runtime.Object) *Reconciler {
	return &Reconciler{
		client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existingObjs...).Build(),
		scheme: scheme,
	}
}

func validateError(t *testing.T, err, expectedErrorType error) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
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
			name:         "ManagedClusterActionNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      actionName,
					Namespace: clusterNamespace,
				},
			},
		},
		{
			name: "ManagedClusterActionWaitOk",
			existingObjs: []runtime.Object{
				&actionv1beta1.ManagedClusterAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      actionName,
						Namespace: clusterNamespace,
					},
					Spec: actionv1beta1.ActionSpec{},
					Status: actionv1beta1.ActionStatus{
						Conditions: []metav1.Condition{
							{
								Type:               actionv1beta1.ConditionActionCompleted,
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(time.Now()),
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      actionName,
					Namespace: clusterNamespace,
				},
			},
			requeue: true,
		},
		{
			name: "ManagedClusterActionNoCondition",
			existingObjs: []runtime.Object{
				&actionv1beta1.ManagedClusterAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      actionName,
						Namespace: clusterNamespace,
					},
					Spec: actionv1beta1.ActionSpec{},
					Status: actionv1beta1.ActionStatus{
						Conditions: []metav1.Condition{},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      actionName,
					Namespace: clusterNamespace,
				},
			},
			requeue: true,
		},
		{
			name: "ManagedClusterActionDeleteOk",
			existingObjs: []runtime.Object{
				&actionv1beta1.ManagedClusterAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      actionName,
						Namespace: clusterNamespace,
					},
					Spec: actionv1beta1.ActionSpec{},
					Status: actionv1beta1.ActionStatus{
						Conditions: []metav1.Condition{
							{
								Type:               actionv1beta1.ConditionActionCompleted,
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(time.Now().Add(-120 * time.Second)),
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      actionName,
					Namespace: clusterNamespace,
				},
			},
			requeue: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc := newTestReconciler(test.existingObjs)
			res, err := svrc.Reconcile(ctx, test.req)
			validateError(t, err, test.expectedErrorType)
			assert.Equal(t, res.Requeue, false)
		})
	}
}

var objs = []runtime.Object{
	&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusterrole1",
			Finalizers: []string{
				ClustersetFinalizerName,
			},
		},
		Rules: nil,
	},
	&rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusterrole2",
			Finalizers: []string{
				ClustersetFinalizerName,
			},
		},
		Rules: nil,
	},
}

func TestClean(t *testing.T) {
	kubeClient := k8sfake.NewSimpleClientset(objs...)
	ctx := context.Background()
	gf := NewCleanGarbageFinalizer(kubeClient)
	gf.clean()
	clusterrole1, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, "clusterrole1", metav1.GetOptions{})
	validateError(t, err, nil)
	if utils.ContainsString(clusterrole1.GetFinalizers(), ClustersetFinalizerName) {
		t.Errorf("Finalizer is not removed, clusterrole: %v", clusterrole1)
	}
	clusterrole2, err := kubeClient.RbacV1().ClusterRoles().Get(ctx, "clusterrole1", metav1.GetOptions{})
	validateError(t, err, nil)
	if utils.ContainsString(clusterrole2.GetFinalizers(), ClustersetFinalizerName) {
		t.Errorf("Finalizer is not removed")
	}
}
