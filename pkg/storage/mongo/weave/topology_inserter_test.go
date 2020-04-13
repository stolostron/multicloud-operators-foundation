package weave

import (
	"testing"

	mcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	mongostorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo"
	"k8s.io/client-go/tools/cache"
)

func NewFakeClusterTopologyInserter(stopCh <-chan struct{}) *ClusterTopologyInserter {
	newCollection := &MongoStorage{}

	return &ClusterTopologyInserter{
		topologyQueue: cache.NewFIFO(topoDataKeyFunc),
		storage:       newCollection,
		stopCh:        stopCh,
	}
}

func Test_ClusterTopologyInserter(t *testing.T) {
	options := &mongostorage.Options{}
	stopCh := make(chan struct{})
	inserter, _ := NewClusterTopologyInserter(options, stopCh)
	if inserter == nil {
		inserter = NewFakeClusterTopologyInserter(stopCh)
	}
	topologyData := &mcmv1alpha1.ClusterStatusTopology{
		Name: "topology",
		Data: "data",
	}

	inserter.InsertData(topologyData)
	inserter.runWorker()
}
