package helpers

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
)

func TestRule(t *testing.T) {
	NewRule("get", "list", "watch").Names("test").Groups(clusterv1beta1.GroupName).Resources("managedclusteractions").RuleOrDie()
	NewClusterBinding("test").Groups(clusterv1beta1.GroupName).Users("admin").SAs("default", "admin_sa").BindingOrDie()
	NewRoleBinding("clusterName", "clusterNamespace").Users("hcm:clusters:" + "clusterNamespace" + ":" + "clusterName").BindingOrDie()

	// Test new Subjects() method
	subjects := []rbacv1.Subject{
		{
			Kind:     rbacv1.UserKind,
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "test-user",
		},
	}
	NewRoleBindingForClusterRole("test-role", "test-namespace").Subjects(subjects...).BindingOrDie()
}
