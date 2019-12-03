// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rest

import (
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/api"
	clusterstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/cluster-registry/cluster/etcd"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type RESTStorageProvider struct{}

func (p RESTStorageProvider) NewRESTStorage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, bool) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(
		clusterv1alpha1.SchemeGroupVersion.Group, api.Scheme, api.ParameterCodec, api.Codecs)
	// If you add a version here, be sure to add an entry in `k8s.io/kubernetes/cmd/kube-apiserver/app/aggregator.go with
	// specific priorities.
	// TODO refactor the plumbing to provide the information in the APIGroupInfo

	if apiResourceConfigSource.VersionEnabled(clusterv1alpha1.SchemeGroupVersion) {
		apiGroupInfo.VersionedResourcesStorageMap[clusterv1alpha1.SchemeGroupVersion.Version] = p.v1alpha1Storage(
			restOptionsGetter)
	}

	return apiGroupInfo, true
}

func (p RESTStorageProvider) v1alpha1Storage(optsGetter generic.RESTOptionsGetter) map[string]rest.Storage {
	storage := map[string]rest.Storage{}

	clusterStorage, clusterStatusStorage := clusterstorage.NewREST(optsGetter)

	// cluster storage
	storage["clusters"] = clusterStorage
	storage["clusters/status"] = clusterStatusStorage

	return storage
}

func (p RESTStorageProvider) GroupName() string {
	return clusterv1alpha1.SchemeGroupVersion.Group
}
