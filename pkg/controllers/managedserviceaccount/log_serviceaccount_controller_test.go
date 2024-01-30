package managedserviceaccount

import (
	"context"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	msav1beta1 "open-cluster-management.io/managed-serviceaccount/apis/authentication/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

var (
	scheme = runtime.NewScheme()
)

func newTestReconciler(existingObjs ...runtime.Object) *Reconciler {
	s := kubescheme.Scheme
	_ = clusterv1.Install(s)
	_ = msav1beta1.AddToScheme(s)

	client := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(existingObjs...).Build()
	return &Reconciler{
		client:     client,
		scheme:     scheme,
		logMcaName: LogManagedServiceAccountName,
	}
}

func TestReconciler(t *testing.T) {
	clusterName := "cluster1"
	ctx := context.TODO()

	tests := []struct {
		name            string
		existingCluster *clusterv1.ManagedCluster
		existingMsa     *msav1beta1.ManagedServiceAccount
		createdMsa      bool
	}{
		{
			name:       "no cluster",
			createdMsa: false,
		},
		{
			name: "cluster without label",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			},
			createdMsa: false,
		},
		{
			name: "cluster without addon feature availabel label",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   clusterName,
					Labels: map[string]string{helpers.MsaAddOnFeatureLabel: ""},
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
			createdMsa: true,
		},
		{
			name: "has cluster and msa, update msa",
			existingCluster: &clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			},
			existingMsa: &msav1beta1.ManagedServiceAccount{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      LogManagedServiceAccountName,
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
			if test.existingMsa != nil {
				existingObjs = append(existingObjs, test.existingMsa)
			}

			r := newTestReconciler(existingObjs...)
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}})
			if err != nil {
				t.Errorf("unexpected error :%v", err)
			}

			logMsa := &msav1beta1.ManagedServiceAccount{}
			err = r.client.Get(ctx, types.NamespacedName{Name: r.logMcaName, Namespace: clusterName}, logMsa)
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
