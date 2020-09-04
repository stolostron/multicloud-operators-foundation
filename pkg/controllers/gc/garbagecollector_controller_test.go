package gc

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"
	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
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
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "deploy", "foundation", "hub", "resources", "crds")},
	}

	actionv1beta1.AddToScheme(cliScheme.Scheme)

	var err error
	if cfg, err = t.Start(); err != nil {
		klog.Errorf("Failed to start, %v", err)
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
func StartTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup) {
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
	}()

	return stop, wg
}

func TestControllerReconcile(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	c = mgr.GetClient()

	SetupWithManager(mgr)

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	time.Sleep(time.Second * 1)
}

func newTestReconciler(existingObjs []runtime.Object) *Reconciler {
	return &Reconciler{
		client: fake.NewFakeClientWithScheme(scheme, existingObjs...),
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
			res, err := svrc.Reconcile(test.req)
			validateError(t, err, test.expectedErrorType)
			assert.Equal(t, res.Requeue, false)
		})
	}
}
