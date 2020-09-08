package helpers

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"sync"
)

type ClusterSetMapper struct {
	mutex sync.RWMutex
	// mapping: ClusterSet - Clusters
	clusterSetToClusters map[string]sets.String
}

func NewClusterSetMapper() *ClusterSetMapper {
	return &ClusterSetMapper{
		clusterSetToClusters: make(map[string]sets.String),
	}
}

func (c *ClusterSetMapper) UpdateClusterSetByClusters(clusterSetName string, clusters sets.String) {
	if clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.clusterSetToClusters[clusterSetName] = clusters

	return
}

func (c *ClusterSetMapper) DeleteClusterSet(clusterSetName string) {
	if clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.clusterSetToClusters, clusterSetName)

	return
}

func (c *ClusterSetMapper) GetClustersOfClusterSet(clusterSetName string) sets.String {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.clusterSetToClusters[clusterSetName]
}

func (c *ClusterSetMapper) GetAllClusterSetToClusters() map[string]sets.String {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.clusterSetToClusters
}

func (c *ClusterSetMapper) GetClusterSetsOfCluster(clusterName string) sets.String {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	clusterSets := sets.NewString()
	for clusterSet, clusters := range c.clusterSetToClusters {
		if clusters.Has(clusterName) {
			clusterSets.Insert(clusterSet)
		}
	}
	return clusterSets
}

func (c *ClusterSetMapper) GetAllClusterToClusterSets() map[string]sets.String {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	clusterToClusterSets := make(map[string]sets.String)
	for clusterSet, clusters := range c.clusterSetToClusters {
		for _, cluster := range clusters.UnsortedList() {
			if _, ok := clusterToClusterSets[cluster]; !ok {
				clusterToClusterSets[cluster] = sets.NewString(clusterSet)
				continue
			}
			clusterToClusterSets[cluster].Insert(clusterSet)
		}
	}
	return clusterToClusterSets
}
