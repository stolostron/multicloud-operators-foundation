 package inventory

import (
	"os"
	"testing"
	"time"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	inventoryv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	bmaerrors "github.com/open-cluster-management/multicloud-operators-foundation/pkg/controller/inventory/errors"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	hiveconstants "github.com/openshift/hive/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"github.com/stretchr/testify/assert"
)

const (
	testName                 = "foo"
	testNamespace            = "bar"
	testBMHKind              = "BareMetalHost"
	testBMAKind              = "BareMetalAsset"
	testSSKind               = "SyncSet"
	testRole                 = "role"
	testBMAAPIVersion        = "inventory.open-cluster-management.io/v1alpha1"
	testSSOwnerRefAPIVersion = "hive.openshift.io/v1"
	testRoleLabel            = "metal3.io/role"
	testFinalizerLabel       = "baremetalasset.inventory.open-cluster-management.io"
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
		name         string
		existingObjs []runtime.Object
		bma          *inventoryv1alpha1.BareMetalAsset
		req          reconcile.Request
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
			existingObjs: []runtime.Object{NewBMA()},
			req: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testName,
					Namespace: testNamespace,
				},
			},
		},
		{
			name: "BareMetalAssetNoClusterSpecfied",
			existingObjs: []runtime.Object{NewBMAWithClusterDeployment(),
				NewSecret(),
				NewClusterDeployment(),
				NewSyncSet(),
				NewSyncSetInstanceSecretApplySuccess()},
			bma: NewBMAWithClusterDeployment(),
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
					bma := NewBMA()
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
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			res, err := rbma.Reconcile(test.req)
			if err != nil {
				assert.Error(t, err)
			}
			// Check the result of reconciliation to make sure it has the desired state.
			if !res.Requeue {
				assert.NoError(t, err)
			}
			instance := &inventoryv1alpha1.BareMetalAsset{}
			bmaInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, instance)

			switch test.name {
			case "BareMetalAssetNotFound":
				assert.Error(t, bmaInstanceError)
				assert.Equal(t, res, reconcile.Result{})
			case "BareMetalAssetFound":
				assert.NoError(t, bmaInstanceError)
				assert.Equal(t, res, reconcile.Result{Requeue: false, RequeueAfter: 60000000000})
				assert.Equal(t, instance.ObjectMeta.Finalizers[0], testFinalizerLabel)
				assert.True(t, conditionsv1.IsStatusConditionFalse(instance.Status.Conditions, inventoryv1alpha1.ConditionCredentialsFound))

			case "BareMetalAssetNoClusterSpecfied":
				assert.Equal(t, res, reconcile.Result{})
				secretRef, secretRefError := reference.GetReference(rbma.scheme, NewSecret())
				assert.NoError(t, secretRefError)
				assert.Equal(t, instance.ObjectMeta.Finalizers[0], testFinalizerLabel)
				assert.Equal(t, instance.Status.RelatedObjects[0], *secretRef)
				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionCredentialsFound))
				assert.Equal(t, instance.ObjectMeta.Labels[ClusterDeploymentNameLabel], test.bma.Spec.ClusterDeployment.Name)
				assert.Equal(t, instance.ObjectMeta.Labels[ClusterDeploymentNamespaceLabel], test.bma.Spec.ClusterDeployment.Namespace)

				secretInstance := &corev1.Secret{}
				secretInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, secretInstance)
				assert.NoError(t, secretInstanceError)

				secretWithOwnerRef := NewSecret()
				secretWithOwnerRef.OwnerReferences = []metav1.OwnerReference{
					metav1.OwnerReference{
						Kind:       testBMAKind,
						Name:       testName,
						APIVersion: testBMAAPIVersion,
					},
				}
				assert.Equal(t, secretInstance.OwnerReferences[0].Kind, secretWithOwnerRef.OwnerReferences[0].Kind)
				assert.Equal(t, secretInstance.OwnerReferences[0].Name, secretWithOwnerRef.OwnerReferences[0].Name)
				assert.Equal(t, secretInstance.OwnerReferences[0].APIVersion, secretWithOwnerRef.OwnerReferences[0].APIVersion)

				assert.Equal(t, instance.ObjectMeta.Labels[ClusterDeploymentNameLabel], test.bma.Spec.ClusterDeployment.Name)
				assert.Equal(t, instance.ObjectMeta.Labels[ClusterDeploymentNamespaceLabel], test.bma.Spec.ClusterDeployment.Namespace)

				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionClusterDeploymentFound))

				syncSetWithOwnerRef := NewSyncSet()
				syncSetWithOwnerRef.SetOwnerReferences([]metav1.OwnerReference{
					metav1.OwnerReference{
						Kind:       testSSKind,
						Name:       testName,
						APIVersion: testSSOwnerRefAPIVersion,
					},
				})
				assert.Equal(t, instance.Status.RelatedObjects[1].Kind, syncSetWithOwnerRef.OwnerReferences[0].Kind)
				assert.Equal(t, instance.Status.RelatedObjects[1].Name, syncSetWithOwnerRef.OwnerReferences[0].Name)
				assert.Equal(t, instance.Status.RelatedObjects[1].APIVersion, syncSetWithOwnerRef.OwnerReferences[0].APIVersion)

				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncStarted))
				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncCompleted))

			case "BareMetalAssetWithDeletionTimestampAndFinalizer":
				assert.Equal(t, res, reconcile.Result{})
				syncSet := &hivev1.SyncSet{}
				syncSetError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, syncSet)
				assert.Error(t, syncSetError)
				assert.NotContains(t, instance.ObjectMeta.Finalizers, testFinalizerLabel)
			}
		})
	}
}

