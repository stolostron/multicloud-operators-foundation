// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"fmt"

	aggregatorv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator/v1alpha1"
	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
	clusterjoinstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterjoinrequest/etcd"
	clusterstatusstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/etcd"
	clusterstatustopostorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/mongo"
	clusterstatusrest "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/rest"
	restv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/rest/v1alpha1"
	leadervotestorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/leadervote/etcd"
	placementbindingstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/placementbinding/etcd"
	placementpolicystorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/placementpolicy/etcd"
	resourceviewstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/resourceview/etcd"
	workstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/work/etcd"
	workresultstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/work/mongo"
	worksetstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/workset/etcd"
	mcmstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo/weave"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog"
)

func Storage(
	optsGetter generic.RESTOptionsGetter,
	storageOptions *mcmstorage.Options,
	inserter *weave.ClusterTopologyInserter,
	clientConfig klusterlet.ClientConfig,
	aggregatorGetter *aggregatorv1alpha1.InfoGetters) map[string]rest.Storage {
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

	if storageOptions.Mongo.MongoHost != "" {
		clusterstatustopoStorage := clusterstatustopostorage.NewTopologyRest(optsGetter, inserter)
		storage["clusterstatuses/topology"] = clusterstatustopoStorage
	} else {
		klog.Warning("Topology API Disabled. To enable, you must specify --mongo-host")
	}

	// cluster log storage
	if connection != nil {
		clusterlog := clusterstatusrest.NewLogRest(optsGetter, connection)
		clusterMonitor := clusterstatusrest.NewMonitorRest(optsGetter, connection)
		storage["clusterstatuses/log"] = clusterlog
		storage["clusterstatuses/monitor"] = clusterMonitor
	}

	// Aggregator storage
	for path, connection := range *aggregatorGetter {
		storagePath := fmt.Sprintf("clusterstatuses/%s", path)
		aggreator := restv1alpha1.NewAggregateRest(optsGetter, connection)
		storage[storagePath] = aggreator
	}

	// hcm join storage
	clusterjoinStorage, clusterjoinStatusStorage := clusterjoinstorage.NewREST(optsGetter)
	storage["clusterjoinrequests"] = clusterjoinStorage
	storage["clusterjoinrequests/status"] = clusterjoinStatusStorage

	leadervoteStorage, leadervoteStatusStorage := leadervotestorage.NewREST(optsGetter)
	storage["leadervotes"] = leadervoteStorage
	storage["leadervotes/status"] = leadervoteStatusStorage

	placementpolicyStorage, placementpolicyStatusStorage := placementpolicystorage.NewREST(optsGetter)
	storage["placementpolicies"] = placementpolicyStorage
	storage["placementpolicies/status"] = placementpolicyStatusStorage

	placementBindingStorage := placementbindingstorage.NewREST(optsGetter)
	storage["placementbindings"] = placementBindingStorage

	return storage
}
