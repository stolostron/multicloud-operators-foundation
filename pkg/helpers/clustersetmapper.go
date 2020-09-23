package helpers

import (
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
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

//DeleteClusterInClusterSet will delete cluster in all clusterset mapping
func (c *ClusterSetMapper) DeleteClusterInClusterSet(clusterName string) {
	if clusterName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for clusterset, clusters := range c.clusterSetToClusters {
		if !clusters.Has(clusterName) {
			continue
		}
		clusters.Delete(clusterName)
		if len(clusters) == 0 {
			delete(c.clusterSetToClusters, clusterset)
		}
	}

	return
}

//UpdateClusterInClusterSet updates clusterset to cluster mapping.
//If a the clusterset of a cluster is changed, this func remove cluster from the previous mapping and add in new one.
func (c *ClusterSetMapper) UpdateClusterInClusterSet(clusterName, clusterSetName string) {
	if clusterName == "" || clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.clusterSetToClusters[clusterSetName]; !ok {
		cluster := sets.NewString(clusterName)
		c.clusterSetToClusters[clusterSetName] = cluster
	} else {
		c.clusterSetToClusters[clusterSetName].Insert(clusterName)
	}

	for clusterset, clusters := range c.clusterSetToClusters {
		if clusterSetName == clusterset {
			continue
		}
		if !clusters.Has(clusterName) {
			continue
		}
		clusters.Delete(clusterName)
		if len(clusters) == 0 {
			delete(c.clusterSetToClusters, clusterset)
		}
	}

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
