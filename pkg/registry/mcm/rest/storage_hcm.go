// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rest

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/api"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	mcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	mcmv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
	restv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/rest/v1alpha1"
	restv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/rest/v1beta1"
	mcmstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo/weave"
	"k8s.io/apiserver/pkg/registry/generic"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
)

type StorageProvider struct{}

func (p StorageProvider) NewRESTStorage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter,
	storageOptions *mcmstorage.Options,
	inserter *weave.ClusterTopologyInserter,
	clientConfig klusterlet.ClientConfig,
	aggregatorGetters *aggregator.Getters) (genericapiserver.APIGroupInfo, bool) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(mcm.GroupName, api.Scheme, api.ParameterCodec, api.Codecs)
	// If you add a version here, be sure to add an entry in `k8s.io/kubernetes/cmd/kube-apiserver/app/aggregator.go with
	// specific priorities.
	// TODO refactor the plumbing to provide the information in the APIGroupInfo

	if apiResourceConfigSource.VersionEnabled(mcmv1alpha1.SchemeGroupVersion) {
		apiGroupInfo.VersionedResourcesStorageMap[mcmv1alpha1.SchemeGroupVersion.Version] = restv1alpha1.Storage(
			restOptionsGetter, storageOptions, inserter, clientConfig, aggregatorGetters.V1alpha1InfoGetters)
	}
	if apiResourceConfigSource.VersionEnabled(mcmv1beta1.SchemeGroupVersion) {
		apiGroupInfo.VersionedResourcesStorageMap[mcmv1beta1.SchemeGroupVersion.Version] = restv1beta1.Storage(
			restOptionsGetter, storageOptions, clientConfig, aggregatorGetters.InfoGetters)
	}

	return apiGroupInfo, true
}

func (p StorageProvider) GroupName() string {
	return mcm.GroupName
}
