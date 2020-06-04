package inventory

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	inventoryv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	bmaerrors "github.com/open-cluster-management/multicloud-operators-foundation/pkg/controller/inventory/errors"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	hiveconstants "github.com/openshift/hive/pkg/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testName      = "foo"
	testNamespace = "bar"
	testBMHKind   = "BareMetalHost"
	testSSKind    = "SyncSet"
	testRoleLabel = "metal3.io/role"
)

var _ reconcile.Reconciler = &ReconcileBareMetalAsset{}

func TestMain(m *testing.M) {
	// AddToSchemes may be used to add all resources defined in the project to a Scheme
	var AddToSchemes runtime.SchemeBuilder
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, inventoryv1alpha1.SchemeBuilder.AddToScheme)

	if err := AddToSchemes.AddToScheme(scheme.Scheme); err != nil {
		klog.Errorf("Failed adding apis to scheme, %v", err)
		os.Exit(1)
	}

	if err := hivev1.AddToScheme(scheme.Scheme); err != nil {
		klog.Errorf("Failed adding hivev1 to scheme, %v", err)
		os.Exit(1)
	}
	exitVal := m.Run()
	os.Exit(exitVal)
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
			name:         "BareMetalAssetNotFound",
			existingObjs: []runtime.Object{},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
		},
		{
			name:         "BareMetalAssetFound",
			existingObjs: []runtime.Object{newBMA()},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
			requeue: true,
		},
		{
			name: "ClusterDeploymentsNotFound",
			existingObjs: []runtime.Object{
				newBMAWithClusterDeployment(),
				newSecret(),
			},
			expectedErrorType: fmt.Errorf("clusterdeployments.hive.openshift.io \"%s\" not found", testName),
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
		},
		{
			name: "SyncSetInstancesNotFound",
			existingObjs: []runtime.Object{
				newBMAWithClusterDeployment(),
				newSecret(),
				newClusterDeployment(),
			},
			expectedErrorType: fmt.Errorf("no SyncSetInstances with label name %v and label value %v found", hiveconstants.SyncSetNameLabel, testName),
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
		},
		{
			name: "BareMetalAssetWithDeletionTimestampAndFinalizer",
			existingObjs: []runtime.Object{
				func() *inventoryv1alpha1.BareMetalAsset {
					bma := newBMA()
					bma.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
					bma.SetFinalizers([]string{BareMetalAssetFinalizer})
					return bma
				}(),
			},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			res, err := rbma.Reconcile(test.req)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, nil, nil)

			if test.requeue {
				assert.Equal(t, res, reconcile.Result{Requeue: false, RequeueAfter: time.Duration(60) * time.Second})
			} else {
				assert.Equal(t, res, reconcile.Result{Requeue: false, RequeueAfter: 0})
			}
		})
	}
}

func TestCheckAssetSecret(t *testing.T) {
	tests := []struct {
		name               string
		existingObjs       []runtime.Object
		expectedErrorType  error
		expectedConditions []conditionsv1.Condition
		bma                *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:              "SecretNotFound",
			existingObjs:      []runtime.Object{},
			expectedErrorType: bmaerrors.NewAssetSecretNotFoundError(testName, testNamespace),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionCredentialsFound,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name:         "SecretFound",
			existingObjs: []runtime.Object{newSecret()},
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionCredentialsFound,
				Status: corev1.ConditionTrue,
			}},
			bma: newBMA(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			err := rbma.checkAssetSecret(test.bma)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, test.expectedConditions, test.bma)
		})
	}
}

func TestEnsureLabels(t *testing.T) {
	tests := []struct {
		name              string
		existingObjs      []runtime.Object
		expectedErrorType error
		bma               *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:         "EnsureLabelsSuccess",
			existingObjs: []runtime.Object{newBMA()},
			bma:          newBMAWithClusterDeployment(),
		},
		{
			name:         "EnsureLabelsSuccessNoClusterDeployment",
			existingObjs: []runtime.Object{newBMA()},
			bma:          newBMA(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			err := rbma.ensureLabels(test.bma)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, nil, test.bma)
		})
	}
}

