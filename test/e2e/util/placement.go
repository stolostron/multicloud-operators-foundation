package util

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

func newPlacement(namespace, name string, clusterSets []string, selectedLabels map[string]string) *clusterv1beta1.Placement {
	return &clusterv1beta1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: clusterv1beta1.PlacementSpec{
			ClusterSets: clusterSets,
			Predicates: []clusterv1beta1.ClusterPredicate{
				{
					RequiredClusterSelector: clusterv1beta1.ClusterSelector{
						LabelSelector: metav1.LabelSelector{
							MatchLabels: selectedLabels,
						},
					},
				},
			},
			Tolerations: []clusterv1beta1.Toleration{
				{
					Key:      "cluster.open-cluster-management.io/unreachable",
					Operator: clusterv1beta1.TolerationOpExists,
				},
			},
		},
	}
}

func CreatePlacement(client clusterclient.Interface, namespace, name string, clusterSets []string, selectedLabels map[string]string) error {
	_, err := client.ClusterV1beta1().Placements(namespace).Create(context.TODO(), newPlacement(namespace, name, clusterSets, selectedLabels), metav1.CreateOptions{})
	return err
}

func DeletePlacement(client clusterclient.Interface, namespace, name string) error {
	err := client.ClusterV1beta1().Placements(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}
