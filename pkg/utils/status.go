// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// GetStatusFromCluster get status string from cluster
func GetStatusFromCluster(cluster v1alpha1.Cluster) (v1alpha1.ClusterConditionType, metav1.Time) {
	if len(cluster.Status.Conditions) == 0 {
		return "", metav1.Time{}
	}

	return cluster.Status.Conditions[0].Type, cluster.Status.Conditions[0].LastHeartbeatTime
}
