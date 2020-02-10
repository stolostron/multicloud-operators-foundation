// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	v1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
)

// ConvertLabels returns label
func ConvertLabels(labelSelector *metav1.LabelSelector) (labels.Selector, error) {
	if labelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return labels.Nothing(), err
		}

		return selector, nil
	}

	return labels.Everything(), nil
}

// GetWorkSetFromWork return worksets based on work
func GetWorkSetFromWork(work *v1alpha1.Work, worksets []*v1alpha1.WorkSet) ([]*v1alpha1.WorkSet, error) {
	var filteredWorksets []*v1alpha1.WorkSet
	for _, workset := range worksets {
		selector, err := metav1.LabelSelectorAsSelector(workset.Spec.Selector)
		if err != nil {
			// this should not happen if the workset passed validation
			return nil, err
		}

		// If a work with a nil or empty selector creeps in, it should match nothing, not everything.
		if selector.Empty() || !selector.Matches(labels.Set(work.Labels)) {
			continue
		}
		filteredWorksets = append(filteredWorksets, workset)
	}

	if len(filteredWorksets) == 0 {
		return nil, fmt.Errorf("could not find work set for work %s with labels: %v", work.Name, work.Labels)
	}

	return filteredWorksets, nil
}

func FilterHealthyClusters(clusters []*clusterv1alpha1.Cluster) []*clusterv1alpha1.Cluster {
	filteredClusters := []*clusterv1alpha1.Cluster{}
	for _, cluster := range clusters {
		if len(cluster.Status.Conditions) == 0 {
			continue
		}
		if cluster.Status.Conditions[0].Type != clusterv1alpha1.ClusterOK {
			continue
		}
		filteredClusters = append(filteredClusters, cluster)
	}
	return filteredClusters
}
