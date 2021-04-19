package clustersetmapper

import (
	"context"
	"time"

	clusterv1 "github.com/open-cluster-management/api/cluster/v1"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	hivev1 "github.com/openshift/hive/pkg/apis/hive/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

const (
	clustersetRoleFinalizerName = "cluster.open-cluster-management.io/managedclusterset-clusterrole"
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
	if err := add(mgr, newReconciler(mgr, kubeClient, clusterSetClusterMapper, clusterSetNamespaceMapper)); err != nil {
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
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterset-clusterrole-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &clusterv1alpha1.ManagedClusterSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	reconcile := reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}

	clusterset := &clusterv1alpha1.ManagedClusterSet{}

	err := r.client.Get(ctx, types.NamespacedName{Name: req.Name}, clusterset)
	if err != nil {
		return reconcile, client.IgnoreNotFound(err)
	}
	// Check DeletionTimestamp to determine if object is under deletion
	if !clusterset.GetDeletionTimestamp().IsZero() {
		// The object is being deleted
		if utils.ContainsString(clusterset.GetFinalizers(), clustersetRoleFinalizerName) {
			klog.Infof("deleting ManagedClusterset %v", clusterset.Name)
			err := r.cleanClusterSetResource(clusterset.Name)
			if err != nil {
				klog.Warningf("will reconcile since failed to delete ManagedClusterSet %v : %v", clusterset.Name, err)
				return reconcile, err
			}
			klog.Infof("removing ManagedClusterset Finalizer in ManagedCluster %v", clusterset.Name)
			clusterset.ObjectMeta.Finalizers = utils.RemoveString(clusterset.ObjectMeta.Finalizers, clustersetRoleFinalizerName)
			if err := r.client.Update(context.TODO(), clusterset); err != nil {
				klog.Warningf("will reconcile since failed to remove Finalizer from ManagedClusterset %v, %v", clusterset.Name, err)
				return reconcile, err
			}
		}
		return reconcile, nil
	}

	if !utils.ContainsString(clusterset.GetFinalizers(), clustersetRoleFinalizerName) {
		klog.Infof("adding ManagedClusterSet Finalizer to ManagedCluster %v", clusterset.Name)
		clusterset.ObjectMeta.Finalizers = append(clusterset.ObjectMeta.Finalizers, clustersetRoleFinalizerName)
		if err := r.client.Update(context.TODO(), clusterset); err != nil {
			klog.Warningf("will reconcile since failed to add finalizer to ManagedClusterset %v, %v", clusterset.Name, err)
			return reconcile, err
		}
	}

	//add clusterset clusterrole
	adminRules := buildAdminRoleRules(clusterset.Name)
	err = utils.ApplyClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "admin"), adminRules)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", clusterset.Name, err)
		return reconcile, err
	}
	viewRules := buildViewRoleRules(clusterset.Name)
	err = utils.ApplyClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clusterset.Name, "view"), viewRules)
	if err != nil {
		klog.Warningf("will reconcile since failed to create/update clusterrole %v, %v", clusterset.Name, err)
		return reconcile, err
	}

	r.syncClustersetMapper(clusterset.Name)

	return reconcile, nil
}

//cleanClusterSetResource clean the clusterrole
//and delete clusterset related resource in clusterSetClusterMapper and clusterSetNamespaceMapper
func (r *Reconciler) cleanClusterSetResource(clustersetName string) error {
	//Delete clusterset clusterrole
	err := utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clustersetName, "admin"))
	if err != nil {
		klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", clustersetName, err)
		return err
	}
	err = utils.DeleteClusterRole(r.kubeClient, utils.GenerateClustersetClusterroleName(clustersetName, "view"))
	if err != nil {
		klog.Warningf("will reconcile since failed to delete clusterrole. clusterset: %v, err: %v", clustersetName, err)
		return err
	}

	//Delete clusterset which in clustersetMapper
	r.clusterSetClusterMapper.DeleteClusterSet(clustersetName)
	r.clusterSetNamespaceMapper.DeleteClusterSet(clustersetName)

	return nil
}

func (r *Reconciler) syncClustersetMapper(clustersetName string) error {
	//List Clusterset related resource by utils.ClusterSetLabel
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{
		utils.ClusterSetLabel: clustersetName,
	}}
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return err
	}
	clusterSetClusterMapper, err := r.generateClustersetClusterMapper(selector)
	if err != nil {
		return err
	}
	r.clusterSetClusterMapper.CopyClusterSetMapper(clusterSetClusterMapper)

	clusterSetNamespaceMapper, err := r.generateClustersetNamespaceMapper(selector)
	if err != nil {
		return err
	}
	r.clusterSetNamespaceMapper.CopyClusterSetMapper(clusterSetNamespaceMapper)

	return nil
}

// generateClustersetClusterMapper generate clustersetclustermapper,
// which is used to create clusterrolebinding for each managedcluster.
func (r *Reconciler) generateClustersetClusterMapper(selector labels.Selector) (*helpers.ClusterSetMapper, error) {
	managedClustersList := &clusterv1.ManagedClusterList{}

	err := r.client.List(context.TODO(), managedClustersList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list managedClustersList %v", err)
		return nil, err
	}

	newClusterSetClusterMapper := helpers.NewClusterSetMapper()
	for _, managedCluster := range managedClustersList.Items {
		newClusterSetClusterMapper.UpdateObjectInClusterSet(managedCluster.Name, managedCluster.Labels[utils.ClusterSetLabel])
	}
	return newClusterSetClusterMapper, nil
}

// generateClustersetNamespaceMapper generate Clusterset to namespace mapper
// The namespace include clusterclaim/clusterpool/clusterdeployment namespace.
// For each namespace, we create an admin role binding and an view rolebinding in another controller.
func (r *Reconciler) generateClustersetNamespaceMapper(selector labels.Selector) (*helpers.ClusterSetMapper, error) {
	newClusterSetNamespaceMapper := helpers.NewClusterSetMapper()

	//Add clusterclaim namespace to newClusterSetNamespaceMapper
	clusterClaimList := &hivev1.ClusterClaimList{}
	err := r.client.List(context.TODO(), clusterClaimList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list clusterclaim %v", err)
		return nil, err
	}
	for _, clusterClaim := range clusterClaimList.Items {
		newClusterSetNamespaceMapper.UpdateObjectInClusterSet(clusterClaim.Namespace, clusterClaim.Labels[utils.ClusterSetLabel])
	}

	//Add clusterdeployment namespace to newClusterSetNamespaceMapper
	clusterDeploymentList := &hivev1.ClusterDeploymentList{}
	err = r.client.List(context.TODO(), clusterDeploymentList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list clusterclaim %v", err)
		return nil, err
	}
	for _, clusterDeployment := range clusterDeploymentList.Items {
		newClusterSetNamespaceMapper.UpdateObjectInClusterSet(clusterDeployment.Namespace, clusterDeployment.Labels[utils.ClusterSetLabel])
	}

	//Add clusterpool namespace to newClusterSetNamespaceMapper
	clusterPoolList := &hivev1.ClusterPoolList{}
	err = r.client.List(context.TODO(), clusterPoolList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		klog.Errorf("failed to list clusterclaim %v", err)
		return nil, err
	}

	for _, clusterPool := range clusterPoolList.Items {
		newClusterSetNamespaceMapper.UpdateObjectInClusterSet(clusterPool.Namespace, clusterPool.Labels[utils.ClusterSetLabel])
	}

	return newClusterSetNamespaceMapper, nil
}
