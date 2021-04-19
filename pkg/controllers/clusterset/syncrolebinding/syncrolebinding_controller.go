package syncrolebinding

import (
	"context"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//This controller apply clusterset related clusterrolebinding based on clustersetToNamespace and clustersetAdminToSubject map
type Reconciler struct {
	client                client.Client
	scheme                *runtime.Scheme
	clusterSetCache       *cache.ClusterSetCache
	clustersetToNamespace *helpers.ClusterSetMapper
	clustersetToClusters  *helpers.ClusterSetMapper
}

func SetupWithManager(mgr manager.Manager, clusterSetCache *cache.ClusterSetCache, clustersetToNamespace *helpers.ClusterSetMapper, clustersetToClusters *helpers.ClusterSetMapper) error {
	if err := add(mgr, newReconciler(mgr, clusterSetCache, clustersetToNamespace, clustersetToClusters)); err != nil {
		klog.Errorf("Failed to create clusterset rolebinding controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clusterSetCache *cache.ClusterSetCache, clustersetToNamespace *helpers.ClusterSetMapper, clustersetToClusters *helpers.ClusterSetMapper) reconcile.Reconciler {
	return &Reconciler{
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		clusterSetCache:       clusterSetCache,
		clustersetToNamespace: clustersetToNamespace,
		clustersetToClusters:  clustersetToClusters,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterset-rolebinding-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

//This function sycn the rolebinding in namespace which in r.clustersetToNamespace and r.clustersetToClusters
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//reconcile every 10s
	reconcile := reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}

	ctx := context.Background()

	//union the clusterset to namespace and clusterset to cluster(it's same as managedcluster namespace).
	//so we can use unionclustersetToNamespace to generate role bindings.
	unionclustersetToNamespace := r.clustersetToNamespace.UnionObjectsInClusterSet(r.clustersetToClusters)
	klog.Errorf("######admin:%v", unionclustersetToNamespace.GetAllClusterSetToObjects())
	clustersetToAdminSubjects := utils.GenerateClustersetSubjects(r.clusterSetCache.AdminCache)
	clustersetToViewSubjects := utils.GenerateClustersetSubjects(r.clusterSetCache.ViewCache)

	adminErrs := r.syncRoleBinding(ctx, unionclustersetToNamespace, clustersetToAdminSubjects, "admin")
	viewErrs := r.syncRoleBinding(ctx, unionclustersetToNamespace, clustersetToViewSubjects, "view")

	errs := append(adminErrs, viewErrs...)
	return reconcile, utils.NewMultiLineAggregate(errs)
}

func (r *Reconciler) syncRoleBinding(ctx context.Context, clustersetToNamespace *helpers.ClusterSetMapper, clustersetToSubject map[string][]rbacv1.Subject, role string) []error {
	namespaceToSubject := utils.GenerateObjectSubjectMap(clustersetToNamespace, clustersetToSubject)
	//apply all disired clusterrolebinding
	errs := []error{}
	for namespace, subjects := range namespaceToSubject {
		requiredRoleBinding := generateRequiredRoleBinding(namespace, subjects, role)
		err := utils.ApplyRoleBinding(ctx, r.client, requiredRoleBinding)
		if err != nil {
			klog.Errorf("Failed to apply rolebinding: %v, error:%v", requiredRoleBinding.Name, err)
			errs = append(errs, err)
		}
	}

	//Delete rolebinding
	roleBindingList := &rbacv1.RoleBindingList{}

	//List Clusterset related clusterrolebinding
	matchExpressions := metav1.LabelSelectorRequirement{Key: utils.ClusterSetLabel, Operator: metav1.LabelSelectorOpExists}
	labelSelector := metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{matchExpressions}}
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return []error{err}
	}

	err = r.client.List(ctx, roleBindingList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return []error{err}
	}
	for _, roleBinding := range roleBindingList.Items {
		curRoleBinding := roleBinding

		//only handle current resource rolebinding
		matchRoleBindingName := utils.GenerateClustersetResourceRoleBindingName(role)

		if matchRoleBindingName != curRoleBinding.Name {
			continue
		}

		if _, ok := namespaceToSubject[roleBinding.Namespace]; !ok {
			err = r.client.Delete(ctx, &curRoleBinding)
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}
	}
	return errs
}

func generateRequiredRoleBinding(resourceNameSpace string, subjects []rbacv1.Subject, role string) *rbacv1.RoleBinding {
	roleBindingName := utils.GenerateClustersetResourceRoleBindingName(role)

	var labels = make(map[string]string)
	labels[utils.ClusterSetLabel] = "true"
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleBindingName,
			Namespace: resourceNameSpace,
			Labels:    labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     role,
		},
		Subjects: subjects,
	}
}