func TestCheckAssetSecret(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []runtime.Object
		expectErr    bool
	}{
		{
			name:         "SecretNotFound",
			existingObjs: []runtime.Object{NewBMAWithConditionCredentialsFoundFalse()},
			expectErr:    true,
		},
		{
			name: "SecretFound",
			existingObjs: []runtime.Object{
				func() *inventoryv1alpha1.BareMetalAsset {
					bma := NewBMAWithRelatedObjects()
					bma.Status.Conditions = NewBMAWithConditionCredentialsFoundTrue().Status.Conditions
					return bma
				}(),
				NewSecret()},
			expectErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			err := rbma.checkAssetSecret(NewBMA())

			instance := &inventoryv1alpha1.BareMetalAsset{}
			bmaInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, instance)

			secretInstance := &corev1.Secret{}
			secretInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, secretInstance)

			assert.NoError(t, bmaInstanceError)

			if test.expectErr {
				assert.Error(t, err)
				assert.Error(t, secretInstanceError)
				assert.True(t, conditionsv1.IsStatusConditionFalse(instance.Status.Conditions, inventoryv1alpha1.ConditionCredentialsFound))
				assert.EqualError(t, err, bmaerrors.NewAssetSecretNotFoundError(instance.Spec.BMC.CredentialsName, instance.Namespace).Error())
			} else {
				assert.NoError(t, secretInstanceError)

				secretRef, secretRefError := reference.GetReference(rbma.scheme, NewSecret())
				assert.NoError(t, secretRefError)

				assert.Equal(t, instance.Status.RelatedObjects[1], *secretRef)
				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionCredentialsFound))

				err := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, secretInstance)
				assert.NoError(t, err)

				secretWithOwnerRef := NewSecret()
				secretWithOwnerRef.OwnerReferences = []metav1.OwnerReference{
					metav1.OwnerReference{
						Kind:       testBMAKind,
						Name:       testName,
						APIVersion: testBMAAPIVersion,
					},
				}
				assert.Equal(t, secretInstance.OwnerReferences[0].Kind, secretWithOwnerRef.OwnerReferences[0].Kind)
				assert.Equal(t, secretInstance.OwnerReferences[0].Name, secretWithOwnerRef.OwnerReferences[0].Name)
				assert.Equal(t, secretInstance.OwnerReferences[0].APIVersion, secretWithOwnerRef.OwnerReferences[0].APIVersion)
			}
		})
	}
}

