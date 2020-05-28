package clusterrbac

import (
	"testing"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
)

func TestRule(t *testing.T) {
	NewRule("get", "list", "watch").Names("test").Groups(clusterv1beta1.GroupName).Resources("clusteractions").RuleOrDie()
	NewClusterBinding("test").Groups(clusterv1beta1.GroupName).Users("admin").SAs("default", "admin_sa").BindingOrDie()
	NewRoleBinding("clusterName", "clusterNamespace").Users("hcm:clusters:" + "clusterNamespace" + ":" + "clusterName").BindingOrDie()
}
