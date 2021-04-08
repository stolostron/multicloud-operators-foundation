package rbac

import (
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func APIGroupMatches(rule *rbacv1.PolicyRule, requestedGroup string) bool {
	for _, ruleGroup := range rule.APIGroups {
		if ruleGroup == rbacv1.APIGroupAll {
			return true
		}
		if ruleGroup == requestedGroup {
			return true
		}
	}

	return false
}

func ResourceMatches(rule *rbacv1.PolicyRule, combinedRequestedResource, requestedSubresource string) bool {
	for _, ruleResource := range rule.Resources {
		// if everything is allowed, we match
		if ruleResource == rbacv1.ResourceAll {
			return true
		}
		// if we have an exact match, we match
		if ruleResource == combinedRequestedResource {
			return true
		}

		// We can also match a */subresource.
		// if there isn't a subresource, then continue
		if len(requestedSubresource) == 0 {
			continue
		}
		// if the rule isn't in the format */subresource, then we don't match, continue
		if len(ruleResource) == len(requestedSubresource)+2 &&
			strings.HasPrefix(ruleResource, "*/") &&
			strings.HasSuffix(ruleResource, requestedSubresource) {
			return true

		}
	}

	return false
}

func VerbMatches(rule *rbacv1.PolicyRule, requestedVerb string) bool {
	for _, verb := range rule.Verbs {
		if verb == requestedVerb {
			return true
		}
	}

	return false
}

func GetViewResourceFromClusterRole(clusterRole *rbacv1.ClusterRole, group, resource string) (sets.String, bool) {
	names := sets.NewString()
	all := false
	for _, rule := range clusterRole.Rules {
		if !APIGroupMatches(&rule, group) {
			continue
		}

		if !VerbMatches(&rule, "get") && !VerbMatches(&rule, "list") && !VerbMatches(&rule, "*") {
			continue
		}

		if len(rule.ResourceNames) == 0 {
			all = true
			return names, all
		}

		if !ResourceMatches(&rule, resource, "") {
			continue
		}

		names.Insert(rule.ResourceNames...)
	}
	return names, all
}

func GetAdminResourceFromClusterRole(clusterRole *rbacv1.ClusterRole, group, resource string) (sets.String, bool) {
	names := sets.NewString()
	all := false
	for _, rule := range clusterRole.Rules {
		if !APIGroupMatches(&rule, group) {
			continue
		}

		if !VerbMatches(&rule, "update") && !VerbMatches(&rule, "*") {
			continue
		}

		if len(rule.ResourceNames) == 0 {
			all = true
			return names, all
		}

		if !ResourceMatches(&rule, resource, "") {
			continue
		}

		names.Insert(rule.ResourceNames...)
	}
	return names, all
}
