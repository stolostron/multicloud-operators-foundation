// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package cluster

import (
	"context"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// Registry is an interface implemented by things that know how to store Cluster objects.
type Registry interface {
	ListClusters(ctx context.Context, options *metainternalversion.ListOptions) (*clusterregistry.ClusterList, error)
	WatchCluster(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error)
	GetCluster(ctx context.Context, name string, options *metav1.GetOptions) (*clusterregistry.Cluster, error)
	CreateCluster(
		ctx context.Context,
		cluster *clusterregistry.Cluster, createValidation rest.ValidateObjectFunc) (*clusterregistry.Cluster, error)
	UpdateCluster(
		ctx context.Context,
		cluster *clusterregistry.Cluster,
		createValidation rest.ValidateObjectFunc,
		updateValidation rest.ValidateObjectUpdateFunc) (*clusterregistry.Cluster, error)
	DeleteCluster(ctx context.Context, name string) error
}

// storage puts strong typing around storage calls
type storage struct {
	rest.StandardStorage
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched
// types will panic.
func NewRegistry(s rest.StandardStorage) Registry {
	return &storage{s}
}

func (s *storage) ListClusters(
	ctx context.Context, options *metainternalversion.ListOptions) (*clusterregistry.ClusterList, error) {
	obj, err := s.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*clusterregistry.ClusterList), nil
}

func (s *storage) WatchCluster(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return s.Watch(ctx, options)
}

func (s *storage) GetCluster(ctx context.Context, name string, options *metav1.GetOptions) (*clusterregistry.Cluster, error) {
	obj, err := s.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), nil
}

func (s *storage) CreateCluster(
	ctx context.Context,
	cluster *clusterregistry.Cluster, createValidation rest.ValidateObjectFunc) (*clusterregistry.Cluster, error) {
	obj, err := s.Create(ctx, cluster, rest.ValidateAllObjectFunc, &metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), nil
}

func (s *storage) UpdateCluster(
	ctx context.Context, cluster *clusterregistry.Cluster, createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc) (*clusterregistry.Cluster, error) {
	obj, _, err := s.Update(
		ctx, cluster.Name,
		rest.DefaultUpdatedObjectInfo(cluster), createValidation, updateValidation, false, &metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return obj.(*clusterregistry.Cluster), nil
}

func (s *storage) DeleteCluster(ctx context.Context, name string) error {
	_, _, err := s.Delete(ctx, name, rest.ValidateAllObjectFunc, &metav1.DeleteOptions{})
	return err
}
