package helpers

import (
	"testing"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
)

func TestRule(t *testing.T) {
	NewRule("get", "list", "watch").Names("test").Groups(clusterv1beta1.GroupName).Resources("managedclusteractions").RuleOrDie()
	NewClusterBinding("test").Groups(clusterv1beta1.GroupName).Users("admin").SAs("default", "admin_sa").BindingOrDie()
	NewRoleBinding("clusterName", "clusterNamespace").Users("hcm:clusters:" + "clusterNamespace" + ":" + "clusterName").BindingOrDie()
}
