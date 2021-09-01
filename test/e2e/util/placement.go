package util

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

func newPlacement(namespace, name string, clusterSets []string, selectedLabels map[string]string) *clusterv1alpha1.Placement {
	return &clusterv1alpha1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: clusterv1alpha1.PlacementSpec{
			ClusterSets: clusterSets,
			Predicates: []clusterv1alpha1.ClusterPredicate{
				{
					RequiredClusterSelector: clusterv1alpha1.ClusterSelector{
						LabelSelector: metav1.LabelSelector{
							MatchLabels: selectedLabels,
						},
					},
				},
			},
		},
	}
}

func CreatePlacement(client clusterclient.Interface, namespace, name string, clusterSets []string, selectedLabels map[string]string) error {
	_, err := client.ClusterV1alpha1().Placements(namespace).Create(context.TODO(), newPlacement(namespace, name, clusterSets, selectedLabels), metav1.CreateOptions{})
	return err
}

func DeletePlacement(client clusterclient.Interface, namespace, name string) error {
	err := client.ClusterV1alpha1().Placements(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}
