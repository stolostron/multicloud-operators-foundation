// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package clusterrole

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/clusterrbac"
	rbacv1 "k8s.io/api/rbac/v1"
)

var managedclusterGroup = "cluster.open-cluster-management.io"

// buildAdminRoleRules builds the clusteadminroles
func buildAdminRoleRules(clusterName string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		clusterrbac.NewRule("create", "get", "list", "watch", "update", "patch", "delete").
			Groups(managedclusterGroup).
			Resources("managedclusters").
			Names(clusterName).
			RuleOrDie(),
	}
}

// buildViewRoleRules builds the clusteviewroles
func buildViewRoleRules(clusterName string) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		clusterrbac.NewRule("get", "list", "watch").
			Groups(managedclusterGroup).
			Resources("managedclusters").
			Names(clusterName).RuleOrDie(),
	}
}
