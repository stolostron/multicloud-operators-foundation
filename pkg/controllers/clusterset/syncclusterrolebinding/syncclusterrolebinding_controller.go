package syncclusterrolebinding

import (
	"context"
	"strings"
	"time"

	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"

	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	clustersetutils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

//This controller apply clusterset related clusterrolebinding based on clustersetToClusters and clustersetAdminToSubject map
type Reconciler struct {
	kubeClient           kubernetes.Interface
	clusterSetAdminCache *cache.AuthCache
	clusterSetViewCache  *cache.AuthCache
	clustersetToClusters *helpers.ClusterSetMapper
}

func NewReconciler(kubeClient kubernetes.Interface,
	clusterSetAdminCache *cache.AuthCache,
	clusterSetViewCache *cache.AuthCache,
	clustersetToClusters *helpers.ClusterSetMapper) Reconciler {
	return Reconciler{
		kubeClient:           kubeClient,
		clusterSetAdminCache: clusterSetAdminCache,
		clusterSetViewCache:  clusterSetViewCache,
		clustersetToClusters: clustersetToClusters,
	}
}

// Run a routine to sync the clusterrolebinding periodically.
func (r *Reconciler) Run(period time.Duration) {
	go utilwait.Forever(r.reconcile, period)
}

func (r *Reconciler) reconcile() {
	ctx := context.Background()
	clustersetToAdminSubjects := clustersetutils.GenerateClustersetSubjects(r.clusterSetAdminCache)
	clustersetToViewSubjects := clustersetutils.GenerateClustersetSubjects(r.clusterSetViewCache)
	r.syncManagedClusterClusterroleBinding(ctx, clustersetToAdminSubjects, "admin")
	r.syncManagedClusterClusterroleBinding(ctx, clustersetToViewSubjects, "view")
}

func (r *Reconciler) syncManagedClusterClusterroleBinding(ctx context.Context, clustersetToSubject map[string][]rbacv1.Subject, role string) {
	clusterToSubject := clustersetutils.GenerateObjectSubjectMap(r.clustersetToClusters, clustersetToSubject)

	//apply all disired clusterrolebinding
	for clusterName, subjects := range clusterToSubject {
		clustersetName := r.clustersetToClusters.GetObjectClusterset(clusterName)
		requiredClusterRoleBinding := generateRequiredClusterRoleBinding(clusterName, subjects, clustersetName, role)
		err := utils.ApplyClusterRoleBinding(ctx, r.kubeClient, requiredClusterRoleBinding)
		if err != nil {
			klog.Errorf("Failed to apply clusterrolebinding: %v, error:%v", requiredClusterRoleBinding.Name, err)
		}
	}

	//Delete clusterrolebinding
	//List Clusterset related clusterrolebinding
	clusterRoleBindingList, err := r.kubeClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{LabelSelector: clusterv1beta1.ClusterSetLabel})
	if err != nil {
		klog.Errorf("Error to list clusterrolebinding. error:%v", err)
	}
	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		curClusterRoleBinding := clusterRoleBinding
		// Only handle managedcluster clusterrolebinding
		if !utils.IsManagedClusterClusterrolebinding(curClusterRoleBinding.Name, role) {
			continue
		}

		curClusterName := getClusterNameInClusterrolebinding(curClusterRoleBinding.Name)
		if curClusterName == "" {
			continue
		}
		if _, ok := clusterToSubject[curClusterName]; !ok {
			err := r.kubeClient.RbacV1().ClusterRoleBindings().Delete(ctx, curClusterRoleBinding.Name, metav1.DeleteOptions{})
			if err != nil {
				klog.Errorf("Error to delete clusterrolebinding, error:%v", err)
			}
		}
	}
}

// The clusterset related managedcluster clusterrolebinding format should be: open-cluster-management:managedclusterset:"admin":managedcluster:cluster1
// So the last field should be managedcluster name.
func getClusterNameInClusterrolebinding(clusterrolebindingName string) string {
	splitName := strings.Split(clusterrolebindingName, ":")
	l := len(splitName)
	if l <= 0 {
		return ""
	}
	return splitName[l-1]
}

func generateRequiredClusterRoleBinding(clusterName string, subjects []rbacv1.Subject, clustersetName string, role string) *rbacv1.ClusterRoleBinding {
	clusterRoleBindingName := utils.GenerateClustersetClusterRoleBindingName(clusterName, role)
	clusterRoleName := utils.GenerateClusterRoleName(clusterName, role)

	var labels = make(map[string]string)
	labels[clusterv1beta1.ClusterSetLabel] = clustersetName
	labels[clustersetutils.ClusterSetRole] = role
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
