package clustersetmapper

import (
	"context"

	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//This controller apply the clusterset clusterrole and sync clusterSetClusterMapper and clusterSetNamespaceMapper
type Reconciler struct {
	client                    client.Client
	scheme                    *runtime.Scheme
	kubeClient                kubernetes.Interface
	clusterSetClusterMapper   *helpers.ClusterSetMapper
	clusterSetNamespaceMapper *helpers.ClusterSetMapper
}

func SetupWithManager(mgr manager.Manager, kubeClient kubernetes.Interface, clusterSetClusterMapper *helpers.ClusterSetMapper, clusterSetNamespaceMapper *helpers.ClusterSetMapper) error {
	if err := add(mgr, clusterSetClusterMapper, clusterSetNamespaceMapper, newReconciler(mgr, kubeClient, clusterSetClusterMapper, clusterSetNamespaceMapper)); err != nil {
		klog.Errorf("Failed to create clusterrole controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, kubeClient kubernetes.Interface, clusterSetClusterMapper *helpers.ClusterSetMapper, clusterSetNamespaceMapper *helpers.ClusterSetMapper) reconcile.Reconciler {
	return &Reconciler{
		client:                    mgr.GetClient(),
		scheme:                    mgr.GetScheme(),
		kubeClient:                kubeClient,
		clusterSetClusterMapper:   clusterSetClusterMapper,
		clusterSetNamespaceMapper: clusterSetNamespaceMapper,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, clusterSetClusterMapper *helpers.ClusterSetMapper, clusterSetNamespaceMapper *helpers.ClusterSetMapper, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterset-clusterrole-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1beta1.ManagedClusterSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterPool
	err = c.Watch(
		&source.Kind{Type: &hivev1.ClusterPool{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				clusterPool, ok := a.(*hivev1.ClusterPool)
				if !ok {
					klog.Error("clusterPool handler received non-clusterPool object")
					return []reconcile.Request{}
				}
				requests := getRequiredClusterSet(clusterPool.Labels, clusterSetNamespaceMapper, clusterPool.Namespace)
				return requests
			},
			),
		),
	)
	if err != nil {
		return err
	}

	// Watch for changes to ClusterClaim
	err = c.Watch(
		&source.Kind{Type: &hivev1.ClusterClaim{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				clusterClaim, ok := a.(*hivev1.ClusterClaim)
				if !ok {
					klog.Error("clusterClaim handler received non-clusterClaim object")
					return []reconcile.Request{}
				}
				requests := getRequiredClusterSet(clusterClaim.Labels, clusterSetNamespaceMapper, clusterClaim.Namespace)
				return requests
			}),
		))
	if err != nil {
		return err
	}

	// Watch for changes to ClusterDeployment
	err = c.Watch(
		&source.Kind{Type: &hivev1.ClusterDeployment{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				clusterDeployment, ok := a.(*hivev1.ClusterDeployment)
				if !ok {
					klog.Error("clusterDeployment handler received non-clusterDeployment object")
					return []reconcile.Request{}
				}
				requests := getRequiredClusterSet(clusterDeployment.Labels, clusterSetNamespaceMapper, clusterDeployment.Namespace)
				return requests
			}),
		))
	if err != nil {
		return err
	}

	// Watch for changes to ManagedCluster
	err = c.Watch(
		&source.Kind{Type: &clusterv1.ManagedCluster{}},
		handler.EnqueueRequestsFromMapFunc(
			handler.MapFunc(func(a client.Object) []reconcile.Request {
				managedCluster, ok := a.(*clusterv1.ManagedCluster)
				if !ok {
					klog.Error("managedCluster handler received non-managedCluster object")
					return []reconcile.Request{}
				}
				requests := getRequiredClusterSet(managedCluster.Labels, clusterSetClusterMapper, managedCluster.Name)
				return requests
			}),
		))
	if err != nil {
		return err
	}
	return nil
}

// if the labels include clusterset, add the clusterset to request
// find the resource from clusterSetMapper, then add resource related clusterset to request
func getRequiredClusterSet(labels map[string]string, clusterSetMapper *helpers.ClusterSetMapper, namespace string) []reconcile.Request {
	var currentSet string
	var requests []reconcile.Request
	if labels != nil && len(labels[clusterv1beta1.ClusterSetLabel]) != 0 {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: labels[clusterv1beta1.ClusterSetLabel],
			},
		}
		requests = append(requests, request)
		currentSet = labels[clusterv1beta1.ClusterSetLabel]
	}

	managedClusterset := clusterSetMapper.GetObjectClusterset(namespace)
	if managedClusterset != "" {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: managedClusterset,
			},
		}
		if currentSet != managedClusterset {
			requests = append(requests, request)
		}
	}
	return requests
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	clusterset := &clusterv1beta1.ManagedClusterSet{}

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, clusterset)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check DeletionTimestamp to determine if object is under deletion
	if !clusterset.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(clusterset.GetFinalizers(), clustersetutils.ClustersetRoleFinalizerName) {
			klog.Infof("deleting ManagedClusterset %v", clusterset.Name)
			err := r.cleanClusterSetResource(clusterset)
			if err != nil {
				klog.Warningf("will reconcile since failed to delete ManagedClusterSet %v : %v", clusterset.Name, err)
				return ctrl.Result{}, err
			}
			klog.Infof("removing Clusterrole Finalizer in ManagedClusterset %v", clusterset.Name)
			clusterset.ObjectMeta.Finalizers = utils.RemoveString(clusterset.ObjectMeta.Finalizers, clustersetutils.ClustersetRoleFinalizerName)
			if err := r.client.Update(context.TODO(), clusterset); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedClusterset %v, %v", clusterset.Name, err)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !utils.ContainsString(clusterset.GetFinalizers(), clustersetutils.ClustersetRoleFinalizerName) {
		klog.Infof("adding Clusterrole Finalizer to ManagedClusterset %v", clusterset.Name)
		clusterset.ObjectMeta.Finalizers = append(clusterset.ObjectMeta.Finalizers, clustersetutils.ClustersetRoleFinalizerName)
		if err := r.client.Update(context.TODO(), clusterset); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ManagedClusterset %v, %v", clusterset.Name, err)
			return ctrl.Result{}, err
		}
	}

	err = r.applyClusterSetClusterRoles(clusterset)
	if err != nil {
		klog.Warningf("will reconcile since failed to apply clusterset clusterrole. clusterset: %v, err: %v", clusterset.Name, err)
		return ctrl.Result{}, err
	}

	//The following code is used to sync the r.clusterSetClusterMapper and r.clusterSetNamespaceMapper
	//r.clusterSetClusterMapper(map[string]sets.String) stores the map of <ClusterSet Name> to <Clusters Name>, each item in the map means the clusterset include these clusters
	//r.clusterSetNamespaceMapper (map[string]sets.String) stores the map of <ClusterSet Name> to <namespaces>, the namespaces are the namespace of clusterpools/clusterclaims/clusterdeployments which are in this clusterset.
	//These two Mappers are used to propagate the clusterset admin/bind/view permission to managedclusters/managedclusters namespaces/clusterpools namespace/clusterclaims namespace/clusterdeployments namespaces which are in the clusterset.
	//Currentlly, global clusterset is mainly used for scheduling, So there is no need to propoagate the clusterpools/clusterclaims/clusterdeployments permissions.

	//TODO: Will propagate global clusterset view permission to all managedclusters and managedclusters namespaces later.
	if clusterset.Spec.ClusterSelector.SelectorType != clusterv1beta1.LegacyClusterSetLabel {
		return ctrl.Result{}, nil
	}

	err = r.syncClustersetMapper(clusterset.Name)
	if err != nil {
		klog.Warningf("will reconcile since failed to sync clustersetmapper %v, %v", clusterset.Name, err)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

//applyClusterSetClusterRoles apply the clusterset clusterrole(admin/bind/view)
func (r *Reconciler) applyClusterSetClusterRoles(clusterset *clusterv1beta1.ManagedClusterSet) error {
	errs := []error{}
	if clusterset.Spec.ClusterSelector.SelectorType == clusterv1beta1.LegacyClusterSetLabel {
		adminRole := clustersetutils.BuildAdminRole(clusterset.Name)
		err := utils.ApplyClusterRole(r.kubeClient, adminRole)
		if err != nil {
			klog.Warningf("will reconcile since failed to create/update admin clusterrole %v, %v", clusterset.Name, err)
			errs = append(errs, err)
		}
	}

	bindRole := clustersetutils.BuildBindRole(clusterset.Name)
	err := utils.ApplyClusterRole(r.kubeClient, bindRole)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update bind clusterrole %v, %v", clusterset.Name, err)
		errs = append(errs, err)
	}

	viewRole := clustersetutils.BuildViewRole(clusterset.Name)
	err = utils.ApplyClusterRole(r.kubeClient, viewRole)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update view clusterrole %v, %v", clusterset.Name, err)
		errs = append(errs, err)
	}
	return utils.NewMultiLineAggregate(errs)
}

