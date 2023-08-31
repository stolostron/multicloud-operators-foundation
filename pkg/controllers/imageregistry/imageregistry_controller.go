package imageregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stolostron/cluster-lifecycle-api/imageregistry/v1alpha1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	v1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	imageRegistryFinalizerName = "imageregistry.finalizers.open-cluster-management.io"
)

type Reconciler struct {
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func SetupWithManager(mgr manager.Manager) error {
	if err := add(mgr, newReconciler(mgr)); err != nil {
		klog.Errorf("Failed to create imageregistry controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("image-registry"),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// create a new controller
	c, err := controller.New("imageregistry-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// watch for changes of ManagedClusterImageRegistry
	err = c.Watch(source.Kind(mgr.GetCache(), &v1alpha1.ManagedClusterImageRegistry{}), &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// watch for the changes of PlacementDecision.
	// queue all ManagedClusterImageRegistries if a Placement is referred to multi ManagedClusterImageRegistry.
	err = c.Watch(source.Kind(mgr.GetCache(), &clusterv1beta1.PlacementDecision{}),
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(ctx context.Context, a client.Object) []reconcile.Request {
				placementDecision, ok := a.(*clusterv1beta1.PlacementDecision)
				if !ok {
					// not a placementDecision, returning empty
					klog.Error("imageRegistry handler received non-placementDecision object")
					return []reconcile.Request{}
				}

				labels := placementDecision.GetLabels()
				if len(labels) == 0 || labels[clusterv1beta1.PlacementLabel] == "" {
					klog.V(2).Infof("Could not get placement label in placementDecision %v", placementDecision.Name)
					return []reconcile.Request{}
				}

				imageRegistryList := &v1alpha1.ManagedClusterImageRegistryList{}
				err := mgr.GetClient().List(context.TODO(), imageRegistryList, client.InNamespace(placementDecision.Namespace))
				if err != nil {
					klog.Errorf("Could not list imageRegistry %v", err)
					return []reconcile.Request{}
				}

				var requests []reconcile.Request
				for _, imageRegistry := range imageRegistryList.Items {
					switch imageRegistry.Spec.PlacementRef.Group {
					case clusterv1beta1.GroupName:
						if imageRegistry.Spec.PlacementRef.Name == labels[clusterv1beta1.PlacementLabel] {
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Name:      imageRegistry.Name,
									Namespace: imageRegistry.Namespace,
								},
							})
						}
					}
				}

				return requests
			}),
		))
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	imageRegistry := &v1alpha1.ManagedClusterImageRegistry{}
	err := r.client.Get(ctx, req.NamespacedName, imageRegistry)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !imageRegistry.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(imageRegistry.GetFinalizers(), imageRegistryFinalizerName) {
			err := r.cleanAllClustersImageRegistry(ctx, req.Namespace, req.Name)
			if err != nil {
				return reconcile.Result{}, err
			}

			imageRegistry.ObjectMeta.Finalizers = utils.RemoveString(imageRegistry.ObjectMeta.Finalizers,
				imageRegistryFinalizerName)
			r.recorder.Eventf(imageRegistry, "Normal", "ImageRegistryDelete", "imageRegistry %v is deleted.", imageRegistry.Name)
			return reconcile.Result{}, r.client.Update(context.TODO(), imageRegistry)
		}

		return reconcile.Result{}, nil
	}

	if !utils.ContainsString(imageRegistry.GetFinalizers(), imageRegistryFinalizerName) {
		imageRegistry.ObjectMeta.Finalizers = append(imageRegistry.ObjectMeta.Finalizers, imageRegistryFinalizerName)
		return reconcile.Result{}, r.client.Update(context.TODO(), imageRegistry)
	}

	var conditions []metav1.Condition
	switch imageRegistry.Spec.PlacementRef.Group {
	case clusterv1beta1.GroupName:
		selectedClusters, err := r.getSelectedClusters(ctx, imageRegistry.Namespace, imageRegistry.Spec.PlacementRef.Name)
		if err != nil {
			conditions = append(conditions, metav1.Condition{
				Type:    v1alpha1.ConditionClustersSelected,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ConditionReasonClusterSelectedFailure,
				Message: err.Error(),
			})
			r.recorder.Eventf(imageRegistry, "Warning", "ImageRegistryUpdate", "failed to get selectedClusters: %v", err)
			updateErr := r.updateImageRegistryCondition(ctx, imageRegistry.Namespace, imageRegistry.Name, conditions)
			if updateErr != nil {
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, err
		}

		conditions = append(conditions, metav1.Condition{
			Type:    v1alpha1.ConditionClustersSelected,
			Status:  metav1.ConditionTrue,
			Reason:  v1alpha1.ConditionReasonClusterSelected,
			Message: fmt.Sprintf("the clusters are selected by the placement %v", imageRegistry.Spec.PlacementRef.Name),
		})

		err = r.updateAllClustersImageRegistry(ctx, selectedClusters, imageRegistry)
		if err != nil {
			klog.Errorf("failed to update image registry %v", req.Name)
			conditions = append(conditions, metav1.Condition{
				Type:    v1alpha1.ConditionClustersUpdated,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ConditionReasonClustersUpdatedFailure,
				Message: err.Error(),
			})
			r.recorder.Eventf(imageRegistry, "Warning", "ImageRegistryUpdate", "failed to update image registry labels: %v", err)
			updateErr := r.updateImageRegistryCondition(ctx, imageRegistry.Namespace, imageRegistry.Name, conditions)
			if updateErr != nil {
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, err
		} else {
			conditions = append(conditions, metav1.Condition{
				Type:    v1alpha1.ConditionClustersUpdated,
				Status:  metav1.ConditionTrue,
				Reason:  v1alpha1.ConditionReasonClustersUpdated,
				Message: fmt.Sprintf("the selected clusters are updated with the imageRegistry"),
			})
		}
	}

	return ctrl.Result{}, r.updateImageRegistryCondition(ctx, imageRegistry.Namespace, imageRegistry.Name, conditions)
}

// cleanAllClustersImageRegistry remove the label and annotation of clusters
func (r *Reconciler) cleanAllClustersImageRegistry(ctx context.Context, namespace, imageRegistry string) error {
	imageRegistryLabelValue := generateImageRegistryLabelValue(namespace, imageRegistry)
	clusterList := &v1.ManagedClusterList{}
	err := r.client.List(ctx, clusterList, client.MatchingLabels{v1alpha1.ClusterImageRegistryLabel: imageRegistryLabelValue})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	for _, cluster := range clusterList.Items {
		err := r.removeClusterImageRegistry(ctx, cluster.Name)
		if err != nil {
			klog.Errorf("failed to remove cluster %v image registry label", cluster.Name)
		}
	}

	return nil
}

// updateAllClustersImageRegistry will update the label and annotation of clusters.
// 1. remove the label and annotation from the cluster if the labeled cluster is not selected by placement.
// 2. update the label and annotation of selected clusters.
// 2.1 add the label and annotation if the selected cluster does not have the ClusterImageRegistryLabel.
// 2.2 send warning event if the selected cluster has another ClusterImageRegistryLabel.
func (r *Reconciler) updateAllClustersImageRegistry(ctx context.Context,
	selectedClusters []string, imageRegistry *v1alpha1.ManagedClusterImageRegistry) error {
	var errs []error
	needUpdateClustersMap := sets.NewString(selectedClusters...)

	imageRegistryLabelValue := generateImageRegistryLabelValue(imageRegistry.Namespace, imageRegistry.Name)
	clusterList := &v1.ManagedClusterList{}
	err := r.client.List(ctx, clusterList, client.MatchingLabels{v1alpha1.ClusterImageRegistryLabel: imageRegistryLabelValue})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	for _, cluster := range clusterList.Items {
		if !needUpdateClustersMap.Has(cluster.Name) {
			// remove the label and annotation from the cluster since the cluster is not selected by the placement.
			err := r.removeClusterImageRegistry(ctx, cluster.Name)
			if err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
			}
		}
	}

	for cluster := range needUpdateClustersMap {
		err := r.updateClusterImageRegistry(ctx, cluster, imageRegistry)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return utils.NewMultiLineAggregate(errs)
	}
	return nil
}