func TestCheckClusterDeployment(t *testing.T) {
	tests := []struct {
		name               string
		existingObjs       []runtime.Object
		expectedErrorType  error
		expectedConditions []conditionsv1.Condition
		bma                *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:              "No cluster specified",
			existingObjs:      []runtime.Object{},
			expectedErrorType: bmaerrors.NewNoClusterError(),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionClusterDeploymentFound,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name:              "ClusterDeploymentNotFound",
			existingObjs:      []runtime.Object{},
			expectedErrorType: fmt.Errorf("clusterdeployments.hive.openshift.io \"%s\" not found", testName),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionClusterDeploymentFound,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMAWithClusterDeployment(),
		},
		{
			name:         "ClusterDeploymentFound",
			existingObjs: []runtime.Object{newClusterDeployment()},
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionClusterDeploymentFound,
				Status: corev1.ConditionTrue,
			}},
			bma: newBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			err := rbma.checkClusterDeployment(test.bma)
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, test.expectedConditions, test.bma)
		})
	}
}

func TestEnsureHiveSyncSet(t *testing.T) {
	tests := []struct {
		name               string
		existingObjs       []runtime.Object
		expectedConditions []conditionsv1.Condition
		bma                *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:         "SyncSetCreate",
			existingObjs: []runtime.Object{},
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncStarted,
				Status: corev1.ConditionTrue,
			}},
			bma: newBMAWithClusterDeployment(),
		},
		{
			name: "SyncSetUpdate",
			existingObjs: []runtime.Object{func() *hivev1.SyncSet {
				return &hivev1.SyncSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testName,
						Namespace: testNamespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								Kind: testSSKind,
								Name: testName,
							},
						},
					},
				}
			}()},
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncStarted,
				Status: corev1.ConditionTrue,
			}},
			bma: newBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			err := rbma.ensureHiveSyncSet(test.bma)
			validateErrorAndStatusConditions(t, err, nil, test.expectedConditions, test.bma)

			syncSet := &hivev1.SyncSet{}
			syncSetError := rbma.client.Get(context.TODO(), types.NamespacedName{Name: testName, Namespace: testNamespace}, syncSet)

			assert.NoError(t, syncSetError)

			assert.Equal(t, syncSet.ObjectMeta.Labels[ClusterDeploymentNameLabel], test.bma.Spec.ClusterDeployment.Name)
			assert.Equal(t, syncSet.ObjectMeta.Labels[ClusterDeploymentNamespaceLabel], test.bma.Spec.ClusterDeployment.Namespace)
			assert.Equal(t, syncSet.ObjectMeta.Labels[testRoleLabel], string(test.bma.Spec.Role))

			if test.name != "SyncSetCreate" {
				assert.Equal(t, test.bma.Status.RelatedObjects[0].Kind, syncSet.TypeMeta.Kind)
				assert.Equal(t, test.bma.Status.RelatedObjects[0].Name, syncSet.Name)
				assert.Equal(t, test.bma.Status.RelatedObjects[0].APIVersion, syncSet.TypeMeta.APIVersion)
			}
		})
	}
}
func TestCheckHiveSyncSetInstance(t *testing.T) {
	tests := []struct {
		name               string
		existingObjs       []runtime.Object
		expectedErrorType  error
		expectedConditions []conditionsv1.Condition
		bma                *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:         "SyncSetInstanceNotFound",
			existingObjs: []runtime.Object{newBMA()},
			expectedErrorType: fmt.Errorf("no SyncSetInstances with label name %v and label value %v found",
				hiveconstants.SyncSetNameLabel, testName),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name:              "UnexpectedResourceCount",
			existingObjs:      []runtime.Object{newBMA(), newSyncSetInstance()},
			expectedErrorType: fmt.Errorf("unexpected number of resources found on SyncSetInstance status. Expected (1) Found (0)"),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name: "BareMetalHostResourceNotFound-Incorrect Kind",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := newSyncSetInstanceResources()
					ssi.Status.Resources = []hivev1.SyncStatus{
						{
							Kind: "AnInvalidKind",
						},
					}
					return ssi
				}(), newBMA(),
			},
			expectedErrorType: fmt.Errorf("unexpected resource found in SyncSetInstance status. "+
				"Expected (Kind: %v APIVersion: %v) Found (Kind: %v APIVersion: %v)",
				BareMetalHostKind, metal3v1alpha1.SchemeGroupVersion.String(), "AnInvalidKind", ""),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name: "BareMetalHostResourceNotFound-Incorrect APIVersion",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := newSyncSetInstanceResources()
					ssi.Status.Resources = []hivev1.SyncStatus{
						{
							APIVersion: "InvalidAPIVersion",
						},
					}
					return ssi
				}(), newBMA(),
			},
			expectedErrorType: fmt.Errorf("unexpected resource found in SyncSetInstance status. "+
				"Expected (Kind: %v APIVersion: %v) Found (Kind: %v APIVersion: %v)",
				BareMetalHostKind, metal3v1alpha1.SchemeGroupVersion.String(), "", "InvalidAPIVersion"),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name: "ResourceApplySuccessSyncCondition-SecretMissing",
			existingObjs: []runtime.Object{
				newSyncSetInstanceResouceApplySuccess(),
			},
			expectedErrorType: fmt.Errorf("unexpected number of secrets found on SyncSetInstance. Expected: (1) Actual: (0)"),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name: "ResourceApplyFailureSyncCondition",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := newSyncSetInstance()
					ssi.Status.Resources = []hivev1.SyncStatus{{
						APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
						Kind:       testBMHKind,
						Name:       testName,
						Conditions: []hivev1.SyncCondition{
							{
								Message: "Apply failed",
								Reason:  "ApplyFailed",
								Status:  corev1.ConditionFalse,
								Type:    hivev1.ApplyFailureSyncCondition,
							},
						},
					}}
					return ssi
				}(),
			},
			expectedErrorType: fmt.Errorf("get SyncSetInstance resource %s failed with message Apply failed", testName),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name: "SecretApplySuccessSyncCondition",
			existingObjs: []runtime.Object{func() *hivev1.SyncSetInstance {
				ssi := newSyncSetInstanceResouceApplySuccess()
				ssi.Status.Secrets = []hivev1.SyncStatus{
					{
						Name: testName,
						Conditions: []hivev1.SyncCondition{
							{
								Message: "Apply successful",
								Reason:  "ApplySucceeded",
								Status:  corev1.ConditionTrue,
								Type:    hivev1.ApplySuccessSyncCondition,
							},
						},
					},
				}
				return ssi
			}()},
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionTrue,
			}},
			bma: newBMA(),
		},
		{
			name: "SecretApplyFailureSyncCondition",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := newSyncSetInstanceResouceApplySuccess()
					ssi.Status.Secrets = []hivev1.SyncStatus{
						{
							Name: testName,
							Conditions: []hivev1.SyncCondition{
								{
									Message: "Apply failed",
									Reason:  "ApplyFailed",
									Status:  corev1.ConditionFalse,
									Type:    hivev1.ApplyFailureSyncCondition,
								},
							},
						},
					}
					return ssi
				}(),
			},
			expectedErrorType: fmt.Errorf("get SyncSetInstance resource %s failed with message Apply failed", testName),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
		{
			name: "MultipleSyncSetInstancesFound",
			existingObjs: []runtime.Object{
				newSyncSetInstance(),
				func() *hivev1.SyncSetInstance {
					ssi := newSyncSetInstance()
					ssi.Name = testNamespace
					return ssi
				}(),
			},
			expectedErrorType: fmt.Errorf("found multiple Hive SyncSetInstances with same label"),
			expectedConditions: []conditionsv1.Condition{{
				Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status: corev1.ConditionFalse,
			}},
			bma: newBMA(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			errb := rbma.checkHiveSyncSetInstance(test.bma)
			var err error
			if errb {
				err = fmt.Errorf("checkHiveSyncSetInstance exited with a value of %t", errb)
			}
			validateErrorAndStatusConditions(t, err, test.expectedErrorType, test.expectedConditions, test.bma)
		})
	}
}

