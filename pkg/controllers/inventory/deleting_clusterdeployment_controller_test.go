package inventory

import (
	"context"
	"testing"

	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	inventoryv1alpha1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestCDReconciler(existingObjs []runtime.Object) (*ReconcileClusterDeployment, client.Client) {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, existingObjs...)
	rbma := &ReconcileClusterDeployment{
		client: fakeClient,
		scheme: scheme.Scheme,
	}
	return rbma, fakeClient
}

func TestCDReconcile(t *testing.T) {
	tests := []struct {
		name              string
		existingObjs      []runtime.Object
		expectedErrorType error
		expectedFinalizer []string
		req               reconcile.Request
		requeue           bool
	}{
		{
			name: "do not add finalizer",
			existingObjs: []runtime.Object{
				newClusterDeployment(),
			},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
			expectedFinalizer: []string{},
		},
		{
			name: "add finalizer",
			existingObjs: []runtime.Object{
				newClusterDeployment(),
				func() *inventoryv1alpha1.BareMetalAsset {
					bma := newBMAWithClusterDeployment()
					bma.Labels = map[string]string{
						ClusterDeploymentNameLabel:      testName,
						ClusterDeploymentNamespaceLabel: testNamespace,
					}
					return bma
				}(),
			},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
			expectedFinalizer: []string{BareMetalAssetFinalizer},
		},
		{
			name: "remove finalizer with no bma",
			existingObjs: []runtime.Object{
				func() *hivev1.ClusterDeployment {
					cd := newClusterDeployment()
					now := metav1.Now()
					cd.DeletionTimestamp = &now
					return cd
				}(),
			},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
			expectedFinalizer: []string{},
		},
		{
			name: "remove finalizer with bma",
			existingObjs: []runtime.Object{
				func() *hivev1.ClusterDeployment {
					cd := newClusterDeployment()
					now := metav1.Now()
					cd.DeletionTimestamp = &now
					cd.Finalizers = []string{BareMetalAssetFinalizer}
					return cd
				}(),
				func() *inventoryv1alpha1.BareMetalAsset {
					bma := newBMAWithClusterDeployment()
					bma.Labels = map[string]string{
						ClusterDeploymentNameLabel:      testName,
						ClusterDeploymentNamespaceLabel: testNamespace,
					}
					return bma
				}(),
			},
			expectedErrorType: nil,
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
			expectedFinalizer: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma, client := newTestCDReconciler(test.existingObjs)
			_, err := rbma.Reconcile(test.req)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, nil, nil)
			cd := &hivev1.ClusterDeployment{}
			err = client.Get(context.TODO(), types.NamespacedName{
				Name:      testName,
				Namespace: testNamespace,
			}, cd)
			validateErrorAndStatusConditions(t, err, nil, nil, nil)
			if len(cd.Finalizers) != len(test.expectedFinalizer) {
				t.Errorf("finalizer is not correct, actual %v, expected %v", cd.Finalizers, test.expectedFinalizer)
			}
		})
	}
}