func TestEnsureLabels(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []runtime.Object
		bma          *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:         "EnsureLabels",
			existingObjs: []runtime.Object{NewBMA()},
			bma:          NewBMA(),
		},
		{
			name:         "EnsureLabelsWithClusterDpeloymentInfo",
			existingObjs: []runtime.Object{NewBMA()},
			bma:          NewBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			err := rbma.ensureLabels(test.bma)

			assert.NoError(t, err)

			instance := &inventoryv1alpha1.BareMetalAsset{}
			instanceErr := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, instance)

			assert.NoError(t, instanceErr)
			assert.Equal(t, instance.ObjectMeta.Labels[ClusterDeploymentNameLabel], test.bma.Spec.ClusterDeployment.Name)
			assert.Equal(t, instance.ObjectMeta.Labels[ClusterDeploymentNamespaceLabel], test.bma.Spec.ClusterDeployment.Namespace)
			assert.Equal(t, instance.ObjectMeta.Labels[testRoleLabel], string(test.bma.Spec.Role))
		})
	}
}

func TestCheckClusterDeployment(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []runtime.Object
		expectErr    bool
		bma          *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name:         "No cluster specified",
			existingObjs: []runtime.Object{NewBMAWithConditionClusterDeploymentFoundFalse()},
			expectErr:    true,
			bma: func() *inventoryv1alpha1.BareMetalAsset {
				bma := NewBMAWithClusterDeployment()
				bma.Spec.ClusterDeployment.Name = ""
				return bma
			}(),
		},
		{
			name:         "BMA With RelatedObjects",
			existingObjs: []runtime.Object{NewBMAWithConditionClusterDeploymentFoundFalse()},
			expectErr:    true,
			bma:          NewBMAWithRelatedObjects(),
		},
		{
			name:         "ClusterDeploymentNotFound",
			existingObjs: []runtime.Object{NewBMAWithConditionClusterDeploymentFoundFalse()},
			expectErr:    true,
			bma:          NewBMAWithClusterDeployment(),
		},
		{
			name:         "ClusterDeploymentFound",
			existingObjs: []runtime.Object{NewClusterDeployment(), NewBMAWithConditionClusterDeploymentFoundTrue()},
			expectErr:    false,
			bma:          NewBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			err := rbma.checkClusterDeployment(test.bma)

			instance := &inventoryv1alpha1.BareMetalAsset{}
			bmaInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, instance)

			assert.NoError(t, bmaInstanceError)

			if test.expectErr {
				assert.Error(t, err)
				switch test.name {
				case "No cluster specified", "BMA With RelatedObjects":
					assert.True(t, conditionsv1.IsStatusConditionFalse(instance.Status.Conditions, inventoryv1alpha1.ConditionClusterDeploymentFound))

					assert.NotContains(t, instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncStarted)
					assert.NotContains(t, instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncCompleted)

					syncSet := &hivev1.SyncSet{}
					syncSetError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, syncSet)

					assert.Error(t, syncSetError)
					assert.NotContains(t, instance.Status.RelatedObjects, NewSyncSet())
					assert.EqualError(t, err, bmaerrors.NewNoClusterError().Error())

				case "ClusterDeploymentNotFound":
					cdInstance := &hivev1.ClusterDeployment{}
					cdInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testName}, cdInstance)

					assert.Error(t, cdInstanceError)
					assert.True(t, conditionsv1.IsStatusConditionFalse(instance.Status.Conditions, inventoryv1alpha1.ConditionClusterDeploymentFound))
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionClusterDeploymentFound))
			}
		})
	}
}

