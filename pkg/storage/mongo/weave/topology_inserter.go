// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package weave

import (
	"fmt"
	"time"

	mcmv1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	mongostorage "github.com/open-cluster-management/multicloud-operators-foundation/pkg/storage/mongo"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// topologyData is the topology data to insert to mongo
type topologyData struct {
	cluster string
	data    string
}

// ClusterTopologyInserter is a process to insert topology data
type ClusterTopologyInserter struct {
	topologyQueue *cache.FIFO
	storage       *MongoStorage
	stopCh        <-chan struct{}
}

func topoDataKeyFunc(obj interface{}) (string, error) {
	data, ok := obj.(*topologyData)
	if !ok {
		return "", fmt.Errorf("data type is not correct")
	}

	return data.cluster, nil
}

// NewClusterTopologyInserter returns a new data inserter
func NewClusterTopologyInserter(mongoOptions *mongostorage.Options, stopCh <-chan struct{}) (*ClusterTopologyInserter, error) {
	newCollection, err := NewMongoStorage(mongoOptions, mongoOptions.MongoCollection)
	if err != nil {
		return nil, err
	}

	return &ClusterTopologyInserter{
		topologyQueue: cache.NewFIFO(topoDataKeyFunc),
		storage:       newCollection,
		stopCh:        stopCh,
	}, nil
}

// InsertData insert a data in to queue
func (ci *ClusterTopologyInserter) InsertData(topoData *mcmv1alpha1.ClusterStatusTopology) {
	data := &topologyData{
		cluster: topoData.Name,
		data:    topoData.Data,
	}

	if err := ci.topologyQueue.Add(data); err != nil {
		klog.Errorf("topologyQueue add fail: %v", err)
	}
}

// Run start the inserter
func (ci *ClusterTopologyInserter) Run() {
	defer utilruntime.HandleCrash()
	defer ci.topologyQueue.Close()

	go wait.Until(ci.runWorker, time.Second, ci.stopCh)
	<-ci.stopCh
	klog.Info("Shutting inserter")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (ci *ClusterTopologyInserter) runWorker() {
	for ci.processEvent() {
	}
}

func (ci *ClusterTopologyInserter) processEvent() bool {
	obj := cache.Pop(ci.topologyQueue)
	data := obj.(*topologyData)
	err := ci.storage.ExtractTransformLoadRemove(data.data)
	if err != nil {
		klog.Errorf("failed to insert data to mongo: %v", err)
		return false
	}
	return true
}
