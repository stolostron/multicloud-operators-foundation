package clusterrole

import (
	"os"
	"testing"
	"time"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
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
	ManagedClusterName = "foo"
)

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
				Name: buildClusterRoleName(ManagedClusterName, "admin"),
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
			name:         "ManagedClusterNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
		{
			name: "ManagedClusterHasFinalizerWithoutClusterRole",
			existingObjs: []runtime.Object{
				&clusterv1.ManagedCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: ManagedClusterName,
						DeletionTimestamp: &metav1.Time{
							Time: time.Now(),
						},
						Finalizers: []string{
							clusterRoleFinalizerName,
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
			name: "ManagedClusterNoFinalizerWithoutClusterRole",
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
			name: "ManagedClusterNoFinalizerWithClusterRole",
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
				},
			},
			existingRoleOjbs: newAdminRoleObjs(),
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: ManagedClusterName,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svrc := newTestReconciler(test.existingObjs, test.existingRoleOjbs)
			clusterNamespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ManagedClusterName,
				},
			}
			svrc.kubeClient.CoreV1().Namespaces().Create(clusterNamespace)

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
