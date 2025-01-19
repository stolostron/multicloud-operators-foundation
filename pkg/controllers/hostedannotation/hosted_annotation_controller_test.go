package hostedannotation

import (
	"context"
	"reflect"
	"testing"

	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	"github.com/stolostron/multicloud-operators-foundation/pkg/addon"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	scheme                 = runtime.NewScheme()
	logCertSecretNamespace = "open-cluster-management"
	logCertSecretName      = "ocm-klusterlet-self-signed-secrets"
)

func newTestReconciler(addOnName string, existingObjs ...runtime.Object) *Reconciler {
	s := kubescheme.Scheme
	clusterv1.Install(s)
	addonapiv1alpha1.Install(s)

	client := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(existingObjs...).Build()
	return &Reconciler{
		client:     client,
		scheme:     scheme,
		addOnNames: sets.NewString(addOnName),
	}
}

func newManagedCluster(name string, annotations map[string]string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
	}
}

func newManagedClusterAddOn(addOnName, clusterName string, annotations map[string]string) *addonapiv1alpha1.ManagedClusterAddOn {
	return &addonapiv1alpha1.ManagedClusterAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:        addOnName,
			Namespace:   clusterName,
			Annotations: annotations,
		},
	}
}

func TestReconciler(t *testing.T) {
	addOnName := "test-addon"
	clusterName := "cluster1"
	ctx := context.TODO()

	tests := []struct {
		name            string
		existingCluster *clusterv1.ManagedCluster
		existingAddOn   *addonapiv1alpha1.ManagedClusterAddOn
		expectedAddOn   *addonapiv1alpha1.ManagedClusterAddOn
	}{
		{
			name: "no cluster",
		},
		{
			name:            "no addon",
			existingCluster: newManagedCluster(clusterName, nil),
		},
		{
			name:            "cluster in default mode",
			existingCluster: newManagedCluster(clusterName, nil),
		},
		{
			name: "add annotation",
			existingCluster: newManagedCluster(clusterName, map[string]string{
				apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
				apiconstants.AnnotationKlusterletHostingClusterName: "local-cluster",
				addon.AnnotationEnableHostedModeAddons:              "true",
			}),
			existingAddOn: newManagedClusterAddOn(addOnName, clusterName, nil),
			expectedAddOn: newManagedClusterAddOn(addOnName, clusterName, map[string]string{
				addonapiv1alpha1.HostingClusterNameAnnotationKey: "local-cluster",
			}),
		},
		{
			name: "no update",
			existingCluster: newManagedCluster(clusterName, map[string]string{
				apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
				apiconstants.AnnotationKlusterletHostingClusterName: "local-cluster",
				addon.AnnotationEnableHostedModeAddons:              "true",
			}),
			existingAddOn: newManagedClusterAddOn(addOnName, clusterName, map[string]string{
				addonapiv1alpha1.HostingClusterNameAnnotationKey: "local-cluster",
				"foo": "bar",
			}),
			expectedAddOn: newManagedClusterAddOn(addOnName, clusterName, map[string]string{
				addonapiv1alpha1.HostingClusterNameAnnotationKey: "local-cluster",
				"foo": "bar",
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			existingObjs := []runtime.Object{}
			if test.existingCluster != nil {
				existingObjs = append(existingObjs, test.existingCluster)
			}
			if test.existingAddOn != nil {
				existingObjs = append(existingObjs, test.existingAddOn)
			}

			r := newTestReconciler(addOnName, existingObjs...)
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterName}})
			if err != nil {
				t.Errorf("unexpected error :%v", err)
			}

			addOn := &addonapiv1alpha1.ManagedClusterAddOn{}
			err = r.client.Get(ctx, types.NamespacedName{Name: addOnName, Namespace: clusterName}, addOn)

			switch {
			case test.expectedAddOn != nil:
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(addOn.Annotations, test.expectedAddOn.Annotations) {
					t.Errorf("expect addon cr \n%v but got: \n%v", test.expectedAddOn.Annotations, addOn.Annotations)
				}
			}
		})
	}
}