func (r *Reconciler) removeClusterImageRegistry(ctx context.Context, clusterName string) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster := &v1.ManagedCluster{}
		err := r.client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			return client.IgnoreNotFound(err)
		}

		modified := false
		utils.MergeMap(&modified, &cluster.Labels, map[string]string{v1alpha1.ClusterImageRegistryLabel + "-": ""})
		utils.MergeMap(&modified, &cluster.Annotations,
			map[string]string{v1alpha1.ClusterImageRegistriesAnnotation + "-": ""})
		if !modified {
			return nil
		}
		err = r.client.Update(ctx, cluster, &client.UpdateOptions{})
		if err != nil {
			return err
		}
		r.recorder.Eventf(cluster, "Normal", "imageRegistryDelete",
			"Delete %s label for mangedCluster %v", v1alpha1.ClusterImageRegistryLabel, cluster.Name)
		return nil
	})
	return err
}

// updateClusterImageRegistry will update the label 'open-cluster-management.io/image-registry: namespace.imageRegistry'
// and annotation 'open-cluster-management.io/registries:"{pullSecret:namespace.pullSecret,
// [{mirror:xxx,source:xxx},{mirror:xxx,source:xxx}]}" ' to clusters.
// 1. add/update the label/annotation if the selected cluster has not the ClusterImageRegistryLabel.
// 2. send warning event if the selected cluster has another ClusterImageRegistryLabel.
func (r *Reconciler) updateClusterImageRegistry(ctx context.Context,
	clusterName string, imageRegistry *v1alpha1.ManagedClusterImageRegistry) error {
	imageRegistryLabelValue := generateImageRegistryLabelValue(imageRegistry.Namespace, imageRegistry.Name)
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster := &v1.ManagedCluster{}
		err := r.client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		oldImageRegistryLabelValue := cluster.Labels[v1alpha1.ClusterImageRegistryLabel]
		if oldImageRegistryLabelValue != "" && oldImageRegistryLabelValue != imageRegistryLabelValue {
			r.recorder.Eventf(cluster, "Warning", "imageRegistryUpdate",
				"Cannot update imageRegistry label %v for mangedCluster %v, since the managedCluster has imageRegistry label %v",
				imageRegistryLabelValue, cluster.Name, oldImageRegistryLabelValue)
			return nil
		}

		modified := false
		utils.MergeMap(&modified, &cluster.Annotations,
			map[string]string{
				v1alpha1.ClusterImageRegistriesAnnotation: getAnnotationRegistries(imageRegistry)})

		utils.MergeMap(&modified, &cluster.Labels,
			map[string]string{v1alpha1.ClusterImageRegistryLabel: imageRegistryLabelValue})
		if !modified {
			return nil
		}
		err = r.client.Update(ctx, cluster, &client.UpdateOptions{})
		if err != nil {
			return err
		}
		r.recorder.Eventf(cluster, "Normal", "imageRegistryAdd",
			"Add imageRegistry label %v for mangedCluster %v",
			imageRegistryLabelValue, cluster.Name)
		return nil
	})
	return err
}