func TestEnsureHiveSyncSet(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []runtime.Object
		bma          *inventoryv1alpha1.BareMetalAsset
	}{
		{
			name: "SyncSetNotFound",
			existingObjs: []runtime.Object{
				func() *inventoryv1alpha1.BareMetalAsset {
					bma := NewBMAWithConditionAssetSyncStarted()
					bma.Status.Conditions[0].Status = corev1.ConditionTrue
					return bma
				}()},
		},
		{
			name: "SyncSetFound",
			existingObjs: []runtime.Object{
				func() *inventoryv1alpha1.BareMetalAsset {
					bma := NewBMAWithRelatedObjects()
					bma.Status.Conditions = NewBMAWithConditionAssetSyncStartedTrue().Status.Conditions
					return bma
				}(),
				NewSyncSet()},
			bma: NewBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			err := rbma.ensureHiveSyncSet(NewBMAWithClusterDeployment())

			assert.NoError(t, err)

			instance := &inventoryv1alpha1.BareMetalAsset{}
			bmaInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, instance)

			assert.NoError(t, bmaInstanceError)

			syncSet := &hivev1.SyncSet{}
			syncSetError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, syncSet)

			assert.NoError(t, syncSetError)

			switch test.name {
			case "SyncSetNotFound":
				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncStarted))

			case "SyncSetFound":
				syncSetWithOwnerRef := NewSyncSet()
				syncSetWithOwnerRef.SetOwnerReferences([]metav1.OwnerReference{
					metav1.OwnerReference{
						Kind:       testSSKind,
						Name:       testName,
						APIVersion: testSSOwnerRefAPIVersion,
					},
				})
				assert.Equal(t, instance.Status.RelatedObjects[0].Kind, syncSetWithOwnerRef.OwnerReferences[0].Kind)
				assert.Equal(t, instance.Status.RelatedObjects[0].Name, syncSetWithOwnerRef.OwnerReferences[0].Name)
				assert.Equal(t, instance.Status.RelatedObjects[0].APIVersion, syncSetWithOwnerRef.OwnerReferences[0].APIVersion)

				assert.Equal(t, syncSet.ObjectMeta.Labels[ClusterDeploymentNameLabel], test.bma.Spec.ClusterDeployment.Name)
				assert.Equal(t, syncSet.ObjectMeta.Labels[ClusterDeploymentNamespaceLabel], test.bma.Spec.ClusterDeployment.Namespace)
				assert.Equal(t, syncSet.ObjectMeta.Labels[testRoleLabel], string(test.bma.Spec.Role))

				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncStarted))

			}
		})
	}
}

