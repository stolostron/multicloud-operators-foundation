package managedserviceaccount

import (
	"context"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	msav1beta1 "open-cluster-management.io/managed-serviceaccount/apis/authentication/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	msav1beta1fake "open-cluster-management.io/managed-serviceaccount/pkg/generated/clientset/versioned/fake"
)

var (
	scheme = runtime.NewScheme()
)

func newTestReconciler(msaObj *msav1beta1.ManagedServiceAccount, existingObjs ...runtime.Object) *Reconciler {
	s := kubescheme.Scheme
	_ = clusterv1.Install(s)
	_ = msav1beta1.AddToScheme(s)
	_ = addonv1alpha1.Install(s)

	msaList := []runtime.Object{
		&msav1beta1.ManagedServiceAccount{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			}},
	}
	if msaObj != nil {
		msaList = append(msaList, msaObj)
	}
	fakeMsaClient := msav1beta1fake.NewSimpleClientset(msaList...).AuthenticationV1beta1()

	client := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(existingObjs...).Build()
	return &Reconciler{
		msaClient:  fakeMsaClient,
		client:     client,
		scheme:     scheme,
		logMcaName: helpers.LogManagedServiceAccountName,
	}
}

func TestReconciler(t *testing.T) {
	clusterName := "cluster1"
	ctx := context.TODO()

	tests := []struct {
		name            string
		existingCluster *clusterv1.ManagedCluster
		existingMsa     *msav1beta1.ManagedServiceAccount
		existingAddon   *addonv1alpha1.ManagedClusterAddOn
		createdMsa      bool
	}{
		{
			name:       "no cluster",
			createdMsa: false,
		},
		{
			name: "cluster is deleting",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"test"},
				},
			},
			createdMsa: false,
		},
		{
			name: "no addon",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			},
			createdMsa: false,
		},
		{
			name: "addon is deleting",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			},
			existingAddon: &addonv1alpha1.ManagedClusterAddOn{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: clusterName,
					Name:      helpers.MsaAddonName,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"test"},
				},
			},
			createdMsa: false,
		},

		{
			name: "has cluster no msa, create msa",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			},
			existingAddon: &addonv1alpha1.ManagedClusterAddOn{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: clusterName,
					Name:      helpers.MsaAddonName,
				},
			},
			createdMsa: true,
		},
		{
			name: "has cluster and msa, update msa",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			},
			existingAddon: &addonv1alpha1.ManagedClusterAddOn{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: clusterName,
					Name:      helpers.MsaAddonName,
				},
			},
			existingMsa: &msav1beta1.ManagedServiceAccount{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      helpers.LogManagedServiceAccountName,
					Namespace: clusterName,
				},
				Spec: msav1beta1.ManagedServiceAccountSpec{
					Rotation: msav1beta1.ManagedServiceAccountRotation{
						Enabled:  true,
						Validity: metav1.Duration{Duration: time.Minute * 365 * 24 * 60},
					},
				},
			},
			createdMsa: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			existingObjs := []runtime.Object{}
			if test.existingCluster != nil {
				existingObjs = append(existingObjs, test.existingCluster)
			}
			if test.existingAddon != nil {
				existingObjs = append(existingObjs, test.existingAddon)
			}
			r := newTestReconciler(test.existingMsa, existingObjs...)
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}})
			if err != nil {
				t.Errorf("unexpected error :%v", err)
			}

			logMsa, err := r.msaClient.ManagedServiceAccounts(clusterName).Get(context.TODO(), r.logMcaName, metav1.GetOptions{})
			switch {
			case errors.IsNotFound(err):
				if test.createdMsa {
					t.Errorf("unexpected error :%v", err)
				}
			case err != nil:
				t.Errorf("unexpected error :%v", err)
			default:
				if !logMsa.Spec.Rotation.Enabled {
					t.Errorf("unexpected spce: %v", logMsa.Spec.Rotation)
				}
			}
		})
	}
}
