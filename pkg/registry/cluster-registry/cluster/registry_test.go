// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package cluster

import (
	"context"
	"fmt"
	"testing"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type standardStorage struct{}

func (s *standardStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("name should not be null")
	}
	return &clusterregistry.Cluster{}, nil
}
func (s *standardStorage) NewList() runtime.Object {
	return nil
}
func (s *standardStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return &clusterregistry.ClusterList{}, nil
}

func (s *standardStorage) New() runtime.Object {
	return nil
}

func (s *standardStorage) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	if obj == nil {
		return nil, fmt.Errorf("obj should not be null")
	}
	return &clusterregistry.Cluster{}, nil
}

func (s *standardStorage) Delete(
	ctx context.Context,
	name string,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return nil, true, nil
}
func (s *standardStorage) DeleteCollection(
	ctx context.Context,
	deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions,
	listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	return nil, nil
}
func (s *standardStorage) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return nil, nil
}
func (s *standardStorage) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	if len(name) == 0 {
		return nil, false, fmt.Errorf("name should not be null")
	}
	return &clusterregistry.Cluster{}, true, nil
}

func Test_storage_ListClusters(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	opt := metainternalversion.ListOptions{}
	std := &standardStorage{}
	s := NewRegistry(std)
	_, err := s.ListClusters(ctx, &opt)
	if err != nil {
		t.Errorf("Error list cluster. %s", err)
	}
	_, err = s.WatchCluster(ctx, &opt)
	if err != nil {
		t.Errorf("Error watch cluster. %s", err)
	}
	_, err = s.GetCluster(ctx, "n1", &metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error get cluster. %s", err)
	}
	_, err = s.GetCluster(ctx, "", &metav1.GetOptions{})
	if err == nil {
		t.Errorf("Should have error, when get cluster. %s", err)
	}
	_, err = s.CreateCluster(ctx, &clusterregistry.Cluster{}, nil)
	if err != nil {
		t.Errorf("Error create cluster. %s", err)
	}
	_, err = s.UpdateCluster(ctx, &clusterregistry.Cluster{}, nil, nil)
	if err == nil {
		t.Errorf("Error update cluster. %s", err)
	}
	_, err = s.UpdateCluster(ctx, &clusterregistry.Cluster{}, nil, nil)
	if err == nil {
		t.Errorf("Error update cluster. %s", err)
	}
	err = s.DeleteCluster(ctx, "c1")
	if err != nil {
		t.Errorf("Error delete cluster. %s", err)
	}
}
