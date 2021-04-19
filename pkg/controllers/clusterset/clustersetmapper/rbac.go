// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clustersetmapper

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/controllers/clusterrbac"
	rbacv1 "k8s.io/api/rbac/v1"
)

var managedclusterGroup = "cluster.open-cluster-management.io"
var hiveGroup = "hive.openshift.io"
var managedClusterViewGroup = "clusterview.open-cluster-management.io"
var registerGroup = "register.open-cluster-management.io"

// buildAdminRoleRules builds the clustesetadminroles
func buildAdminRoleRules(clustersetName string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		clusterrbac.NewRule("get", "update").
			Groups(managedclusterGroup).
			Resources("managedclustersets").
			Names(clustersetName).
			RuleOrDie(),
		clusterrbac.NewRule("create").
			Groups(managedclusterGroup).
			Resources("managedclustersets/join").
			Names(clustersetName).
			RuleOrDie(),
		clusterrbac.NewRule("create").
			Groups(managedclusterGroup).
			Resources("managedclustersets/bind").
			Names(clustersetName).
			RuleOrDie(),
		clusterrbac.NewRule("create").
			Groups(managedclusterGroup).
			Resources("managedclusters").
			RuleOrDie(),
		//TODO
		// We will restrict the update permission only for authenticated clusterset in another pr
		clusterrbac.NewRule("update").
			Groups(registerGroup).
			Resources("managedclusters/accept").
			RuleOrDie(),
		clusterrbac.NewRule("get", "list", "watch").
			Groups(managedClusterViewGroup).
			Resources("managedclustersets").
			RuleOrDie(),
	}
}

// buildViewRoleRules builds the clustersetviewroles
func buildViewRoleRules(clustersetName string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		clusterrbac.NewRule("get").
			Groups(managedclusterGroup).
			Resources("managedclustersets").
			Names(clustersetName).
			RuleOrDie(),
		clusterrbac.NewRule("get", "list", "watch").
			Groups(managedClusterViewGroup).
			Resources("managedclustersets").
			RuleOrDie(),
	}
}
