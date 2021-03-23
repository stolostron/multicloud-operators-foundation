package clusterrole

import (
	"context"
	"os"
	"testing"
	"time"

	cliScheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme = runtime.NewScheme()
	cfg    *rest.Config
	c      client.Client
)

const (
	ManagedClusterSetName = "foo"
)

func TestMain(m *testing.M) {
	clusterv1alpha1.AddToScheme(cliScheme.Scheme)
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, clusterv1alpha1.Install)

	if err := AddToSchemes.AddToScheme(scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}
	if err := clusterv1alpha1.Install(scheme); err != nil {
		klog.Errorf("Failed adding cluster to scheme, %v", err)
		os.Exit(1)
	}

	exitVal := m.Run()
	os.Exit(exitVal)
}

func validateError(t *testing.T, err, expectedErrorType error) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}
}

func newTestReconciler(existingObjs, existingRoleOjb []runtime.Object) *Reconciler {
	return &Reconciler{
		client:     fake.NewFakeClientWithScheme(scheme, existingObjs...),
		scheme:     scheme,
		kubeClient: k8sfake.NewSimpleClientset(existingRoleOjb...),
	}
}
func newAdminRoleObjs() []runtime.Object {
	return []runtime.Object{
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: utils.BuildClusterRoleName(ManagedClusterSetName, "clusterset-admin"),
			},
			Rules: nil,
		},
	}
}

func TestReconcile(t *testing.T) {
	tests := []struct {
		name              string
		existingObjs      []runtime.Object
		existingRoleOjbs  []runtime.Object
		expectedErrorType error
		req               reconcile.Request
		requeue           bool
	}{
		{
			name:         "ManagedClusterSetNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
		},
		{
			name: "ManagedClusterSetHasFinalizerWithoutClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1alpha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterSetName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clustersetRoleFinalizerName,
						},
					},
					Spec: clusterv1alpha1.ManagedClusterSetSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
		},
		{
			name: "ManagedClusterSetNoFinalizerWithoutClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1alpha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterSetName,
					},
					Spec: clusterv1alpha1.ManagedClusterSetSpec{},
				},
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
			requeue: false,
		},
		{
			name: "ManagedClusterSetNoFinalizerWithClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1alpha1.ManagedClusterSet{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterSetName,
					},
					Spec: clusterv1alpha1.ManagedClusterSetSpec{},
				},
			},
			existingRoleOjbs: newAdminRoleObjs(),
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterSetName,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc := newTestReconciler(test.existingObjs, test.existingRoleOjbs)
			clusterNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ManagedClusterSetName,
				},
			}
			svrc.kubeClient.CoreV1().Namespaces().Create(context.TODO(), clusterNamespace, metav1.CreateOptions{})

			res, err := svrc.Reconcile(test.req)
			validateError(t, err, test.expectedErrorType)
			if test.requeue {
				assert.Equal(t, res.Requeue, true)
			} else {
				assert.Equal(t, res.Requeue, false)
			}
		})
	}
}
