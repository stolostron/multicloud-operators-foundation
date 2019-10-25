// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package etcd

import (
	"context"
	"fmt"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	klusterlet "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/client"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers"
	printersinternal "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/internalversion"
	printerstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/printers/storage"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against clusterstatuses.
func NewREST(optsGetter generic.RESTOptionsGetter, config klusterlet.KlusterletClientConfig) (*REST, klusterlet.ConnectionInfoGetter) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &mcm.ClusterStatus{} },
		NewListFunc:              func() runtime.Object { return &mcm.ClusterStatusList{} },
		PredicateFunc:            clusterstatus.MatchClusterStatus,
		DefaultQualifiedResource: mcm.Resource("clusterstatuses"),

		CreateStrategy:      clusterstatus.Strategy,
		UpdateStrategy:      clusterstatus.Strategy,
		DeleteStrategy:      clusterstatus.Strategy,
		ReturnDeletedObject: true,

		TableConvertor: printerstorage.TableConvertor{TablePrinter: printers.NewTablePrinter().With(printersinternal.AddHandlers)},
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: clusterstatus.GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	clusterStatusRest := &REST{store}

	// Build a NodeGetter that looks up nodes using the REST handler
	clusterGetter := klusterlet.ClusterGetterFunc(
		func(ctx context.Context, name string, options metav1.GetOptions) (*mcm.ClusterStatus, error) {
			obj, err := clusterStatusRest.Get(ctx, name, &options)
			if err != nil {
				return nil, err
			}
			node, ok := obj.(*mcm.ClusterStatus)
			if !ok {
				return nil, fmt.Errorf("unexpected type %T", obj)
			}

			return node, nil
		})

	connectionInfoGetter, err := klusterlet.NewClusterConnectionInfoGetter(clusterGetter, config)
	if err != nil {
		return clusterStatusRest, nil
	}

	return clusterStatusRest, connectionInfoGetter
}