//cleanClusterSetResource clean the clusterrole
//and delete clusterset related resource in clusterSetClusterMapper and clusterSetNamespaceMapper
func (r *Reconciler) cleanClusterSetResource(clusterset *clusterv1beta1.ManagedClusterSet) error {

	err := utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "bind"))
	if err != nil {
		klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", clusterset.Name, err)
		return err
	}

	err = utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "view"))
	if err != nil {
		klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", clusterset.Name, err)
		return err
	}

	//Only LegacyClusterSet has admin clusterrole, so only LegacyClusterSet need to delete it here.
	if clusterset.Spec.ClusterSelector.SelectorType == clusterv1beta1.LegacyClusterSetLabel {
		err := utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "admin"))
		if err != nil {
			klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", clusterset.Name, err)
			return err
		}
		//Delete clusterset which in clustersetMapper
		r.clusterSetClusterMapper.DeleteClusterSet(clusterset.Name)
		r.clusterSetNamespaceMapper.DeleteClusterSet(clusterset.Name)

	}

	return nil
}

func (r *Reconciler) syncClustersetMapper(clustersetName string) error {
	//List Clusterset related resource by ClusterSetLabel
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
		clusterv1beta1.ClusterSetLabel: clustersetName,
	}}
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return err
	}
	clusters, err := r.generateClustersetCluster(selector)
	if err != nil {
		return err
	}
	r.clusterSetClusterMapper.UpdateClusterSetByObjects(clustersetName, clusters)

	namespaces, err := r.generateClustersetNamespace(selector)
	if err != nil {
		return err
	}
	r.clusterSetNamespaceMapper.UpdateClusterSetByObjects(clustersetName, namespaces)

	return nil
}