func TestCheckHiveSyncSetInstance(t *testing.T) {
	tests := []struct {
		name         string
		existingObjs []runtime.Object
	}{
		{
			name:         "SyncSetInstanceNotFound",
			existingObjs: []runtime.Object{NewBMAWithConditionAssetSyncCompletedFalse()},
		},
		{
			name:         "UnexpectedResourceCount",
			existingObjs: []runtime.Object{NewBMAWithConditionAssetSyncCompletedFalse(), NewSyncSetInstance()},
		},
		{
			name: "BareMetalHostResourceNotFound-Incorrect Kind",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := NewSyncSetInstanceResources()
					ssi.Status.Resources = []hivev1.SyncStatus{
						hivev1.SyncStatus{
							Kind: "AnInvalidKind",
						},
					}
					return ssi
				}(), NewBMAWithConditionAssetSyncCompletedFalse(),
			},
		},
		{
			name: "BareMetalHostResourceNotFound-Incorrect APIVersion",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := NewSyncSetInstanceResources()
					ssi.Status.Resources = []hivev1.SyncStatus{
						hivev1.SyncStatus{
							APIVersion: "InvalidAPIVersion",
						},
					}
					return ssi
				}(), NewBMAWithConditionAssetSyncCompletedFalse(),
			},
		},
		{
			name: "ResourceApplySuccessSyncCondition-SecretMissing",
			existingObjs: []runtime.Object{
				NewSyncSetInstanceResouceApplySuccess(),
				NewBMAWithConditionAssetSyncCompletedFalse(),
			},
		},
		{
			name: "ResourceApplyFailureSyncCondition",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := NewSyncSetInstance()
					ssi.Status.Resources = []hivev1.SyncStatus{hivev1.SyncStatus{
						APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
						Kind:       testBMHKind,
						Name:       testName,
						Conditions: []hivev1.SyncCondition{
							hivev1.SyncCondition{
								Message: "Apply failed",
								Reason:  "ApplyFailed",
								Status:  corev1.ConditionFalse,
								Type:    hivev1.ApplyFailureSyncCondition,
							},
						},
					}}
					return ssi
				}(),
				NewBMAWithConditionAssetSyncCompletedFalse(),
			},
		},
		{
			name: "SecretApplySuccessSyncCondition",
			existingObjs: []runtime.Object{
				NewSyncSetInstanceSecretApplySuccess(),
				NewBMAWithConditionAssetSyncCompletedTrue(),
			},
		},
		{
			name: "SecretApplyFailureSyncCondition",
			existingObjs: []runtime.Object{
				func() *hivev1.SyncSetInstance {
					ssi := NewSyncSetInstanceResouceApplySuccess()
					ssi.Status.Secrets = []hivev1.SyncStatus{
						hivev1.SyncStatus{
							Name: testName,
							Conditions: []hivev1.SyncCondition{
								hivev1.SyncCondition{
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
				NewBMAWithConditionAssetSyncCompletedFalse(),
			},
		},
		{
			name: "MultipleSyncSetInstancesFound",
			existingObjs: []runtime.Object{NewSyncSetInstance(),
				func() *hivev1.SyncSetInstance {
					ssi := NewSyncSetInstance()
					ssi.Name = testNamespace
					return ssi
				}(),
				NewBMAWithConditionAssetSyncCompletedFalse(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			err := rbma.checkHiveSyncSetInstance(NewBMA())

			instance := &inventoryv1alpha1.BareMetalAsset{}
			bmaInstanceError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, instance)

			assert.NoError(t, bmaInstanceError)

			syncSetInstanceList := &hivev1.SyncSetInstanceList{}
			syncSetInstanceListError := rbma.client.List(nil, syncSetInstanceList, client.MatchingLabels{hiveconstants.SyncSetNameLabel: instance.Name})

			assert.NoError(t, syncSetInstanceListError)

			switch test.name {
			case "SyncSetInstanceNotFound", "UnexpectedResourceCount", "BareMetalHostResourceNotFound-Incorrect Kind",
				"BareMetalHostResourceNotFound-Incorrect APIVersion", "ResourceApplySuccessSyncCondition-SecretMissing",
				"ResourceApplyFailureSyncCondition", "SecretApplyFailureSyncCondition", "MultipleSyncSetInstancesFound":
				assert.Error(t, err)
				assert.True(t, conditionsv1.IsStatusConditionFalse(instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncCompleted))
			case "SecretApplySuccessSyncCondition":
				assert.NoError(t, err)
				assert.True(t, conditionsv1.IsStatusConditionTrue(instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncCompleted))
			}

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
			bma:          NewBMA(),
		},
		{
			name:         "SyncSetNotFound",
			existingObjs: []runtime.Object{NewBMAWithClusterDeployment()},
			bma:          NewBMAWithClusterDeployment(),
		},
		{
			name: "SyncSetFound",
			existingObjs: []runtime.Object{
				NewBMAWithClusterDeployment(),
				NewSyncSet()},
			bma: NewBMAWithClusterDeployment(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClientWithScheme(scheme.Scheme, test.existingObjs...)
			rbma := &ReconcileBareMetalAsset{
				client: fakeClient,
				scheme: scheme.Scheme,
			}
			err := rbma.deleteSyncSet(test.bma)
			switch test.name {
			case "ClusterDeploymentWithEmptyNamespace", "SyncSetNotFound":
				assert.NoError(t, err)

			case "SyncSetFound":
				syncSet := &hivev1.SyncSet{}
				syncSetError := rbma.client.Get(nil, types.NamespacedName{Name: testName, Namespace: testNamespace}, syncSet)
				assert.Error(t, syncSetError)
			}
		})
	}
}

func NewBMA() *inventoryv1alpha1.BareMetalAsset {
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

func NewBMAWithConditionAssetSyncStarted() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMA()
	bma.Status.Conditions = []conditionsv1.Condition{
		conditionsv1.Condition{
			Type: inventoryv1alpha1.ConditionAssetSyncStarted,
		},
	}
	return bma
}

func NewBMAWithConditionAssetSyncStartedTrue() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionAssetSyncStarted()
	bma.Status.Conditions[0].Status = corev1.ConditionTrue
	return bma
}

func NewBMAWithConditionClusterDeploymentFound() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMA()
	bma.Status.Conditions = []conditionsv1.Condition{
		conditionsv1.Condition{
			Type: inventoryv1alpha1.ConditionClusterDeploymentFound,
		},
	}
	return bma
}

func NewBMAWithConditionClusterDeploymentFoundTrue() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionClusterDeploymentFound()
	bma.Status.Conditions[0].Status = corev1.ConditionTrue
	return bma
}

func NewBMAWithConditionClusterDeploymentFoundFalse() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionClusterDeploymentFound()
	bma.Status.Conditions[0].Status = corev1.ConditionFalse
	return bma
}

