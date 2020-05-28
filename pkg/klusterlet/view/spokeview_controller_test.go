package controllers

import (
	"fmt"
	"os"
	"testing"
	"time"

	tlog "github.com/go-logr/logr/testing"
	viewv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/view/v1beta1"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme = runtime.NewScheme()
)

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
		klog.Errorf("Failed adding spokeView to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

const (
	spokeViewName    = "foo"
	clusterNamespace = "bar"
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

func newTestReconciler(existingObjs []runtime.Object) *SpokeViewReconciler {
	resources := []*restmapper.APIGroupResources{
		{
			Group: metav1.APIGroup{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
				},
				Name: "apps",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "v1",
						Version:      "v1",
					},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: "v1",
					Version:      "v1",
				},
				ServerAddressByClientCIDRs: nil,
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1": {
					{
						Name:         "deployments",
						SingularName: "deployment",
						Group:        "apps",
						Kind:         "Deployment",
						Version:      "v1",
					},
				},
			},
		},
	}

	spokeViewReconciler := &SpokeViewReconciler{
		Client:             fake.NewFakeClientWithScheme(scheme, existingObjs...),
		Log:                tlog.NullLogger{},
		Scheme:             scheme,
		SpokeDynamicClient: dynamicfake.NewSimpleDynamicClient(scheme, newUnstructured()),
		Mapper:             restutils.NewFakeMapper(resources),
	}

	return spokeViewReconciler
}

func validateErrorAndStatusConditions(t *testing.T, err, expectedErrorType error,
	expectedConditions []conditionsv1.Condition, spokeView *viewv1beta1.SpokeView) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}

	for _, condition := range expectedConditions {
		assert.True(t, conditionsv1.IsStatusConditionPresentAndEqual(spokeView.Status.Conditions, condition.Type, condition.Status))
	}
	if spokeView != nil {
		assert.Equal(t, len(expectedConditions), len(spokeView.Status.Conditions))
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
			name:         "SpokeViewNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
			},
		},
		{
			name: "SpokeViewWaitOK",
			existingObjs: []runtime.Object{
				&viewv1beta1.SpokeView{
					ObjectMeta: metav1.ObjectMeta{
						Name:      spokeViewName,
						Namespace: clusterNamespace,
					},
					Spec: viewv1beta1.SpokeViewSpec{
						Scope: viewv1beta1.SpokeViewScope{
							Group:                 "",
							Version:               "v1",
							Kind:                  "Deployment",
							Name:                  "deployment_test",
							Namespace:             "default",
							UpdateIntervalSeconds: 10,
						},
					},
					Status: viewv1beta1.SpokeViewStatus{
						Conditions: []conditionsv1.Condition{
							{
								Type:               viewv1beta1.ConditionViewProcessing,
								Status:             corev1.ConditionTrue,
								LastHeartbeatTime:  metav1.NewTime(time.Now()),
								LastTransitionTime: metav1.NewTime(time.Now()),
							},
						},
						Result: runtime.RawExtension{},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      spokeViewName,
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
		spokeView          *viewv1beta1.SpokeView
		expectedErrorType  error
		expectedConditions []conditionsv1.Condition
	}{
		{
			name: "queryResourceOK",
			spokeView: &viewv1beta1.SpokeView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.SpokeViewSpec{
					Scope: viewv1beta1.SpokeViewScope{
						Group:     "apps",
						Version:   "v1",
						Kind:      "Deployment",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedConditions: []conditionsv1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: corev1.ConditionTrue,
				},
			},
		},
		{
			name: "queryResourceOK2",
			spokeView: &viewv1beta1.SpokeView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.SpokeViewSpec{
					Scope: viewv1beta1.SpokeViewScope{
						Resource:  "deployment",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedConditions: []conditionsv1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: corev1.ConditionTrue,
				},
			},
		},
		{
			name: "queryResourceFailedNoName",
			spokeView: &viewv1beta1.SpokeView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.SpokeViewSpec{
					Scope: viewv1beta1.SpokeViewScope{
						Resource:  "deployments",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("invalid resource name"),
			expectedConditions: []conditionsv1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: corev1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceFailedNoResource",
			spokeView: &viewv1beta1.SpokeView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.SpokeViewSpec{
					Scope: viewv1beta1.SpokeViewScope{
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("invalid resource type"),
			expectedConditions: []conditionsv1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: corev1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceFailedMapper",
			spokeView: &viewv1beta1.SpokeView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.SpokeViewSpec{
					Scope: viewv1beta1.SpokeViewScope{
						Group:     "core",
						Version:   "v1",
						Kind:      "Deployment",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("the server doesn't have a resource type \"Deployment\""),
			expectedConditions: []conditionsv1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: corev1.ConditionFalse,
				},
			},
		},
		{
			name: "queryResourceFailedMapper2",
			spokeView: &viewv1beta1.SpokeView{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeViewName,
					Namespace: clusterNamespace,
				},
				Spec: viewv1beta1.SpokeViewSpec{
					Scope: viewv1beta1.SpokeViewScope{
						Resource:  "deploymentts",
						Name:      "deployment_test",
						Namespace: "default",
					},
				},
			},
			expectedErrorType: fmt.Errorf("the server doesn't have a resource type \"deploymentts\""),
			expectedConditions: []conditionsv1.Condition{
				{
					Type:   viewv1beta1.ConditionViewProcessing,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	svrc := newTestReconciler([]runtime.Object{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := svrc.queryResource(test.spokeView)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, test.expectedConditions, test.spokeView)
		})
	}
}
