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
	clusterv1.Install(cliScheme.Scheme)
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

	SetupWithManager(mgr)

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
		client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(existingObjs...).WithStatusSubresource(&clusterinfov1beta1.ManagedClusterInfo{}).Build(),
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

func TestReconcileConditionMerge(t *testing.T) {
	ctx := context.Background()
	now := metav1.Now()

	cluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: ManagedClusterName,
			Finalizers: []string{
				clusterFinalizerName,
			},
		},
		Spec: clusterv1.ManagedClusterSpec{
			ManagedClusterClientConfigs: []clusterv1.ClientConfig{
				{
					URL: "https://test.example.com",
				},
			},
		},
		Status: clusterv1.ManagedClusterStatus{
			Conditions: []metav1.Condition{
				{
					Type:               clusterv1.ManagedClusterConditionAvailable,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "ManagedClusterAvailable",
					Message:            "Managed cluster is available",
				},
				{
					Type:               clusterv1.ManagedClusterConditionJoined,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "ManagedClusterJoined",
					Message:            "Managed cluster joined",
				},
			},
		},
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ManagedClusterName,
			Namespace: ManagedClusterName,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			MasterEndpoint: "https://test.example.com",
		},
		Status: clusterinfov1beta1.ClusterInfoStatus{
			Conditions: []metav1.Condition{
				{
					Type:               clusterinfov1beta1.ManagedClusterInfoSynced,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "ManagedClusterInfoSynced",
					Message:            "Managed cluster info is synced",
				},
				{
					Type:               "CustomCondition",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "CustomReason",
					Message:            "Custom condition",
				},
			},
		},
	}

	svrc := newTestClusterInfoReconciler([]runtime.Object{cluster, clusterInfo})
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: ManagedClusterName,
		},
	}

	_, err := svrc.Reconcile(ctx, req)
	assert.NoError(t, err)

	// Get the updated clusterInfo
	updatedClusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = svrc.client.Get(ctx, types.NamespacedName{Name: ManagedClusterName, Namespace: ManagedClusterName}, updatedClusterInfo)
	assert.NoError(t, err)

	// Verify that all conditions are present
	// Should have: Available, Joined (from cluster), ManagedClusterInfoSynced, CustomCondition (from clusterInfo)
	assert.Equal(t, 4, len(updatedClusterInfo.Status.Conditions), "Expected 4 conditions after merge")

	// Verify each condition exists
	conditionTypes := make(map[string]bool)
	for _, condition := range updatedClusterInfo.Status.Conditions {
		conditionTypes[condition.Type] = true
	}

	assert.True(t, conditionTypes[clusterv1.ManagedClusterConditionAvailable], "Available condition should be present")
	assert.True(t, conditionTypes[clusterv1.ManagedClusterConditionJoined], "Joined condition should be present")
	assert.True(t, conditionTypes[clusterinfov1beta1.ManagedClusterInfoSynced], "ManagedClusterInfoSynced condition should be preserved")
	assert.True(t, conditionTypes["CustomCondition"], "CustomCondition should be preserved")
}

func TestReconcileConditionUpdate(t *testing.T) {
	ctx := context.Background()
	oldTime := metav1.NewTime(time.Now().Add(-1 * time.Hour))
	newTime := metav1.Now()

	cluster := &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: ManagedClusterName,
			Finalizers: []string{
				clusterFinalizerName,
			},
		},
		Spec: clusterv1.ManagedClusterSpec{
			ManagedClusterClientConfigs: []clusterv1.ClientConfig{
				{
					URL: "https://test.example.com",
				},
			},
		},
		Status: clusterv1.ManagedClusterStatus{
			Conditions: []metav1.Condition{
				{
					Type:               clusterv1.ManagedClusterConditionAvailable,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: newTime,
					Reason:             "ManagedClusterNotAvailable",
					Message:            "Managed cluster is not available",
				},
			},
		},
	}

	clusterInfo := &clusterinfov1beta1.ManagedClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ManagedClusterName,
			Namespace: ManagedClusterName,
		},
		Spec: clusterinfov1beta1.ClusterInfoSpec{
			MasterEndpoint: "https://test.example.com",
		},
		Status: clusterinfov1beta1.ClusterInfoStatus{
			Conditions: []metav1.Condition{
				{
					Type:               clusterv1.ManagedClusterConditionAvailable,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: oldTime,
					Reason:             "ManagedClusterAvailable",
					Message:            "Managed cluster is available",
				},
				{
					Type:               clusterinfov1beta1.ManagedClusterInfoSynced,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: oldTime,
					Reason:             "ManagedClusterInfoSynced",
					Message:            "Managed cluster info is synced",
				},
			},
		},
	}

	svrc := newTestClusterInfoReconciler([]runtime.Object{cluster, clusterInfo})
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name: ManagedClusterName,
		},
	}

	_, err := svrc.Reconcile(ctx, req)
	assert.NoError(t, err)

	// Get the updated clusterInfo
	updatedClusterInfo := &clusterinfov1beta1.ManagedClusterInfo{}
	err = svrc.client.Get(ctx, types.NamespacedName{Name: ManagedClusterName, Namespace: ManagedClusterName}, updatedClusterInfo)
	assert.NoError(t, err)

	// Verify that conditions are properly updated
	var availableCondition *metav1.Condition
	var syncedCondition *metav1.Condition
	for i, condition := range updatedClusterInfo.Status.Conditions {
		if condition.Type == clusterv1.ManagedClusterConditionAvailable {
			availableCondition = &updatedClusterInfo.Status.Conditions[i]
		}
		if condition.Type == clusterinfov1beta1.ManagedClusterInfoSynced {
			syncedCondition = &updatedClusterInfo.Status.Conditions[i]
		}
	}

	assert.NotNil(t, availableCondition, "Available condition should be present")
	assert.Equal(t, metav1.ConditionFalse, availableCondition.Status, "Available condition should be updated to False")
	assert.Equal(t, "ManagedClusterNotAvailable", availableCondition.Reason, "Available condition reason should be updated")

	assert.NotNil(t, syncedCondition, "ManagedClusterInfoSynced condition should be preserved")
	assert.Equal(t, metav1.ConditionTrue, syncedCondition.Status, "ManagedClusterInfoSynced condition should remain True")
}