// getImageRegistry get ManagedClusterImageRegistry from imageRegistryLabelValue
// the format of imageRegistryLabelValue is 'namespace.imageRegistry'
func (r *Reconciler) getImageRegistry(ctx context.Context, imageRegistryLabelValue string) (*v1alpha1.ManagedClusterImageRegistry, error) {
	segs := strings.Split(imageRegistryLabelValue, ".")
	if len(segs) != 2 {
		return nil, fmt.Errorf("wrong imageRegistry label value %v", imageRegistryLabelValue)
	}
	namespace := segs[0]
	imageRegistryName := segs[1]
	imageRegistry := &v1alpha1.ManagedClusterImageRegistry{}
	err := r.client.Get(ctx, types.NamespacedName{Name: imageRegistryName, Namespace: namespace}, imageRegistry)
	return imageRegistry, err
}

func (r *Reconciler) getSelectedClusters(ctx context.Context, namespace, placementName string) ([]string, error) {
	var clusters []string
	placementDecisionList := &clusterv1beta1.PlacementDecisionList{}

	err := r.client.List(ctx, placementDecisionList,
		client.InNamespace(namespace), client.MatchingLabels{clusterv1beta1.PlacementLabel: placementName})
	if err != nil {
		return clusters, client.IgnoreNotFound(err)
	}

	for _, placementDecision := range placementDecisionList.Items {
		for _, decision := range placementDecision.Status.Decisions {
			clusters = append(clusters, decision.ClusterName)
		}
	}
	return clusters, nil
}

func (r *Reconciler) updateImageRegistryCondition(ctx context.Context, namespace, imageRegistryName string, conditions []metav1.Condition) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		imageRegistry := &v1alpha1.ManagedClusterImageRegistry{}
		err := r.client.Get(ctx, types.NamespacedName{Name: imageRegistryName, Namespace: namespace}, imageRegistry)
		if err != nil {
			return err
		}

		oldStatus := imageRegistry.Status
		newStatus := oldStatus.DeepCopy()
		for _, condition := range conditions {
			meta.SetStatusCondition(&newStatus.Conditions, condition)
		}
		if !equality.Semantic.DeepEqual(oldStatus, newStatus) {
			imageRegistry.Status = *newStatus
			return r.client.Status().Update(ctx, imageRegistry)
		}
		return nil
	})
	return err
}

func generateImageRegistryLabelValue(namespace, imageRegistry string) string {
	return namespace + "." + imageRegistry
}

// getAnnotationRegistries generate the registries' annotation values.
// ignore Registry if Registries is not empty
// transfer Registry to Registries if Registries is empty.
// the empty Source means the Mirror registry can override all source registry.
func getAnnotationRegistries(imageRegistry *v1alpha1.ManagedClusterImageRegistry) string {
	if len(imageRegistry.Spec.Registries) == 0 && len(imageRegistry.Spec.Registry) == 0 {
		return ""
	}

	registriesData := v1alpha1.ImageRegistries{
		PullSecret: fmt.Sprintf("%s.%s", imageRegistry.Namespace, imageRegistry.Spec.PullSecret.Name),
		Registries: imageRegistry.Spec.Registries,
	}

	if len(registriesData.Registries) == 0 {
		registriesData.Registries = []v1alpha1.Registries{
			{
				Mirror: imageRegistry.Spec.Registry,
				Source: "",
			},
		}
	}

	registriesDataStr, _ := json.Marshal(registriesData)
	return string(registriesDataStr)
}
