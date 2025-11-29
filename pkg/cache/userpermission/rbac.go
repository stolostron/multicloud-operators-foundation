package userpermission

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// rulesGrantPermission checks if policy rules grant create permission on a specific resource
func rulesGrantPermission(rules []rbacv1.PolicyRule, resource, apiGroup string) bool {
	for _, rule := range rules {
		// Check if the rule applies to the correct API group
		if !ruleCoversAPIGroup(rule, apiGroup) {
			continue
		}

		// Check if the rule covers the resource
		if !ruleCoversResource(rule, resource) {
			continue
		}

		// Check if the rule grants create verb
		if ruleCoversVerb(rule, "create") {
			return true
		}
	}
	return false
}

// clusterRoleGrantsPermission checks if a ClusterRole grants create permission on a specific resource
func clusterRoleGrantsPermission(
	clusterRole *rbacv1.ClusterRole, resource, apiGroup string,
) bool {
	return rulesGrantPermission(clusterRole.Rules, resource, apiGroup)
}

// roleGrantsPermission checks if a Role grants create permission on a specific resource
func roleGrantsPermission(role *rbacv1.Role, resource, apiGroup string) bool {
	return rulesGrantPermission(role.Rules, resource, apiGroup)
}

// ruleCoversAPIGroup checks if a policy rule covers the specified API group
func ruleCoversAPIGroup(rule rbacv1.PolicyRule, apiGroup string) bool {
	for _, ruleAPIGroup := range rule.APIGroups {
		if ruleAPIGroup == "*" || ruleAPIGroup == apiGroup {
			return true
		}
	}
	return false
}

// ruleCoversResource checks if a policy rule covers the specified resource
func ruleCoversResource(rule rbacv1.PolicyRule, resource string) bool {
	for _, ruleResource := range rule.Resources {
		if ruleResource == "*" || ruleResource == resource {
			return true
		}
	}
	return false
}

// ruleCoversVerb checks if a policy rule covers the specified verb
func ruleCoversVerb(rule rbacv1.PolicyRule, verb string) bool {
	for _, ruleVerb := range rule.Verbs {
		if ruleVerb == "*" || ruleVerb == verb {
			return true
		}
	}
	return false
}
