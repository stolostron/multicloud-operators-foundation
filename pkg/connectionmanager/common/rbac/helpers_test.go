// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rbac

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
)

func TestRule(t *testing.T) {
	NewRule("get", "list", "watch").Names("test").Groups(mcm.GroupName).Resources("works").RuleOrDie()
	NewClusterBinding("test").Groups(mcm.GroupName).Users("admin").SAs("default", "admin_sa").BindingOrDie()
	NewRoleBinding("clusterName", "clusterNamespace").Users("hcm:clusters:" + "clusterNamespace" + ":" + "clusterName").BindingOrDie()
}
