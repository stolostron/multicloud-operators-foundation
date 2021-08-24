package util

import (
	"context"
	clusterclient "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewManagedClusterSet(name string) *clusterv1alpha1.ManagedClusterSet {
	return &clusterv1alpha1.ManagedClusterSet{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: clusterv1alpha1.ManagedClusterSetSpec{},
	}
}

func NewManagedClusterSetBinding(namespace, name, clusterSetName string) *clusterv1alpha1.ManagedClusterSetBinding {
	return &clusterv1alpha1.ManagedClusterSetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: clusterv1alpha1.ManagedClusterSetBindingSpec{
			ClusterSet: clusterSetName,
		},
	}
}

func CreateManagedClusterSet(client clusterclient.Interface, name string) error {
	_, err := client.ClusterV1alpha1().ManagedClusterSets().Create(context.TODO(), NewManagedClusterSet(name), metav1.CreateOptions{})
	return err
}

func DeleteManagedClusterSet(client clusterclient.Interface, name string) error {
	err := client.ClusterV1alpha1().ManagedClusterSets().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func CreateManagedClusterSetBinding(client clusterclient.Interface, namespace, name, clusterSetName string) error {
	_, err := client.ClusterV1alpha1().ManagedClusterSetBindings(namespace).Create(context.TODO(),
		NewManagedClusterSetBinding(namespace, name, clusterSetName), metav1.CreateOptions{})
	return err
}

func DeleteManagedClusterSetBinding(client clusterclient.Interface, namespace, name string) error {
	err := client.ClusterV1alpha1().ManagedClusterSetBindings(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}
