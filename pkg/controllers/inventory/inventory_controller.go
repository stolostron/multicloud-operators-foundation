package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/util/retry"
	"reflect"
	"strings"
	"time"

	metal3v1alpha1 "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	inventoryv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/inventory/v1alpha1"
	bmaerrors "github.com/open-cluster-management/multicloud-operators-foundation/pkg/controllers/inventory/errors"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	hiveinternalv1alpha1 "github.com/openshift/hive/pkg/apis/hiveinternal/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	k8slabels "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
)

const (
	// RoleLabel is the key name for the role label associated with the asset
	RoleLabel = "metal3.io/role"
	// ClusterDeploymentNameLabel is the key name for the name label associated with the asset's clusterDeployment
	ClusterDeploymentNameLabel = "metal3.io/cluster-deployment-name"
	// ClusterDeploymentNamespaceLabel is the key name for the namespace label associated with the asset's clusterDeployment
	ClusterDeploymentNamespaceLabel = "metal3.io/cluster-deployment-namespace"
	// BareMetalHostKind contains the value of kind BareMetalHost
	BareMetalHostKind = "BareMetalHost"
)

const (
	// assetSecretRequeueAfter specifies the amount of time, in seconds, before requeue
	assetSecretRequeueAfter int = 60
)

func SetupWithManager(mgr manager.Manager) error {
	if err := addBMAReconciler(mgr, newBMAReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create baremetalasset controller, %v", err)
		return err
	}
	if err := addCDReconciler(mgr, newCDReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create baremetalasset controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newBMAReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBareMetalAsset{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func addBMAReconciler(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("baremetalasset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource BareMetalAsset
	err = c.Watch(&source.Kind{Type: &inventoryv1alpha1.BareMetalAsset{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to SyncSets and requeue BareMetalAssets with the name and matching cluster-deployment-namespace label
	// (which is also the syncset namespace)
	err = c.Watch(
		&source.Kind{Type: &hivev1.SyncSet{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				syncSet, ok := a.Object.(*hivev1.SyncSet)
				if !ok {
					// not a SyncSet, returning empty
					klog.Error("SyncSet handler received non-SyncSet object")
					return []reconcile.Request{}
				}
				bmas := &inventoryv1alpha1.BareMetalAssetList{}
				err := mgr.GetClient().List(context.TODO(), bmas,
					client.MatchingLabels{
						ClusterDeploymentNamespaceLabel: syncSet.Namespace,
					})
				if err != nil {
					klog.Errorf("Could not list BareMetalAsset %v with label %v=%v, %v",
						syncSet.Name, ClusterDeploymentNamespaceLabel, syncSet.Namespace, err)
				}
				var requests []reconcile.Request
				for _, bma := range bmas.Items {
					if syncSet.Name == bma.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      bma.Name,
								Namespace: bma.Namespace,
							},
						})
					}
				}
				return requests
			}),
		})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterSync
	err = c.Watch(
		&source.Kind{Type: &hiveinternalv1alpha1.ClusterSync{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				clusterSync, ok := a.Object.(*hiveinternalv1alpha1.ClusterSync)
				if !ok {
					// not a ClusterSync, returning empty
					klog.Error("ClusterSync handler received non-ClusterSync object")
					return []reconcile.Request{}
				}
				bmas := &inventoryv1alpha1.BareMetalAssetList{}
				err := mgr.GetClient().List(context.TODO(), bmas, client.InNamespace(clusterSync.Namespace))
				if err != nil {
					klog.Error("Could not list BareMetalAssets", err)
				}
				var requests []reconcile.Request
				for _, bma := range bmas.Items {
					if bma.Spec.ClusterDeployment.Name == clusterSync.Name {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      bma.Name,
								Namespace: bma.Namespace,
							},
						})
					}
				}
				return requests
			}),
		})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterDeployments and requeue BareMetalAssets with labels set to
	// ClusterDeployment's name (which is expected to be the clusterName)
	err = c.Watch(
		&source.Kind{Type: &hivev1.ClusterDeployment{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				clusterDeployment, ok := a.Object.(*hivev1.ClusterDeployment)
				if !ok {
					// not a Deployment, returning empty
					klog.Error("ClusterDeployment handler received non-ClusterDeployment object")
					return []reconcile.Request{}
				}
				bmas := &inventoryv1alpha1.BareMetalAssetList{}
				err := mgr.GetClient().List(context.TODO(), bmas,
					client.MatchingLabels{
						ClusterDeploymentNameLabel:      clusterDeployment.Name,
						ClusterDeploymentNamespaceLabel: clusterDeployment.Namespace,
					})
				if err != nil {
					klog.Errorf("could not list BareMetalAssets with label %v=%v, %v",
						ClusterDeploymentNameLabel, clusterDeployment.Name, err)
				}
				var requests []reconcile.Request
				for _, bma := range bmas.Items {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      bma.Name,
							Namespace: bma.Namespace,
						},
					})
				}
				return requests
			}),
		})

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBareMetalAsset implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBareMetalAsset{}

