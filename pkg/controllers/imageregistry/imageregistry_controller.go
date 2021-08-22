package imageregistry

import (
	"context"
	"fmt"
	v1 "github.com/open-cluster-management/api/cluster/v1"
	clusterapiv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

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
	placementLabel = "cluster.open-cluster-management.io/placement"

	// ClusterImageRegistryLabel value is namespace.managedClusterImageRegistry
	ClusterImageRegistryLabel = "open-cluster-management.io/image-registry"
)

type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
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
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// create a new controller
	c, err := controller.New("imageregistry-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// watch for changes of ManagedClusterImageRegistry
	err = c.Watch(&source.Kind{Type: &v1alpha1.ManagedClusterImageRegistry{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// watch for the changes of PlacementDecision.
	// queue all ManagedClusterImageRegistries if a Placement is referred to multi ManagedClusterImageRegistry.
	err = c.Watch(&source.Kind{Type: &clusterapiv1alpha1.PlacementDecision{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				placementDecision, ok := a.(*clusterapiv1alpha1.PlacementDecision)
				if !ok {
					// not a placementDecision, returning empty
					klog.Error("imageRegistry handler received non-placementDecision object")
					return []reconcile.Request{}
				}

				imageRegistryList := &v1alpha1.ManagedClusterImageRegistryList{}
				err := mgr.GetClient().List(context.TODO(), imageRegistryList, client.InNamespace(placementDecision.Namespace))
				if err != nil {
					klog.Errorf("Could not list imageRegistry %v", err)
					return []reconcile.Request{}
				}
				labels := placementDecision.GetLabels()
				placementName := labels[placementLabel]
				if placementName == "" {
					klog.Errorf("Could not get placement label in placementDecision %v", placementDecision.Name)
					return []reconcile.Request{}
				}
				var requests []reconcile.Request
				for _, imageRegistry := range imageRegistryList.Items {
					switch imageRegistry.Spec.PlacementRef.Group {
					case clusterapiv1alpha1.GroupName:
						if imageRegistry.Spec.PlacementRef.Name == placementName {
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
	klog.Info("###### Image Registry Reconcile......")
	imageRegistry := &v1alpha1.ManagedClusterImageRegistry{}
	err := r.client.Get(ctx, req.NamespacedName, imageRegistry)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, r.cleanImageRegistryLabel(ctx, req.Namespace, req.Name)
	case err != nil:
		return ctrl.Result{}, err
	}

	var conditions []metav1.Condition
	resource := imageRegistry.Spec.PlacementRef.Resource
	strings.ToLower(resource)
	if resource != "placements" && resource != "placement" {
		conditions = append(conditions, metav1.Condition{
			Type:    v1alpha1.ConditionPlacementAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  v1alpha1.ConditionReasonPlacementResourceNotFound,
			Message: fmt.Sprintf("the resource %v in PlacementRef is unavailable", imageRegistry.Spec.PlacementRef.Resource),
		})
		return ctrl.Result{}, r.updateImageRegistryCondition(ctx, imageRegistry.Namespace, imageRegistry.Name, conditions)
	}

	switch imageRegistry.Spec.PlacementRef.Group {
	case clusterapiv1alpha1.GroupName:
		selectedClusters, err := r.getSelectedClusters(ctx, imageRegistry.Namespace, imageRegistry.Spec.PlacementRef.Name)
		if err != nil {
			conditions = append(conditions, metav1.Condition{
				Type:    v1alpha1.ConditionClustersSelected,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ConditionReasonClusterSelectedFailure,
				Message: err.Error(),
			})
			return ctrl.Result{}, r.updateImageRegistryCondition(ctx, imageRegistry.Namespace, imageRegistry.Name, conditions)
		}

		conditions = append(conditions, metav1.Condition{
			Type:    v1alpha1.ConditionClustersSelected,
			Status:  metav1.ConditionTrue,
			Reason:  v1alpha1.ConditionReasonClusterSelected,
			Message: fmt.Sprintf("the clusters are selected by the placement %v", imageRegistry.Spec.PlacementRef.Name),
		})

		err = r.updateImageRegistryLabel(ctx, selectedClusters, imageRegistry.Namespace, imageRegistry.Name)
		if err != nil {
			klog.Errorf("failed to update image registry %v", req.Name)
			conditions = append(conditions, metav1.Condition{
				Type:    v1alpha1.ConditionClustersUpdated,
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ConditionReasonClustersUpdatedFailure,
				Message: err.Error(),
			})
		} else {
			conditions = append(conditions, metav1.Condition{
				Type:    v1alpha1.ConditionClustersUpdated,
				Status:  metav1.ConditionTrue,
				Reason:  v1alpha1.ConditionReasonClustersUpdated,
				Message: fmt.Sprintf("the selected clusters are updated with the imageRegistry"),
			})
		}

	default:
		conditions = append(conditions, metav1.Condition{
			Type:    v1alpha1.ConditionPlacementAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  v1alpha1.ConditionReasonPlacementGroupNotFound,
			Message: fmt.Sprintf("the group %v in PlacementRef is unavailable", imageRegistry.Spec.PlacementRef.Group),
		})
	}

	return ctrl.Result{}, r.updateImageRegistryCondition(ctx, imageRegistry.Namespace, imageRegistry.Name, conditions)
}

// cleanImageRegistryLabel remove the label 'open-cluster-management.io/image-registry: namespace.imageRegistry' of clusters
func (r *Reconciler) cleanImageRegistryLabel(ctx context.Context, namespace, imageRegistry string) error {
	imageRegistryLabelValue := generateImageRegistryLabelValue(namespace, imageRegistry)
	clusterList := &v1.ManagedClusterList{}
	err := r.client.List(ctx, clusterList, client.MatchingLabels{ClusterImageRegistryLabel: imageRegistryLabelValue})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	for _, cluster := range clusterList.Items {
		err := r.removeClusterImageRegistryLabel(ctx, cluster.Name, imageRegistryLabelValue)
		if err != nil {
			klog.Errorf("failed to remove cluster %v image registry label", cluster.Name)
		}
	}

	return nil
}

// updateImageRegistryLabel will update the label 'open-cluster-management.io/image-registry: namespace.imageRegistry' of clusters.
// 1. remove the label if the labeled cluster is not selected.
// 2. update the label of selected clusters.
// 2.1 add the label if the selected cluster has not the ClusterImageRegistryLabel.
// 2.2 update the label if the imageRegistry creation time is before the old one.
func (r *Reconciler) updateImageRegistryLabel(ctx context.Context, selectedClusters []string, namespace, imageRegistry string) error {
	var errs []error
	needUpdateClustersMap := sets.NewString(selectedClusters...)

	imageRegistryLabelValue := generateImageRegistryLabelValue(namespace, imageRegistry)
	clusterList := &v1.ManagedClusterList{}
	err := r.client.List(ctx, clusterList, client.MatchingLabels{ClusterImageRegistryLabel: imageRegistryLabelValue})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	for _, cluster := range clusterList.Items {
		if needUpdateClustersMap.Has(cluster.Name) {
			// remove cluster from need update clusters
			needUpdateClustersMap.Delete(cluster.Name)
		} else {
			// remove the label of cluster since the cluster is not selected
			err := r.removeClusterImageRegistryLabel(ctx, cluster.Name, imageRegistryLabelValue)
			if err != nil && !errors.IsNotFound(err) {
				errs = append(errs, err)
			}
		}
	}

	for cluster := range needUpdateClustersMap {
		err := r.updateClusterImageRegistryLabel(ctx, cluster, imageRegistryLabelValue)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return utils.NewMultiLineAggregate(errs)
	}
	return nil
}

func (r *Reconciler) removeClusterImageRegistryLabel(ctx context.Context, clusterName, imageRegistryLabelValue string) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster := &v1.ManagedCluster{}
		err := r.client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			return client.IgnoreNotFound(err)
		}
		labels := cluster.GetLabels()
		if labels == nil {
			return nil
		}
		if labels[ClusterImageRegistryLabel] == "" {
			return nil
		}

		if labels[ClusterImageRegistryLabel] == imageRegistryLabelValue {
			delete(labels, ClusterImageRegistryLabel)
			cluster.SetLabels(labels)
			return r.client.Update(ctx, cluster, &client.UpdateOptions{})
		}
		return nil
	})
	return err
}

// updateClusterImageRegistryLabel will update the label 'open-cluster-management.io/image-registry: namespace.imageRegistry' of clusters.
// 1. add the label if the selected cluster has not the ClusterImageRegistryLabel.
// 2. update the label if the imageRegistry creation time is before the old one.
func (r *Reconciler) updateClusterImageRegistryLabel(ctx context.Context, clusterName, imageRegistryLabelValue string) error {
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster := &v1.ManagedCluster{}
		err := r.client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		labels := cluster.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}

		oldImageRegistryLabelValue := labels[ClusterImageRegistryLabel]

		if oldImageRegistryLabelValue == "" {
			labels[ClusterImageRegistryLabel] = imageRegistryLabelValue
			cluster.SetLabels(labels)
			return r.client.Update(ctx, cluster, &client.UpdateOptions{})
		}

		if oldImageRegistryLabelValue != imageRegistryLabelValue {
			oldImageRegistry, err := r.getImageRegistry(ctx, oldImageRegistryLabelValue)
			if err != nil {
				if errors.IsNotFound(err) {
					labels[ClusterImageRegistryLabel] = imageRegistryLabelValue
					cluster.SetLabels(labels)
					return r.client.Update(ctx, cluster, &client.UpdateOptions{})
				}
				klog.Errorf("failed to get old imageRegistry %v", err)
				return err
			}

			newImageRegistry, err := r.getImageRegistry(ctx, imageRegistryLabelValue)
			if err != nil {
				klog.Errorf("failed to get current imageRegistry %v", err)
				return nil
			}

			if oldImageRegistry.CreationTimestamp.After(newImageRegistry.CreationTimestamp.Time) {
				labels[ClusterImageRegistryLabel] = imageRegistryLabelValue
				cluster.SetLabels(labels)
				return r.client.Update(ctx, cluster, &client.UpdateOptions{})
			}
		}
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
	placementDecisionList := &clusterapiv1alpha1.PlacementDecisionList{}

	err := r.client.List(ctx, placementDecisionList,
		client.InNamespace(namespace), client.MatchingLabels{placementLabel: placementName})
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
			return r.client.Status().Update(ctx, imageRegistry, &client.UpdateOptions{})
		}
		return nil
	})
	return err
}

func generateImageRegistryLabelValue(namespace, imageRegistry string) string {
	return namespace + "." + imageRegistry
}
