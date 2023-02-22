package addoninstall

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	apiconstants "github.com/stolostron/cluster-lifecycle-api/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"open-cluster-management.io/addon-framework/pkg/addonmanager/constants"
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
		name                       string
		existingCluster            *clusterv1.ManagedCluster
		existingAddOn              *addonapiv1alpha1.ManagedClusterAddOn
		enabled                    bool
		expectedInstallNamespace   string
		expectedHostingClusterName string
	}{
		{
			name: "no cluster",
		},
		{
			name: "cluster with add-on disabled",
			existingCluster: newManagedCluster(clusterName, map[string]string{
				constants.DisableAddonAutomaticInstallationAnnotationKey: "true",
			}),
		},
		{
			name:            "add-on exists",
			existingCluster: newManagedCluster(clusterName, nil),
			existingAddOn: newManagedClusterAddOn(addOnName, clusterName, map[string]string{
				AnnotationAddOnHostingClusterName: "hosting-cluster",
			}),
		},
		{
			name:                     "cluster in default mode",
			existingCluster:          newManagedCluster(clusterName, nil),
			enabled:                  true,
			expectedInstallNamespace: DefaultAddOnInstallNamespace,
		},
		{
			name: "cluster in hosted mode with add-on in default mode",
			existingCluster: newManagedCluster(clusterName, map[string]string{
				apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
				apiconstants.AnnotationKlusterletHostingClusterName: "local-cluster",
			}),
			enabled:                  true,
			expectedInstallNamespace: DefaultAddOnInstallNamespace,
		},
		{
			name: "cluster in hosted mode with add-on in host mode",
			existingCluster: newManagedCluster(clusterName, map[string]string{
				apiconstants.AnnotationKlusterletDeployMode:         "Hosted",
				apiconstants.AnnotationKlusterletHostingClusterName: "local-cluster",
				AnnotationEnableHostedModeAddons:                    "true",
			}),
			enabled:                    true,
			expectedInstallNamespace:   fmt.Sprintf("klusterlet-%s", clusterName),
			expectedHostingClusterName: "local-cluster",
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
			case test.existingAddOn != nil:
				if err != nil {
					t.Errorf("unexpected error :%v", err)
				}

				if !reflect.DeepEqual(addOn.ObjectMeta, test.existingAddOn.ObjectMeta) {
					t.Errorf("expect addon cr \n%v but got: \n%v", test.existingAddOn, addOn)
				}

				if !reflect.DeepEqual(addOn.Spec, test.existingAddOn.Spec) {
					t.Errorf("expect addon cr \n%v but got: \n%v", test.existingAddOn, addOn)
				}
			case test.enabled:
				if err != nil {
					t.Errorf("unexpected error :%v", err)
				}
				if addOn.Spec.InstallNamespace != test.expectedInstallNamespace {
					t.Errorf("expect install namespace %q but got: %v", test.expectedInstallNamespace, addOn.Spec.InstallNamespace)
				}

				if value := addOn.Annotations[AnnotationAddOnHostingClusterName]; value != test.expectedHostingClusterName {
					t.Errorf("expect hosting cluster %q but got: %v", test.expectedInstallNamespace, value)
				}
			case err == nil:
				t.Errorf("should not create add-on cr")
			case !errors.IsNotFound(err):
				t.Errorf("unexpected error :%v", err)
			}
		})
	}
}