// ReconcileBareMetalAsset reconciles a BareMetalAsset object
type ReconcileBareMetalAsset struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a BareMetalAsset object and makes changes based on the state read
// and what is in the BareMetalAsset.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBareMetalAsset) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling BareMetalAsset")

	// Fetch the BareMetalAsset instance
	instance := &inventoryv1alpha1.BareMetalAsset{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if instance.GetDeletionTimestamp().IsZero() {
		if !contains(instance.GetFinalizers(), BareMetalAssetFinalizer) {
			klog.Info("Finalizer not found for BareMetalAsset. Adding finalizer")
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, BareMetalAssetFinalizer)
			if err := r.client.Update(context.TODO(), instance); err != nil {
				klog.Errorf("Failed to add finalizer to baremetalasset, %v", err)
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if contains(instance.GetFinalizers(), BareMetalAssetFinalizer) {
			return r.deleteSyncSet(instance)
		}
		return reconcile.Result{}, nil
	}

	for _, f := range []func(*inventoryv1alpha1.BareMetalAsset) error{
		r.ensureLabels,
		r.checkAssetSecret,
		r.cleanupOldHiveSyncSet,
		r.checkClusterDeployment,
		r.ensureHiveSyncSet,
	} {
		err = f(instance)
		if err != nil {
			switch {
			case bmaerrors.IsNoClusterError(err):
				klog.Info("No cluster specified")
				return reconcile.Result{}, r.updateStatus(instance)
			case bmaerrors.IsAssetSecretNotFoundError(err):
				// since we won't be notified when the secret is created, requeue after some time
				klog.Infof("Secret not found, RequeueAfter.Duration %v seconds", assetSecretRequeueAfter)
				return reconcile.Result{RequeueAfter: time.Duration(assetSecretRequeueAfter) * time.Second},
					r.updateStatus(instance)
			}

			klog.Errorf("Failed reconcile, %v", err)
			if statusErr := r.updateStatus(instance); statusErr != nil {
				klog.Errorf("Failed to update status, %v", statusErr)
			}

			return reconcile.Result{}, err
		}
	}

	klog.Info("BareMetalAsset Reconciled")
	return reconcile.Result{}, r.updateStatus(instance)
}

func (r *ReconcileBareMetalAsset) updateStatus(instance *inventoryv1alpha1.BareMetalAsset) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		newInstance := &inventoryv1alpha1.BareMetalAsset{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, newInstance)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if equality.Semantic.DeepEqual(newInstance.Status, instance.Status) {
			return nil
		}
		newInstance.Status = instance.Status
		return r.client.Status().Update(context.TODO(), newInstance)
	})
	return err
}

