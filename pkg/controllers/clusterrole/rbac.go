// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterrole

import (
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var managedclusterGroup = "cluster.open-cluster-management.io"
var managedClusterViewGroup = "clusterview.open-cluster-management.io"

// buildAdminRole builds the admin clusterrole for the cluster.
// The users with this clusterrole has admin permission(create/update/delete/...) for the cluster.
func buildAdminRole(clusterName string) *rbacv1.ClusterRole {
	adminRoleName := utils.GenerateClusterRoleName(clusterName, "admin")
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: adminRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("create", "get", "list", "watch", "update", "patch", "delete").
				Groups(managedclusterGroup).
				Resources("managedclusters").
				Names(clusterName).
				RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclusters", "kubevirtprojects").
				RuleOrDie(),
		},
	}

}

// buildViewRole builds the view clusterrole for the cluster.
// The users with this clusterrole has admin permission(get/list/watch...) for the cluster.
func buildViewRole(clusterName string) *rbacv1.ClusterRole {
	viewRoleName := utils.GenerateClusterRoleName(clusterName, "view")
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: viewRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("get", "list", "watch").
				Groups(managedclusterGroup).
				Resources("managedclusters").
				Names(clusterName).RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclusters", "kubevirtprojects").
				RuleOrDie(),
		},
	}
}
