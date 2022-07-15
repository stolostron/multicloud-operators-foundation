package helpers

import (
	"testing"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
)

func TestRule(t *testing.T) {
	NewRule("get", "list", "watch").Names("test").Groups(clusterv1beta1.GroupName).Resources("managedclusteractions").RuleOrDie()
	NewRule("get", "list", "watch").Names("test").Groups(clusterv1beta1.GroupName).Resources("managedclusteractions").URLs("xx").Rule()
	NewClusterBinding("test").Groups(clusterv1beta1.GroupName).Users("admin").SAs("default", "admin_sa").BindingOrDie()
	NewClusterBinding("test").Groups(clusterv1beta1.GroupName).Users("admin").SAs("default", "admin_sa").Binding()
	NewRoleBinding("clusterName", "clusterNamespace").Users("hcm:clusters:" + "clusterNamespace" + ":" + "clusterName").BindingOrDie()
	NewRoleBinding("clusterName", "clusterNamespace").Users("hcm:clusters:" + "clusterNamespace" + ":" + "clusterName").Binding()
}