// checkAssetSecret verifies that we can find the secret listed in the BareMetalAsset
func (r *ReconcileBareMetalAsset) checkAssetSecret(instance *inventoryv1alpha1.BareMetalAsset) error {
	secretName := instance.Spec.BMC.CredentialsName

	secret := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: instance.Namespace}, secret)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Errorf("Secret (%s/%s) not found, %v", instance.Namespace, secretName, err)
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    inventoryv1alpha1.ConditionCredentialsFound,
				Status:  metav1.ConditionFalse,
				Reason:  "SecretNotFound",
				Message: err.Error(),
			})
			return bmaerrors.NewAssetSecretNotFoundError(secretName, instance.Namespace)
		}
		return err
	}

	// add secret reference to status
	secretRef, err := reference.GetReference(r.scheme, secret)
	if err != nil {
		klog.Errorf("Failed to get reference from secret, %v", err)
		return err
	}
	if err := objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *secretRef); err != nil {
		klog.Errorf("Failed to set reference, %v", err)
		return err
	}

	// add condition to status
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    inventoryv1alpha1.ConditionCredentialsFound,
		Status:  metav1.ConditionTrue,
		Reason:  "SecretFound",
		Message: fmt.Sprintf("A secret with the name %v in namespace %v was found", secretName, instance.Namespace),
	})

	// Set BaremetalAsset instance as the owner and controller
	if secret.OwnerReferences == nil || len(secret.OwnerReferences) == 0 {
		if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
			klog.Errorf("Failed to set ControllerReference, %v", err)
			return err
		}
		if err := r.client.Update(context.TODO(), secret); err != nil {
			klog.Errorf("Failed to update secret with OwnerReferences, %v", err)
			return err
		}
	}
	return nil
}

func (r *ReconcileBareMetalAsset) ensureLabels(instance *inventoryv1alpha1.BareMetalAsset) error {
	labels := k8slabels.CloneAndAddLabel(instance.Labels, ClusterDeploymentNameLabel, instance.Spec.ClusterDeployment.Name)
	labels = k8slabels.AddLabel(labels, ClusterDeploymentNamespaceLabel, instance.Spec.ClusterDeployment.Namespace)
	labels = k8slabels.AddLabel(labels, RoleLabel, string(instance.Spec.Role))

	if !reflect.DeepEqual(labels, instance.Labels) {
		instance.Labels = labels
		return r.client.Update(context.TODO(), instance)
	}
	return nil
}

// checkClusterDeployment verifies that we can find the ClusterDeployment specified in the BareMetalAsset
func (r *ReconcileBareMetalAsset) checkClusterDeployment(instance *inventoryv1alpha1.BareMetalAsset) error {
	clusterDeploymentName := instance.Spec.ClusterDeployment.Name
	clusterDeploymentNamespace := instance.Spec.ClusterDeployment.Namespace

	// if the clusterDeploymentName is not specified, we need to handle the possibility
	// that it has been removed from the spec
	if clusterDeploymentName == "" {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionClusterDeploymentFound,
			Status:  metav1.ConditionFalse,
			Reason:  "NoneSpecified",
			Message: "No cluster deployment specified",
		})
		meta.RemoveStatusCondition(&instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncStarted)
		meta.RemoveStatusCondition(&instance.Status.Conditions, inventoryv1alpha1.ConditionAssetSyncCompleted)

		return bmaerrors.NewNoClusterError()
	}

	// If a clusterDeployment is specified, we need to find it
	cd := &hivev1.ClusterDeployment{}
	err := r.client.Get(
		context.TODO(), types.NamespacedName{Name: clusterDeploymentName, Namespace: clusterDeploymentNamespace}, cd)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Errorf("ClusterDeployment (%s/%s) not found, %v", clusterDeploymentNamespace, clusterDeploymentName, err)
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    inventoryv1alpha1.ConditionClusterDeploymentFound,
				Status:  metav1.ConditionFalse,
				Reason:  "ClusterDeploymentNotFound",
				Message: err.Error(),
			})
			return err
		}
		return err
	}

	// add condition
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    inventoryv1alpha1.ConditionClusterDeploymentFound,
		Status:  metav1.ConditionTrue,
		Reason:  "ClusterDeploymentFound",
		Message: fmt.Sprintf("A ClusterDeployment with the name %v in namespace %v was found", cd.Name, cd.Namespace),
	})

	return nil
}

