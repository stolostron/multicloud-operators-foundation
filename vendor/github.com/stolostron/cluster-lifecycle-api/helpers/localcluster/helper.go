package localcluster

import (
	"strings"

	"github.com/stolostron/cluster-lifecycle-api/constants"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

func IsClusterSelfManaged(cluster *clusterv1.ManagedCluster) bool {
	if len(cluster.Labels) == 0 {
		return false
	}
	val, ok := cluster.Labels[constants.SelfManagedClusterLabelKey]
	return ok && strings.EqualFold(val, "true")
}