func NewBMAWithConditionCredentialsFound() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMA()
	bma.Status.Conditions = []conditionsv1.Condition{
		conditionsv1.Condition{
			Type: inventoryv1alpha1.ConditionCredentialsFound,
		},
	}
	return bma
}

func NewBMAWithConditionCredentialsFoundTrue() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionCredentialsFound()
	bma.Status.Conditions[0].Status = corev1.ConditionTrue
	return bma
}

func NewBMAWithConditionCredentialsFoundFalse() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionCredentialsFound()
	bma.Status.Conditions[0].Status = corev1.ConditionFalse
	return bma
}

func NewBMAWithConditionAssetSyncCompleted() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMA()
	bma.Status.Conditions = []conditionsv1.Condition{
		conditionsv1.Condition{
			Type: inventoryv1alpha1.ConditionAssetSyncCompleted,
		},
	}
	return bma
}

func NewBMAWithConditionAssetSyncCompletedTrue() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionAssetSyncCompleted()
	bma.Status.Conditions[0].Status = corev1.ConditionTrue
	return bma
}

func NewBMAWithConditionAssetSyncCompletedFalse() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMAWithConditionAssetSyncCompleted()
	bma.Status.Conditions[0].Status = corev1.ConditionFalse
	return bma
}

func NewBMAWithRelatedObjects() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMA()
	bma.Status.RelatedObjects = []corev1.ObjectReference{
		corev1.ObjectReference{
			Name:       testName,
			Kind:       testSSKind,
			APIVersion: hivev1.SchemeGroupVersion.String(),
		},
		corev1.ObjectReference{
			Name:       testName,
			Namespace:  testNamespace,
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}
	return bma
}

func NewBMAWithClusterDeployment() *inventoryv1alpha1.BareMetalAsset {
	bma := NewBMA()
	bma.Spec.ClusterDeployment = metav1.ObjectMeta{
		Name:      testName,
		Namespace: testNamespace,
	}
	return bma
}

func NewSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}
}

func NewClusterDeployment() *hivev1.ClusterDeployment {
	cd := &hivev1.ClusterDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}
	return cd
}

func NewSyncSet() *hivev1.SyncSet {
	return &hivev1.SyncSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
	}
}

func NewSyncSetInstance() *hivev1.SyncSetInstance {
	return &hivev1.SyncSetInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
			Labels:    map[string]string{hiveconstants.SyncSetNameLabel: testName},
		},
		Status: hivev1.SyncSetInstanceStatus{},
	}
}

func NewSyncSetInstanceResources() *hivev1.SyncSetInstance {
	ssi := NewSyncSetInstance()
	ssi.Status.Resources = []hivev1.SyncStatus{
		hivev1.SyncStatus{
			APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
			Kind:       testBMHKind,
			Name:       testName,
		},
	}
	return ssi
}

func NewSyncSetInstanceResouceApplySuccess() *hivev1.SyncSetInstance {
	ssi := NewSyncSetInstance()
	ssi.Status.Resources = []hivev1.SyncStatus{hivev1.SyncStatus{
		APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
		Kind:       testBMHKind,
		Name:       testName,
		Conditions: []hivev1.SyncCondition{
			hivev1.SyncCondition{
				Message: "Apply successful",
				Reason:  "ApplySucceeded",
				Status:  corev1.ConditionTrue,
				Type:    hivev1.ApplySuccessSyncCondition,
			},
		},
	}}
	return ssi
}

func NewSyncSetInstanceSecretApplySuccess() *hivev1.SyncSetInstance {
	ssi := NewSyncSetInstanceResouceApplySuccess()
	ssi.Status.Secrets = []hivev1.SyncStatus{
		hivev1.SyncStatus{
			Name: testName,
			Conditions: []hivev1.SyncCondition{
				hivev1.SyncCondition{
					Message: "Apply successful",
					Reason:  "ApplySucceeded",
					Status:  corev1.ConditionTrue,
					Type:    hivev1.ApplySuccessSyncCondition,
				},
			},
		},
	}
	return ssi
}