func (r *ReconcileBareMetalAsset) ensureHiveSyncSet(instance *inventoryv1alpha1.BareMetalAsset) error {
	assetSyncCompleted := r.checkHiveClusterSync(instance)
	hsc := r.newHiveSyncSet(instance, assetSyncCompleted)
	found := &hivev1.SyncSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: hsc.Name, Namespace: hsc.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			err := r.client.Create(context.TODO(), hsc)
			if err != nil {
				klog.Errorf("Failed to create Hive SyncSet, %v", err)
				meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
					Type:    inventoryv1alpha1.ConditionAssetSyncStarted,
					Status:  metav1.ConditionFalse,
					Reason:  "SyncSetCreationFailed",
					Message: "Failed to create SyncSet",
				})
				return err
			}

			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    inventoryv1alpha1.ConditionAssetSyncStarted,
				Status:  metav1.ConditionTrue,
				Reason:  "SyncSetCreated",
				Message: "SyncSet created successfully",
			})
			return nil
		}
		// other error. fail reconcile
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionAssetSyncStarted,
			Status:  metav1.ConditionFalse,
			Reason:  "SyncSetGetFailed",
			Message: "Failed to get SyncSet",
		})
		klog.Errorf("Failed to get Hive SyncSet (%s/%s), %v", hsc.Namespace, hsc.Name, err)
		return err
	}
	// rebuild the expected SyncSet if the one we found is missing Resources
	// because it means we have successfully applied
	if len(found.Spec.SyncSetCommonSpec.Resources) == 0 {
		hsc = r.newHiveSyncSet(instance, true)
	}

	// Add SyncSet to related objects
	hscRef, err := reference.GetReference(r.scheme, found)
	if err != nil {
		klog.Errorf("Failed to get reference from SyncSet, %v", err)
		return err
	}
	if err := objectreferencesv1.SetObjectReference(&instance.Status.RelatedObjects, *hscRef); err != nil {
		klog.Errorf("Failed to set reference, %v", err)
		return err
	}

	// Add labels to copy for comparison to minimize updates
	labels := k8slabels.CloneAndAddLabel(found.Labels, ClusterDeploymentNameLabel, instance.Spec.ClusterDeployment.Name)
	labels = k8slabels.AddLabel(labels, ClusterDeploymentNamespaceLabel, instance.Spec.ClusterDeployment.Namespace)
	labels = k8slabels.AddLabel(labels, RoleLabel, string(instance.Spec.Role))

	// Update Hive SyncSet CR if it is not in the desired state
	if !reflect.DeepEqual(hsc.Spec, found.Spec) || !reflect.DeepEqual(labels, found.Labels) {
		klog.Infof("Updating Hive SyncSet (%s/%s)", hsc.Namespace, hsc.Name)

		found.Labels = labels
		found.Spec = hsc.Spec

		err := r.client.Update(context.TODO(), found)
		if err != nil {
			klog.Errorf("Failed to update Hive SyncSet (%s/%s), %v", hsc.Namespace, hsc.Name, err)
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    inventoryv1alpha1.ConditionAssetSyncStarted,
				Status:  metav1.ConditionFalse,
				Reason:  "SyncSetUpdateFailed",
				Message: "Failed to update SyncSet",
			})
			return err
		}
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionAssetSyncStarted,
			Status:  metav1.ConditionTrue,
			Reason:  "SyncSetUpdated",
			Message: "SyncSet updated successfully",
		})
	}
	return nil
}

