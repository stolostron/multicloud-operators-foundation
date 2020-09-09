package syncclusterrolebinding

import (
	"context"
	"reflect"
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

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterSetLabel = "clusterset"
	clusterLabel    = "managedCluster"
)

//This controller apply clusterset related clusterrolebinding based on clustersetToClusters and clustersetToSubject map
type Reconciler struct {
	client               client.Client
	scheme               *runtime.Scheme
	clustersetToSubject  *helpers.ClustersetSubjectsMapper
	clustersetToClusters map[string][]string
}

func SetupWithManager(mgr manager.Manager, clustersetToSubject *helpers.ClustersetSubjectsMapper, clustersetToClusters map[string][]string) error {
	if err := add(mgr, newReconciler(mgr, clustersetToSubject, clustersetToClusters)); err != nil {
		klog.Errorf("Failed to create ClusterRoleBinding controller, %v", err)
		return err
	}
	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, clustersetToSubject *helpers.ClustersetSubjectsMapper, clustersetToClusters map[string][]string) reconcile.Reconciler {
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
	//reconcile every 60s
	reconcile := reconcile.Result{RequeueAfter: time.Duration(5) * time.Second}

	ctx := context.Background()
	clusterToSubject := generateClusterSubjectMap(r.clustersetToClusters, r.clustersetToSubject)
	//List Clusterset related clusterrolebinding
	matchExpressions := metav1.LabelSelectorRequirement{Key: clusterSetLabel, Operator: metav1.LabelSelectorOpExists}
	labelSelector := metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{matchExpressions}}
	selector, err := metav1.LabelSelectorAsSelector(&labelSelector)
	if err != nil {
		return reconcile, err
	}
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	err = r.client.List(ctx, clusterRoleBindingList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		return reconcile, client.IgnoreNotFound(err)
	}
	//mark all handled clusterrolebindings
	var curClusterRoleBindingMap = make(map[string]bool)

	for _, curClusterRoleBinding := range clusterRoleBindingList.Items {
		if _, ok := curClusterRoleBinding.Labels[clusterLabel]; !ok {
			klog.Warningf("can not get cluster label from ClusterRoleBinding %v ", curClusterRoleBinding)
			continue
		}
		curClusterName := curClusterRoleBinding.Labels[clusterLabel]
		curClusterRoleBindingMap[curClusterName] = true
		//should delete
		if _, ok := clusterToSubject[curClusterName]; !ok {
			err := r.DeleteClusterRoleBinding(ctx, curClusterRoleBinding.Name)
			if err != nil {
				return reconcile, err
			}
			continue
		}
		if equalSubjects(curClusterRoleBinding.Subjects, clusterToSubject[curClusterName]) {
			err = r.UpdateClusterRoleBinding(ctx, curClusterRoleBinding, clusterToSubject[curClusterName])
			if err != nil {
				return reconcile, err
			}
		}
	}

	//should create
	for curClusterName, curSubjects := range clusterToSubject {
		if _, ok := curClusterRoleBindingMap[curClusterName]; !ok {
			err = r.CreateClusterRoleBinding(ctx, curClusterName, curSubjects)
			if err != nil {
				return reconcile, err
			}
		}
	}
	return reconcile, nil
}

func equalSubjects(subjects1, subjects2 []rbacv1.Subject) bool {
	if len(subjects1) != len(subjects2) {
		return true
	}
	var subjectMap1 = make(map[rbacv1.Subject]bool)
	for _, curSubject := range subjects1 {
		subjectMap1[curSubject] = true
	}

	var subjectMap2 = make(map[rbacv1.Subject]bool)
	for _, curSubject := range subjects2 {
		subjectMap2[curSubject] = true
	}
	return !reflect.DeepEqual(subjectMap1, subjectMap2)
}

func generateClusterSubjectMap(clustersetToClusters map[string][]string, clustersetToSubject *helpers.ClustersetSubjectsMapper) map[string][]rbacv1.Subject {
	var clusterToSubject = make(map[string][]rbacv1.Subject)
	for clusterset, subjects := range clustersetToSubject.GetMap() {
		for _, cluster := range clustersetToClusters[clusterset] {
			clusterToSubject[cluster] = utils.Mergesubjects(clusterToSubject[cluster], subjects)
		}
	}
	return clusterToSubject
}

func generateClusterRoleName(clusterName string) string {
	return "open-cluster-management:admin:" + clusterName
}
func generateClusterRoleBindingName(clusterName string) string {
	return "open-cluster-management:clusterset:cluster:" + clusterName
}

func (r *Reconciler) CreateClusterRoleBinding(ctx context.Context, clusterName string, subjects []rbacv1.Subject) error {
	clusterRoleBindingName := generateClusterRoleBindingName(clusterName)
	clusterRoleName := generateClusterRoleName(clusterName)

	var labels = make(map[string]string)
	labels[clusterLabel] = clusterName
	labels[clusterLabel] = "true"
	rb := &rbacv1.ClusterRoleBinding{
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

	return r.client.Create(ctx, rb)
}

func (r *Reconciler) UpdateClusterRoleBinding(ctx context.Context, curClusterRoleBinding rbacv1.ClusterRoleBinding, subjects []rbacv1.Subject) error {
	curClusterRoleBinding.Subjects = subjects
	return r.client.Update(ctx, &curClusterRoleBinding)
}

func (r *Reconciler) DeleteClusterRoleBinding(ctx context.Context, clusterRoleBindingName string) error {
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
		},
	}
	return r.client.Delete(ctx, rb)
}
