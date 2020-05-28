package clusterinfo

import (
	"os"
	"testing"
	"time"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"github.com/stretchr/testify/assert"
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
	AddToSchemes = append(AddToSchemes, clusterv1.Install, clusterv1beta1.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := clusterv1beta1.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding cluster info to scheme, %v", err)
		os.Exit(1)
	}
	if err := clusterv1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

const (
	spokeClusterName = "foo"
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
		caData: []byte{},
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
			name:         "SpokeClusterNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: spokeClusterName,
				},
			},
		},
		{
			name: "SpokeClusterHasFinalizerWithoutClusterInfo",
			existingObjs: []runtime.Object{
				&clusterv1.SpokeCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: spokeClusterName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clusterFinalizerName,
						},
					},
					Spec: clusterv1.SpokeClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: spokeClusterName,
				},
			},
		},
		{
			name: "SpokeClusterNoFinalizerWithoutClusterInfo",
			existingObjs: []runtime.Object{
				&clusterv1.SpokeCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: spokeClusterName,
					},
					Spec: clusterv1.SpokeClusterSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: spokeClusterName,
				},
			},
			requeue: true,
		},
		{
			name: "SpokeClusterNoFinalizerWithClusterInfo",
			existingObjs: []runtime.Object{
				&clusterv1.SpokeCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: spokeClusterName,
					},
					Spec: clusterv1.SpokeClusterSpec{},
				},
				&clusterv1beta1.ClusterInfo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      spokeClusterName,
						Namespace: spokeClusterName,
					},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: spokeClusterName,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc := newTestReconciler(test.existingObjs)
			res, err := svrc.ReconcileBySpokeCluster(test.req)
			validateError(t, err, test.expectedErrorType)
			if test.requeue {
				assert.Equal(t, res.Requeue, true)
			} else {
				assert.Equal(t, res.Requeue, false)
			}
		})
	}
}