func (r *ReconcileBareMetalAsset) newHiveSyncSet(instance *inventoryv1alpha1.BareMetalAsset, assetSyncCompleted bool) *hivev1.SyncSet {
	bmhJSON, err := newBareMetalHost(instance, assetSyncCompleted)
	if err != nil {
		klog.Errorf("Error marshaling baremetalhost, %v", err)
		return nil
	}

	hsc := &hivev1.SyncSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SyncSet",
			APIVersion: "hive.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Spec.ClusterDeployment.Namespace, // syncset should be created in the same namespace as the clusterdeployment
			Labels: map[string]string{
				ClusterDeploymentNameLabel:      instance.Spec.ClusterDeployment.Name,
				ClusterDeploymentNamespaceLabel: instance.Spec.ClusterDeployment.Namespace,
				RoleLabel:                       string(instance.Spec.Role),
			},
		},
		Spec: hivev1.SyncSetSpec{
			SyncSetCommonSpec: hivev1.SyncSetCommonSpec{
				Resources: []runtime.RawExtension{
					{
						Raw: bmhJSON,
					},
				},
				Patches:           []hivev1.SyncObjectPatch{},
				ResourceApplyMode: hivev1.SyncResourceApplyMode,
				Secrets: []hivev1.SecretMapping{
					{
						SourceRef: hivev1.SecretReference{
							Name:      instance.Spec.BMC.CredentialsName,
							Namespace: instance.Namespace,
						},
						TargetRef: hivev1.SecretReference{
							Name:      instance.Spec.BMC.CredentialsName,
							Namespace: inventoryv1alpha1.ManagedClusterResourceNamespace,
						},
					},
				},
			},
			ClusterDeploymentRefs: []corev1.LocalObjectReference{
				{
					Name: instance.Spec.ClusterDeployment.Name,
				},
			},
		},
	}

	if assetSyncCompleted {
		// Do not delete the BareMetalHost that we are about to remove
		hsc.Spec.SyncSetCommonSpec.ResourceApplyMode = hivev1.UpsertResourceApplyMode
		// Remove the BareMetalHost from the list of resources to sync
		hsc.Spec.SyncSetCommonSpec.Resources = []runtime.RawExtension{}
		// Specify the BareMetalHost as a patch
		hsc.Spec.SyncSetCommonSpec.Patches = []hivev1.SyncObjectPatch{
			{
				APIVersion: metal3v1alpha1.SchemeGroupVersion.String(),
				Kind:       BareMetalHostKind,
				Name:       instance.Name,
				Namespace:  inventoryv1alpha1.ManagedClusterResourceNamespace,
				Patch:      string(bmhJSON),
				PatchType:  "merge",
			},
		}
	}
	return hsc
}

func newBareMetalHost(instance *inventoryv1alpha1.BareMetalAsset, assetSyncCompleted bool) ([]byte, error) {
	bmhSpec := map[string]interface{}{
		"bmc": map[string]string{
			"address":         instance.Spec.BMC.Address,
			"credentialsName": instance.Spec.BMC.CredentialsName,
		},
		"hardwareProfile": instance.Spec.HardwareProfile,
		"bootMACAddress":  instance.Spec.BootMACAddress,
	}
	if !assetSyncCompleted {
		bmhSpec["online"] = true
	}

	bmhJSON, err := json.Marshal(map[string]interface{}{
		"kind":       BareMetalHostKind,
		"apiVersion": metal3v1alpha1.SchemeGroupVersion.String(),
		"metadata": map[string]interface{}{
			"name":      instance.Name,
			"namespace": inventoryv1alpha1.ManagedClusterResourceNamespace,
			"labels": map[string]string{
				ClusterDeploymentNameLabel:      instance.Spec.ClusterDeployment.Name,
				ClusterDeploymentNamespaceLabel: instance.Spec.ClusterDeployment.Namespace,
				RoleLabel:                       string(instance.Spec.Role),
			},
		},
		"spec": bmhSpec,
	})
	if err != nil {
		return []byte{}, err
	}

	return bmhJSON, nil
}