func TestDeleteSyncSet(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []runtime.Object
		bma          *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:         "ClusterDeploymentWithEmptyNamespace",
			existingObjs: []runtime.Object{},
			bma:          newBMA(),
		},
		{
			name:         "ClusterDeploymentWithNamespace",
			existingObjs: []runtime.Object{},
			bma:          newBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rbma := newTestReconciler(test.existingObjs)
			_, err := rbma.deleteSyncSet(test.bma)
			validateErrorAndStatusConditions(t, err, nil, nil, test.bma)
		})
	}
}

func newBMA() *inventoryv1alpha1.BareMetalAsset {
	return &inventoryv1alpha1.BareMetalAsset{
		TypeMeta: metav1.TypeMeta{
			Kind: testBMHKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: inventoryv1alpha1.BareMetalAssetSpec{
			BMC: inventoryv1alpha1.BMCDetails{
				CredentialsName: testName,
			},
			Role: testRoleLabel,
		},
	}
}

func newBMAWithClusterDeployment() *inventoryv1alpha1.BareMetalAsset {
	bma := newBMA()
	bma.Spec.ClusterDeployment = metav1.ObjectMeta{
		Name:      testName,
		Namespace: testNamespace,
	}
	return bma
}

func newSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}
}

func newClusterDeployment() *hivev1.ClusterDeployment {
	cd := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}
	return cd
}

