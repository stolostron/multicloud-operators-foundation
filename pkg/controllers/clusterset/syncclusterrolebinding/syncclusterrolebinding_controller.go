package syncclusterrolebinding

import (
	"context"
	"strings"
	"time"

	v1 "k8s.io/client-go/listers/rbac/v1"
	kubecache "k8s.io/client-go/tools/cache"

	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"

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

// This controller apply clusterset related clusterrolebinding based on clustersetToClusters and clustersetAdminToSubject map
type Reconciler struct {
	kubeClient                       kubernetes.Interface
	clusterRolebindingLister         v1.ClusterRoleBindingLister
	clusterRolebindingInformerSynced kubecache.InformerSynced
	clusterSetAdminCache             *cache.AuthCache
	clusterSetViewCache              *cache.AuthCache
	globalClustersetToClusters       *helpers.ClusterSetMapper
	clustersetToClusters             *helpers.ClusterSetMapper
}

func NewReconciler(kubeClient kubernetes.Interface,
	clusterRolebindingLister v1.ClusterRoleBindingLister,
	clusterRolebindingInformerSynced kubecache.InformerSynced,
	clusterSetAdminCache *cache.AuthCache,
	clusterSetViewCache *cache.AuthCache,
	globalClustersetToClusters *helpers.ClusterSetMapper,
	clustersetToClusters *helpers.ClusterSetMapper) Reconciler {
	return Reconciler{
		kubeClient:                       kubeClient,
		clusterRolebindingLister:         clusterRolebindingLister,
		clusterRolebindingInformerSynced: clusterRolebindingInformerSynced,
		clusterSetAdminCache:             clusterSetAdminCache,
		clusterSetViewCache:              clusterSetViewCache,
		globalClustersetToClusters:       globalClustersetToClusters,
		clustersetToClusters:             clustersetToClusters,
	}
}

// Run a routine to sync the clusterrolebinding periodically.
func (r *Reconciler) Run(ctx context.Context, period time.Duration) {
	if ok := kubecache.WaitForCacheSync(ctx.Done(), r.clusterRolebindingInformerSynced); !ok {
		klog.Errorf("failed to wait for kubernetes caches to sync")
		return
	}
	klog.Info("ClusterRolebinding informer synced")
	utilwait.UntilWithContext(ctx, r.reconcile, period)
}

func (r *Reconciler) reconcile(ctx context.Context) {
	clustersetToAdminSubjects := clustersetutils.GenerateClustersetSubjects(r.clusterSetAdminCache)
	clustersetToViewSubjects := clustersetutils.GenerateClustersetSubjects(r.clusterSetViewCache)
	r.syncManagedClusterClusterroleBinding(ctx, r.clustersetToClusters, clustersetToAdminSubjects, "admin")

	//Sync clusters view permission to the global clusterset users
	unionGlobalClustersetToCluster := r.clustersetToClusters.UnionObjectsInClusterSet(r.globalClustersetToClusters)
	r.syncManagedClusterClusterroleBinding(ctx, unionGlobalClustersetToCluster, clustersetToViewSubjects, "view")
}

// syncManagedClusterClusterroleBinding sync two(admin/view) clusterrolebindings for each clusters which are in a set.
// clustersetToSubject(map[string][]rbacv1.Subject) means the users/groups in "[]rbacv1.Subject" has admin/view permission to the clusterset
// r.clustersetToClusters(map[string]sets.String) means the clusterset include these clusters
// In current acm design, if a user has admin/view permissions to a clusterset, he/she should also has admin/view permissions to the clusters in the set.
// So we will generate two(admin/view) clusterrolebindings which grant the clusters admin/view permissions to clusterset users.
// For each cluster, it will have two clusterrolebindings, so if there are 2k clusters, 4k clusterrolebindings will be created.
func (r *Reconciler) syncManagedClusterClusterroleBinding(ctx context.Context, clustersetToClusters *helpers.ClusterSetMapper, clustersetToSubject map[string][]rbacv1.Subject, role string) {
	//clusterToSubject(map[<clusterName>][]rbacv1.Subject) means the users/groups in subject has permission for this cluster.
	//for each item, we will create a clusterrolebinding
	clusterToSubject := clustersetutils.GenerateObjectSubjectMap(clustersetToClusters, clustersetToSubject)

	//apply all disired clusterrolebinding
	for clusterName, subjects := range clusterToSubject {
		clustersetName := clustersetToClusters.GetObjectClusterset(clusterName)
		requiredClusterRoleBinding := generateRequiredClusterRoleBinding(clusterName, subjects, clustersetName, role)
		err := utils.ApplyClusterRoleBinding(ctx, r.kubeClient, r.clusterRolebindingLister, requiredClusterRoleBinding)
		if err != nil {
			klog.Errorf("Failed to apply clusterrolebinding: %v, error:%v", requiredClusterRoleBinding.Name, err)
		}
	}

	//Delete clusterrolebinding
	//List Clusterset related clusterrolebinding

	clusterSetLabelSelector, err := utils.ConvertLabels(
		&metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      clusterv1beta2.ClusterSetLabel,
					Operator: metav1.LabelSelectorOpExists,
				},
			},
		},
	)
	if err != nil {
		klog.Errorf("Error to build label selector. Error:%v", err)
	}

	clusterRoleBindings, err := r.clusterRolebindingLister.List(clusterSetLabelSelector)

	for _, clusterRoleBinding := range clusterRoleBindings {
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

	//We do not need to set global clusterset label. So we should skip it here.
	if clustersetName != clustersetutils.GlobalSetName {
		labels[clusterv1beta2.ClusterSetLabel] = clustersetName
	}
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