// generateClustersetCluster generate managedclusters sets which owned by one clusterset(selector)
func (r *Reconciler) generateClustersetCluster(selector labels.Selector) (sets.String, error) {
	managedClustersList := &clusterv1.ManagedClusterList{}
	clusters := sets.NewString()
	err := r.client.List(context.TODO(), managedClustersList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list managedClustersList %v", err)
		return nil, err
	}

	for _, managedCluster := range managedClustersList.Items {
		clusters.Insert(managedCluster.Name)
	}
	return clusters, nil
}

// generateClustersetNamespace generate namespace which owned by one clusterset(selector).
// The namespace include clusterclaim/clusterpool/clusterdeployment namespace.
// For each namespace, we create an admin rolebinding and an view rolebinding in another controller.
func (r *Reconciler) generateClustersetNamespace(selector labels.Selector) (sets.String, error) {
	namespaces := sets.NewString()

	//Add clusterclaim namespace to newClusterSetNamespaceMapper
	clusterClaimList := &hivev1.ClusterClaimList{}
	err := r.client.List(context.TODO(), clusterClaimList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list clusterclaim %v", err)
		return nil, err
	}
	for _, clusterClaim := range clusterClaimList.Items {
		namespaces.Insert(clusterClaim.Namespace)
	}

	//Add clusterdeployment namespace to newClusterSetNamespaceMapper
	clusterDeploymentList := &hivev1.ClusterDeploymentList{}
	err = r.client.List(context.TODO(), clusterDeploymentList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list clusterclaim %v", err)
		return nil, err
	}
	for _, clusterDeployment := range clusterDeploymentList.Items {
		namespaces.Insert(clusterDeployment.Namespace)
	}

	//Add clusterpool namespace to newClusterSetNamespaceMapper
	clusterPoolList := &hivev1.ClusterPoolList{}
	err = r.client.List(context.TODO(), clusterPoolList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list clusterclaim %v", err)
		return nil, err
	}

	for _, clusterPool := range clusterPoolList.Items {
		namespaces.Insert(clusterPool.Namespace)
	}

	return namespaces, nil
}