func newSyncSetInstance() *hivev1.SyncSetInstance {
	return &hivev1.SyncSetInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Labels:    map[string]string{hiveconstants.SyncSetNameLabel: testName},
		},
		Status: hivev1.SyncSetInstanceStatus{},
	}
}

func newSyncSetInstanceResources() *hivev1.SyncSetInstance {
	ssi := newSyncSetInstance()
	ssi.Status.Resources = []hivev1.SyncStatus{
		{
			APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
			Kind:       testBMHKind,
			Name:       testName,
		},
	}
	return ssi
}

func newSyncSetInstanceResouceApplySuccess() *hivev1.SyncSetInstance {
	ssi := newSyncSetInstance()
	ssi.Status.Resources = []hivev1.SyncStatus{{
		APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
		Kind:       testBMHKind,
		Name:       testName,
		Conditions: []hivev1.SyncCondition{
			{
				Message: "Apply successful",
				Reason:  "ApplySucceeded",
				Status:  corev1.ConditionTrue,
				Type:    hivev1.ApplySuccessSyncCondition,
			},
		},
	}}
	return ssi
}

func newTestReconciler(existingObjs []runtime.Object) *ReconcileBareMetalAsset {
	fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, existingObjs...)
	rbma := &ReconcileBareMetalAsset{
		client: fakeClient,
		scheme: scheme.Scheme,
	}
	return rbma
}

func validateErrorAndStatusConditions(t *testing.T, err error, expectedErrorType error,
	expectedConditions []conditionsv1.Condition, bma *inventoryv1alpha1.BareMetalAsset) {
	if expectedErrorType != nil {
		assert.EqualError(t, err, expectedErrorType.Error())
	} else {
		assert.NoError(t, err)
	}
	for _, condition := range expectedConditions {
		assert.True(t, conditionsv1.IsStatusConditionPresentAndEqual(bma.Status.Conditions, condition.Type, condition.Status))
	}
	if bma != nil {
		assert.Equal(t, len(expectedConditions), len(bma.Status.Conditions))
	}
}
