// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clustersetmapper

import (
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	utils "github.com/stolostron/multicloud-operators-foundation/pkg/utils/clusterset"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var managedclusterGroup = "cluster.open-cluster-management.io"
var hiveGroup = "hive.openshift.io"
var managedClusterViewGroup = "clusterview.open-cluster-management.io"
var registerGroup = "register.open-cluster-management.io"
var submarinerGroup = "submariner.io"

// buildAdminRoleRules builds the clustesetadminroles
func buildAdminRole(clustersetName, clusteroleName string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusteroleName,
			Labels: map[string]string{
				utils.ClusterSetLabel: clustersetName,
				utils.ClusterSetRole:  "admin",
			},
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("get", "update").
				Groups(managedclusterGroup).
				Resources("managedclustersets").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclustersets/join").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclustersets/bind").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclusters").
				RuleOrDie(),
			//TODO
			// We will restrict the update permission only for authenticated clusterset in another pr
			helpers.NewRule("update").
				Groups(registerGroup).
				Resources("managedclusters/accept").
				RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclustersets").
				RuleOrDie(),
			helpers.NewRule("create", "get", "update").
				Groups(submarinerGroup).
				Resources("brokers").
				Names(clustersetName).
				RuleOrDie(),
		},
	}
}

// buildViewRoleRules builds the clustersetviewroles
func buildViewRole(clustersetName, clusteroleName string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusteroleName,
			Labels: map[string]string{
				utils.ClusterSetLabel: clustersetName,
				utils.ClusterSetRole:  "view",
			},
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("get").
				Groups(managedclusterGroup).
				Resources("managedclustersets").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclustersets").
				RuleOrDie(),
		},
	}
}
