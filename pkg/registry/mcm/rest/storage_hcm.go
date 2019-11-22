// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rest

import (
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	hcmv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	klusterlet "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/klusterlet/client"
	clusterjoinstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/clusterjoinrequest/etcd"
	clusterstatusstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/etcd"
	clusterstatusrest "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/rest"
	placementbindingstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/placementbinding/etcd"
	placementpolicystorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/placementpolicy/etcd"
	resourceviewstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/resourceview/etcd"
	workstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/work/etcd"
	workresultstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/work/mongo"
	worksetstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/registry/mcm/workset/etcd"
	mcmstorage "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/storage"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/klog"
)

type RESTStorageProvider struct{}

func (p RESTStorageProvider) NewRESTStorage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	restOptionsGetter generic.RESTOptionsGetter,
	storageOptions *mcmstorage.StorageOptions,
	clientConfig klusterlet.KlusterletClientConfig) (genericapiserver.APIGroupInfo, bool) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(hcmv1alpha1.GroupName, api.Scheme, api.ParameterCodec, api.Codecs)
	// If you add a version here, be sure to add an entry in `k8s.io/kubernetes/cmd/kube-apiserver/app/aggregator.go with specific priorities.
	// TODO refactor the plumbing to provide the information in the APIGroupInfo

	if apiResourceConfigSource.VersionEnabled(hcmv1alpha1.SchemeGroupVersion) {
		apiGroupInfo.VersionedResourcesStorageMap[hcmv1alpha1.SchemeGroupVersion.Version] = p.v1alpha1Storage(
			apiResourceConfigSource, restOptionsGetter, storageOptions, clientConfig)
	}

	return apiGroupInfo, true
}

func (p RESTStorageProvider) v1alpha1Storage(
	apiResourceConfigSource serverstorage.APIResourceConfigSource,
	optsGetter generic.RESTOptionsGetter,
	storageOptions *mcmstorage.StorageOptions,
	clientConfig klusterlet.KlusterletClientConfig) map[string]rest.Storage {
	storage := map[string]rest.Storage{}

	// work
	workStorage, workStatusStorage, err := workstorage.NewREST(optsGetter, storageOptions)
	if err == nil {
		storage["works"] = workStorage
		storage["works/status"] = workStatusStorage
	} else {
		klog.Errorf("failed to create works: %v", err)
	}
	workresultStorage, err := workresultstorage.NewWorkResultRest(optsGetter, storageOptions)
	if err == nil {
		storage["works/result"] = workresultStorage
	} else {
		klog.Errorf("failed to create works/result: %v", err)
	}

	// workset storage
	worksetStorage, worksetStatusStorage := worksetstorage.NewREST(optsGetter)
	storage["worksets"] = worksetStorage
	storage["worksets/status"] = worksetStatusStorage

	// workset storage
	resourceviewStorage, resourceviewStatusStorage, err := resourceviewstorage.NewREST(optsGetter, storageOptions)
	if err == nil {
		storage["resourceviews"] = resourceviewStorage
		storage["resourceviews/status"] = resourceviewStatusStorage
	} else {
		klog.Errorf("failed to create resourceview: %v", err)
	}

	// cluster status storage
	clusterstatusStorage, connection := clusterstatusstorage.NewREST(optsGetter, clientConfig)
	storage["clusterstatuses"] = clusterstatusStorage

	// cluster log storage
	if connection != nil {
		clusterlog := clusterstatusrest.NewLogRest(optsGetter, connection)
		clusterMonitor := clusterstatusrest.NewMonitorRest(optsGetter, connection)
		storage["clusterstatuses/log"] = clusterlog
		storage["clusterstatuses/monitor"] = clusterMonitor
	}

	// hcm join storage
	clusterjoinStorage, clusterjoinStatusStorage := clusterjoinstorage.NewREST(optsGetter)
	storage["clusterjoinrequests"] = clusterjoinStorage
	storage["clusterjoinrequests/status"] = clusterjoinStatusStorage

	placementpolicyStorage, placementpolicyStatusStorage := placementpolicystorage.NewREST(optsGetter)
	storage["placementpolicies"] = placementpolicyStorage
	storage["placementpolicies/status"] = placementpolicyStatusStorage

	placementBindingStorage := placementbindingstorage.NewREST(optsGetter)
	storage["placementbindings"] = placementBindingStorage

	return storage
}

func (p RESTStorageProvider) GroupName() string {
	return mcm.GroupName
}
