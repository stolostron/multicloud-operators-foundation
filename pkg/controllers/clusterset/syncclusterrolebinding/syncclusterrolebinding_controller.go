package syncclusterrolebinding

import (
	"context"
	"strings"
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

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/controllers/clusterset/clusterrolebinding"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//This controller apply clusterset related clusterrolebinding based on clustersetToClusters and clustersetToSubject map
type Reconciler struct {
	client               client.Client
	scheme               *runtime.Scheme
	clustersetToSubject  *helpers.ClustersetSubjectsMapper
	clustersetToClusters *helpers.ClusterSetMapper
}

func SetupWithManager(mgr manager.Manager, clustersetToSubject *helpers.ClustersetSubjectsMapper, clustersetToClusters *helpers.ClusterSetMapper) error {
	if err := add(mgr, newReconciler(mgr, clustersetToSubject, clustersetToClusters)); err != nil {
		klog.Errorf("Failed to create ClusterRoleBinding controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clustersetToSubject *helpers.ClustersetSubjectsMapper, clustersetToClusters *helpers.ClusterSetMapper) reconcile.Reconciler {
	return &Reconciler{
		client:               mgr.GetClient(),
		scheme:               mgr.GetScheme(),
		clustersetToSubject:  clustersetToSubject,
		clustersetToClusters: clustersetToClusters,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("clusterinfo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//reconcile every 10s
	reconcile := reconcile.Result{RequeueAfter: time.Duration(10) * time.Second}

	ctx := context.Background()
	clusterToSubject := generateClusterSubjectMap(r.clustersetToClusters, r.clustersetToSubject)
	//apply all disired clusterrolebinding
	errs := []error{}
	for cluster, subjects := range clusterToSubject {
		requiredClusterRoleBinding := generateRequiredClusterRoleBinding(cluster, subjects)
		err := utils.ApplyClusterRoleBinding(ctx, r.client, requiredClusterRoleBinding)
		if err != nil {
			klog.Errorf("Failed to apply clusterrolebinding: %v, error:%v", requiredClusterRoleBinding, err)
			errs = append(errs, err)
		}
	}

	//Delete clusterrolebinding
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}

	//List Clusterset related clusterrolebinding
	matchExpressions := metav1.LabelSelectorRequirement{Key: clusterrolebinding.ClusterSetLabel, Operator: metav1.LabelSelectorOpExists}
	labelSelector := metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{matchExpressions}}
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return reconcile, err
	}

	err = r.client.List(ctx, clusterRoleBindingList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return reconcile, err
	}
	for _, curClusterRoleBinding := range clusterRoleBindingList.Items {
		curClusterName := getClusterNameInClusterrolebinding(curClusterRoleBinding)
		if curClusterName == "" {
			klog.Errorf("Failed to get cluster name from clusterrolebinding. clusterrolebinding:%v", curClusterRoleBinding)
			continue
		}
		if _, ok := clusterToSubject[curClusterName]; !ok {
			err = r.client.Delete(ctx, &curClusterRoleBinding)
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}
	}
	return reconcile, utils.NewMultiLineAggregate(errs)
}

func getClusterNameInClusterrolebinding(clusterrolebinding rbacv1.ClusterRoleBinding) string {
	splitName := strings.Split(clusterrolebinding.Name, ":")
	l := len(splitName)
	if l <= 0 {
		return ""
	}
	return splitName[l-1]
}
func generateClusterSubjectMap(clustersetToClusters *helpers.ClusterSetMapper, clustersetToSubject *helpers.ClustersetSubjectsMapper) map[string][]rbacv1.Subject {
	var clusterToSubject = make(map[string][]rbacv1.Subject)

	for clusterset, subjects := range clustersetToSubject.GetMap() {
		if clusterset == "*" {
			continue
		}
		clusters := clustersetToClusters.GetClustersOfClusterSet(clusterset)
		for _, cluster := range clusters.List() {
			clusterToSubject[cluster] = utils.Mergesubjects(clusterToSubject[cluster], subjects)
		}
	}

	if len(clustersetToSubject.Get("*")) == 0 {
		return clusterToSubject
	}
	//if clusterset is "*", should map this subjects to all clusters
	allClustersetToClusters := clustersetToClusters.GetAllClusterSetToClusters()
	for _, clusters := range allClustersetToClusters {
		subjects := clustersetToSubject.Get("*")
		for _, cluster := range clusters.List() {
			clusterToSubject[cluster] = utils.Mergesubjects(clusterToSubject[cluster], subjects)
		}
	}
	return clusterToSubject
}

func generateRequiredClusterRoleBinding(clusterName string, subjects []rbacv1.Subject) *rbacv1.ClusterRoleBinding {
	clusterRoleBindingName := utils.GenerateClusterRoleBindingName(clusterName)
	clusterRoleName := utils.GenerateClusterRoleName(clusterName)

	var labels = make(map[string]string)
	labels[clusterrolebinding.ClusterSetLabel] = "true"
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterRoleBindingName,
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: subjects,
	}
}
