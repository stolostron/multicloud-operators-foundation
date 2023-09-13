package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	gmg "github.com/onsi/gomega"
	cacheddiscovery "k8s.io/client-go/discovery/cached"

	viewv1beta1 "github.com/stolostron/cluster-lifecycle-api/view/v1beta1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme = runtime.NewScheme()
)

func setupEnvtest(t *testing.T) (*rest.Config, func(t *testing.T)) {
	t.Log("Setup envtest")

	testEnv := &envtest.Environment{}
	g := gmg.NewWithT(t)

	cfg, err := testEnv.Start()
	g.Expect(err).NotTo(gmg.HaveOccurred())
	g.Expect(cfg).NotTo(gmg.BeNil())

	teardownFunc := func(t *testing.T) {
		t.Log("Stop envtest")
		g.Expect(testEnv.Stop()).To(gmg.Succeed())
	}

	return cfg, teardownFunc
}

func TestMain(m *testing.M) {
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, viewv1beta1.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := viewv1beta1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding managedClusterView to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

const (
	managedClusterViewName = "foo"
	clusterNamespace       = "bar"
)

func newUnstructured() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      "deployment_test",
			},
		},
	}
}

func newTestReconciler(existingObjs []client.Object, fakeMapper meta.RESTMapper) *ViewReconciler {
	viewReconciler := &ViewReconciler{
		Client: fake.NewClientBuilder().WithScheme(scheme).
			WithObjects(existingObjs...).WithStatusSubresource(existingObjs...).
			Build(),
		Log:                         ctrl.Log.WithName("controllers").WithName("ManagedClusterView"),
		Scheme:                      scheme,
		ManagedClusterDynamicClient: dynamicfake.NewSimpleDynamicClient(scheme, newUnstructured()),
		Mapper:                      fakeMapper,
	}

	return viewReconciler
}

func validateErrorAndStatusConditions(t *testing.T, err, expectedErrorType error,
	expectedConditions []metav1.Condition, view *viewv1beta1.ManagedClusterView) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}

	for _, condition := range expectedConditions {
		assert.True(t, meta.IsStatusConditionPresentAndEqual(view.Status.Conditions, condition.Type, condition.Status))
	}
	if view != nil {
		assert.Equal(t, len(expectedConditions), len(view.Status.Conditions))
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name              string
		existingObjs      []client.Object
		expectedErrorType error
		req               reconcile.Request
		requeue           bool
	}{
		{
			name:         "managedClusterViewNotFound",
			existingObjs: []client.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
			},
		},
		{
			name: "managedClusterViewWaitOK",
			existingObjs: []client.Object{
				&viewv1beta1.ManagedClusterView{
					ObjectMeta: metav1.ObjectMeta{
						Name:      managedClusterViewName,
						Namespace: clusterNamespace,
					},
					Spec: viewv1beta1.ViewSpec{
						Scope: viewv1beta1.ViewScope{
							Group:                 "",
							Version:               "v1",
							Kind:                  "Deployment",
							Name:                  "deployment_test",
							Namespace:             "default",
							UpdateIntervalSeconds: 10,
						},
					},
					Status: viewv1beta1.ViewStatus{
						Conditions: []metav1.Condition{
							{
								Type:               viewv1beta1.ConditionViewProcessing,
								Status:             metav1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(time.Now()),
							},
						},
						Result: runtime.RawExtension{},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
			},
			requeue: true,
		},
	}

	restCfg, tearDownFn := setupEnvtest(t)
	defer tearDownFn(t)

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		t.Fatalf("Failed to create discovery client, %v", err)
	}
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cacheddiscovery.NewMemCacheClient(discoveryClient))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc := newTestReconciler(test.existingObjs, restMapper)
			res, err := svrc.Reconcile(ctx, test.req)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, nil, nil)

			if test.requeue {
				assert.Equal(t, res.Requeue, false)
				assert.NotEqual(t, res.RequeueAfter, 0*time.Second)
			} else {
				assert.Equal(t, res, reconcile.Result{Requeue: false, RequeueAfter: 0})
			}
		})
	}
}

func TestQueryResource(t *testing.T) {
	tests := []struct {
		name               string
		managedClusterView *viewv1beta1.ManagedClusterView
		expectedErrorType  error
		expectedConditions []metav1.Condition
	}{
		{
			name: "queryResource Resource Only",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Resource:  "deployment",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionTrue,
				},
			},
		},
		{
			name: "queryResource GVK",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Group:     "apps",
						Version:   "v1",
						Kind:      "Deployment",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionTrue,
				},
			},
		},
		{
			name: "queryResourceFailedNoName",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Resource:  "deployments",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("invalid resource name"),
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceFailedNoResource",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("invalid resource type"),
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceFailedMapper",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Group:     "core",
						Version:   "v1",
						Kind:      "Deployment",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("no matches for kind \"Deployment\" in version \"core/v1\""),
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceFailedMapper2",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Resource:  "deploymentts",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: errors.New("fail to mapping GroupKind deploymentts, GroupKindVersion , resource:deploymentts err: no matches for kind \"deploymentts\" in version \"\""),
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceSecrets",
			managedClusterView: &viewv1beta1.ManagedClusterView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      managedClusterViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.ViewSpec{
					Scope: viewv1beta1.ViewScope{
						Resource:  "secrets",
						Name:      "secret1",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: nil,
			expectedConditions: []metav1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: metav1.ConditionFalse,
				},
			},
		},
	}

	restCfg, tearDownFn := setupEnvtest(t)
	defer tearDownFn(t)

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restCfg)
	if err != nil {
		t.Fatalf("Failed to create discovery client, %v", err)
	}
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cacheddiscovery.NewMemCacheClient(discoveryClient))

	svrc := newTestReconciler([]client.Object{}, restMapper)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := svrc.queryResource(test.managedClusterView)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, test.expectedConditions, test.managedClusterView)
		})
	}
}
