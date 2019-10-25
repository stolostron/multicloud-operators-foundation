// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"k8s.io/klog"

	clusterrest "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/cluster-registry/rest"
	hcmrest "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/rest"
	mcmstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/storage"

	klusterlet "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/client"
	"k8s.io/apiserver/pkg/registry/generic"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/storage"
)

func installHCMAPIs(
	g *genericapiserver.GenericAPIServer,
	optsGetter generic.RESTOptionsGetter,
	apiResourceConfigSource storage.APIResourceConfigSource,
	options *mcmstorage.StorageOptions,
	clientConfig klusterlet.KlusterletClientConfig) {
	hcmStorageProvider := hcmrest.RESTStorageProvider{}
	hcmGroupInfo, shouldInstallGroup := hcmStorageProvider.NewRESTStorage(
		apiResourceConfigSource, optsGetter, options, clientConfig)
	if !shouldInstallGroup {
		return
	}

	if err := g.InstallAPIGroup(&hcmGroupInfo); err != nil {
		klog.Fatalf("Error in registering group versions: %v", err)
	}

	clusterStorageProvider := clusterrest.RESTStorageProvider{}
	clusterGroupInfo, shouldInstallGroup := clusterStorageProvider.NewRESTStorage(apiResourceConfigSource, optsGetter)
	if !shouldInstallGroup {
		return
	}

	if err := g.InstallAPIGroup(&clusterGroupInfo); err != nil {
		klog.Fatalf("Error in registering group versions: %v", err)
	}
}