func (r *ReconcileBareMetalAsset) checkHiveClusterSync(instance *inventoryv1alpha1.BareMetalAsset) bool {
	//get related syncSet
	syncSetNsN := types.NamespacedName{
		Name:      instance.Name,
		Namespace: instance.Spec.ClusterDeployment.Namespace,
	}
	foundSyncSet := &hivev1.SyncSet{}
	err := r.client.Get(context.TODO(), syncSetNsN, foundSyncSet)
	if err != nil {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
			Status: metav1.ConditionFalse,
			Reason: "SyncStatusNotFound",
			Message: fmt.Sprintf("Problem getting Hive SyncSet for Name %s in Namespace %s, %v",
				syncSetNsN.Name, syncSetNsN.Namespace, err),
		})
		return false
	}

	//get related clusterSync
	clusterSyncNsN := types.NamespacedName{
		Name:      instance.Spec.ClusterDeployment.Name,
		Namespace: instance.Spec.ClusterDeployment.Namespace,
	}

	foundClusterSync := &hiveinternalv1alpha1.ClusterSync{}
	if r.client.Get(context.TODO(), clusterSyncNsN, foundClusterSync) != nil {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:   inventoryv1alpha1.ConditionAssetSyncCompleted,
			Status: metav1.ConditionFalse,
			Reason: "SyncStatusNotFound",
			Message: fmt.Sprintf("Problem getting Hive ClusterSync for ClusterDeployment.Name %s in Namespace %s, %v",
				clusterSyncNsN.Name, clusterSyncNsN.Namespace, err),
		})
		return false
	}

	//find locate the correct syncstatus
	foundSyncStatuses := []hiveinternalv1alpha1.SyncStatus{}
	for _, syncStatus := range foundClusterSync.Status.SyncSets {
		if syncStatus.Name == instance.Name {
			foundSyncStatuses = append(foundSyncStatuses, syncStatus)
		}
	}

	if len(foundSyncStatuses) != 1 {
		err = fmt.Errorf("unable to find SyncStatus  with Name %v in ClusterSyncs %v", instance.Name, clusterSyncNsN.Name)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionAssetSyncCompleted,
			Status:  metav1.ConditionFalse,
			Reason:  "SyncStatusNotFound",
			Message: err.Error(),
		})
		return false
	}

	foundSyncStatus := foundSyncStatuses[0]
	if foundSyncStatus.ObservedGeneration != foundSyncSet.Generation {
		klog.Errorf("SyncStatus.ObserveGeneration does not match SyncSet.Generation, %v", err)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionAssetSyncStarted,
			Status:  metav1.ConditionFalse,
			Reason:  "SyncSetNotApplied",
			Message: "SyncSet not yet been applied",
		})
		return false
	}

	return r.checkHiveSyncStatus(instance, foundSyncSet, foundSyncStatus)
}

func (r *ReconcileBareMetalAsset) checkHiveSyncStatus(
	instance *inventoryv1alpha1.BareMetalAsset,
	syncSet *hivev1.SyncSet,
	syncSetStatus hiveinternalv1alpha1.SyncStatus,
) bool {
	resourceCount := len(syncSet.Spec.Resources)
	patchCount := len(syncSet.Spec.Patches)

	if resourceCount == 1 {
		if syncSetStatus.Result == hiveinternalv1alpha1.SuccessSyncSetResult {
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status:  metav1.ConditionTrue,
				Reason:  "SyncSetAppliedSuccessful",
				Message: "Successfully applied SyncSet",
			})
			return true
		}

		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionAssetSyncCompleted,
			Status:  metav1.ConditionFalse,
			Reason:  "SyncSetAppliedFailed",
			Message: fmt.Sprintf("Failed to apply SyncSet with err %s", syncSetStatus.FailureMessage),
		})
		return false
	}

	if patchCount == 1 {
		if syncSetStatus.Result == hiveinternalv1alpha1.SuccessSyncSetResult {
			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:    inventoryv1alpha1.ConditionAssetSyncCompleted,
				Status:  metav1.ConditionTrue,
				Reason:  "SyncSetAppliedSuccessful",
				Message: "Successfully applied SyncSet",
			})
			return true
		}

		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    inventoryv1alpha1.ConditionAssetSyncCompleted,
			Status:  metav1.ConditionFalse,
			Reason:  "SyncSetAppliedFailed",
			Message: fmt.Sprintf("Failed to apply SyncSet with err %s", syncSetStatus.FailureMessage),
		})

		if strings.Contains(syncSetStatus.FailureMessage, "not found") {
			if r.client.Delete(context.TODO(), syncSet) != nil {
				klog.Errorf("Failed to delete syncSet %v", instance.Name)
			}
		}
		return false
	}

	err := fmt.Errorf(
		"unexpected number of resources found on SyncSet. Expected (1) Found (%v)",
		resourceCount,
	)

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    inventoryv1alpha1.ConditionAssetSyncCompleted,
		Status:  metav1.ConditionFalse,
		Reason:  "UnexpectedResourceCount",
		Message: err.Error(),
	})

	return false
}

