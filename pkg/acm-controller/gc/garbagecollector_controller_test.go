package gc

import (
	"os"
	"testing"
	"time"

	actionv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/action/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/conditions"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	AddToSchemes = append(AddToSchemes, actionv1beta1.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := actionv1beta1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding clusteraction to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

const (
	clusterActionName = "foo"
	clusterNamespace  = "bar"
)

func validateError(t *testing.T, err, expectedErrorType error) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}
}

func newTestReconciler(existingObjs []runtime.Object) *Reconciler {
	return &Reconciler{
		client: fake.NewFakeClientWithScheme(scheme, existingObjs...),
		scheme: scheme,
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
			name:         "ClusterActionNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      clusterActionName,
					Namespace: clusterNamespace,
				},
			},
		},
		{
			name: "ClusterActionWaitOk",
			existingObjs: []runtime.Object{
				&actionv1beta1.ClusterAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterActionName,
						Namespace: clusterNamespace,
					},
					Spec: actionv1beta1.ClusterActionSpec{},
					Status: actionv1beta1.ClusterActionStatus{
						Conditions: []conditions.Condition{
							{
								Type:               actionv1beta1.ConditionActionCompleted,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(time.Now()),
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      clusterActionName,
					Namespace: clusterNamespace,
				},
			},
			requeue: true,
		},
		{
			name: "ClusterActionNoCondition",
			existingObjs: []runtime.Object{
				&actionv1beta1.ClusterAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterActionName,
						Namespace: clusterNamespace,
					},
					Spec: actionv1beta1.ClusterActionSpec{},
					Status: actionv1beta1.ClusterActionStatus{
						Conditions: []conditions.Condition{},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      clusterActionName,
					Namespace: clusterNamespace,
				},
			},
			requeue: true,
		},
		{
			name: "ClusterActionDeleteOk",
			existingObjs: []runtime.Object{
				&actionv1beta1.ClusterAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterActionName,
						Namespace: clusterNamespace,
					},
					Spec: actionv1beta1.ClusterActionSpec{},
					Status: actionv1beta1.ClusterActionStatus{
						Conditions: []conditions.Condition{
							{
								Type:               actionv1beta1.ConditionActionCompleted,
								Status:             corev1.ConditionTrue,
								LastTransitionTime: metav1.NewTime(time.Now().Add(-120 * time.Second)),
							},
						},
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      clusterActionName,
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
