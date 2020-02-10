// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1beta1

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator"
	klusterlet "github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/client"
	clusterjoinstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterjoinrequest/etcd"
	clusterstatusstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/etcd"
	clusterstatusrest "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/clusterstatus/rest"
	resourceviewstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/resourceview/etcd"
	workstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/work/etcd"
	workresultstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/work/mongo"
	worksetstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/registry/mcm/workset/etcd"
	mcmstorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/klog"
)

func Storage(
	optsGetter generic.RESTOptionsGetter,
	storageOptions *mcmstorage.Options,
	clientConfig klusterlet.ClientConfig,
	aggregatorGetter *aggregator.InfoGetters) map[string]rest.Storage {
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

	storage["clusterstatuses/aggregator"] = clusterstatusrest.NewAggregateRest(aggregatorGetter)

	// hcm join storage
	clusterjoinStorage, clusterjoinStatusStorage := clusterjoinstorage.NewREST(optsGetter)
	storage["clusterjoinrequests"] = clusterjoinStorage
	storage["clusterjoinrequests/status"] = clusterjoinStatusStorage

	return storage
}