func (r *ReconcileBareMetalAsset) deleteSyncSet(instance *inventoryv1alpha1.BareMetalAsset) (reconcile.Result, error) {
	if instance.Spec.ClusterDeployment.Namespace == "" && instance.Spec.ClusterDeployment.Name == "" {
		instance.ObjectMeta.Finalizers = remove(instance.ObjectMeta.Finalizers, BareMetalAssetFinalizer)
		return reconcile.Result{}, r.client.Update(context.TODO(), instance)
	}

	syncSet := r.newHiveSyncSet(instance, false)
	foundSyncSet := &hivev1.SyncSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: syncSet.Name, Namespace: syncSet.Namespace}, foundSyncSet)
	if err != nil {
		if errors.IsNotFound(err) {
			instance.ObjectMeta.Finalizers = remove(instance.ObjectMeta.Finalizers, BareMetalAssetFinalizer)
			return reconcile.Result{}, r.client.Update(context.TODO(), instance)
		}
		klog.Errorf("Failed to get Hive SyncSet (%s/%s) in cleanup, %v", syncSet.Namespace, syncSet.Name, err)
		return reconcile.Result{}, err
	}

	// Only update the SyncSet if the BareMetalHost is not defined in the
	// Resources section
	if len(foundSyncSet.Spec.SyncSetCommonSpec.Resources) == 0 {
		foundSyncSet.Spec = syncSet.Spec
		return reconcile.Result{}, r.client.Update(context.TODO(), foundSyncSet)
	}

	// Don't delete the SyncSet until the ClusterSync is applied
	if r.checkHiveClusterSync(instance) {
		return reconcile.Result{}, r.client.Delete(context.TODO(), syncSet)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileBareMetalAsset) cleanupOldHiveSyncSet(instance *inventoryv1alpha1.BareMetalAsset) error {
	// If clusterDeployment.Namespace is updated to a new namespace or removed from the spec, we need to
	// ensure that existing syncset, if any, is deleted from the old namespace.
	// We can get the old syncset from relatedobjects if it exists.
	hscRef := corev1.ObjectReference{}
	for _, ro := range instance.Status.RelatedObjects {
		if ro.Name == instance.Name &&
			ro.Kind == "SyncSet" &&
			ro.APIVersion == hivev1.SchemeGroupVersion.String() &&
			ro.Namespace != instance.Spec.ClusterDeployment.Namespace {
			hscRef = ro
			break
		}
	}
	if hscRef == (corev1.ObjectReference{}) {
		// Nothing to do if no such syncset was found
		return nil
	}

	// Delete syncset in old namespace
	klog.Infof("Cleaning up Hive SyncSet in old namespace (%s/%s)", hscRef.Name, hscRef.Namespace)
	err := r.client.Delete(context.TODO(), &hivev1.SyncSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: hscRef.Namespace,
			Name:      hscRef.Name,
		},
	})

	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Failed to delete Hive SyncSet (%s/%s), %v", hscRef.Name, hscRef.Namespace, err)
			return err
		}
	}

	// Remove SyncSet from related objects
	err = objectreferencesv1.RemoveObjectReference(&instance.Status.RelatedObjects, hscRef)
	if err != nil {
		klog.Errorf("Failed to remove reference from status.RelatedObjects, %v", err)
		return err
	}

	return nil
}

// Checks whether a string is contained within a slice
func contains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Removes a given string from a slice and returns the new slice
func remove(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
