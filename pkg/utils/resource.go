// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

package utils

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func FormatQuatityToMi(q resource.Quantity) resource.Quantity {
	raw, _ := q.AsInt64()
	raw = raw / (1024 * 1024)
	rq, err := resource.ParseQuantity(fmt.Sprintf("%dMi", raw))
	if err != nil {
		return q
	}
	return rq
}

func FormatQuatityToGi(q resource.Quantity) resource.Quantity {
	raw, _ := q.AsInt64()
	raw = raw / (1024 * 1024 * 1024)
	rq, err := resource.ParseQuantity(fmt.Sprintf("%dGi", raw))
	if err != nil {
		return q
	}
	return rq
}

func IsClusterExit(ClusterName string, clusters []*v1alpha1.Cluster) bool {
	if len(clusters) == 0 || ClusterName == "" {
		return false
	}
	for _, cl := range clusters {
		if cl.Name == ClusterName {
			return true
		}
	}
	return false
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(pod *apiv1.Pod) (reqs map[apiv1.ResourceName]resource.Quantity, limits map[apiv1.ResourceName]resource.Quantity, err error) {
	reqs, limits = map[apiv1.ResourceName]resource.Quantity{}, map[apiv1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = *quantity.Copy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = *quantity.Copy()
			}
		}
	}
	return
}
