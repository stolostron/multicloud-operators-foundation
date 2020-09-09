package helpers

import (
	"sync"
)

type ClusterSetMapper struct {
	mutex sync.RWMutex
	// mapping: ClusterSet - Clusters
	clusterSet2Clusters map[string][]string
	// mapping: Cluster - ClusterSets
	cluster2ClusterSets map[string][]string
}

func NewClusterSetMapper() *ClusterSetMapper {
	return &ClusterSetMapper{
		clusterSet2Clusters: make(map[string][]string),
		cluster2ClusterSets: make(map[string][]string),
	}
}

func (c *ClusterSetMapper) UpdateClusterSetByClusters(clusterSetName string, clusters []string) {
	if clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, oldCluster := range c.clusterSet2Clusters[clusterSetName] {
		c.deleteClusterSetOfCluster(oldCluster, clusterSetName)
	}

	c.clusterSet2Clusters[clusterSetName] = clusters
	for _, cluster := range clusters {
		c.cluster2ClusterSets[cluster] = append(c.cluster2ClusterSets[cluster], clusterSetName)
	}
	return
}

func (c *ClusterSetMapper) DeleteClusterSet(clusterSetName string) {
	if clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, oldCluster := range c.clusterSet2Clusters[clusterSetName] {
		c.deleteClusterSetOfCluster(oldCluster, clusterSetName)
	}
	delete(c.clusterSet2Clusters, clusterSetName)

	return
}

func (c *ClusterSetMapper) GetClustersOfClusterSet(clusterSetName string) []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.clusterSet2Clusters[clusterSetName]
}

func (c *ClusterSetMapper) GetAllClusterSet2Clusters() map[string][]string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.clusterSet2Clusters
}

func (c *ClusterSetMapper) GetClusterSetsOfCluster(clusterName string) []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.cluster2ClusterSets[clusterName]
}

func (c *ClusterSetMapper) GetAllCluster2ClusterSets() map[string][]string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.cluster2ClusterSets
}

func (c *ClusterSetMapper) deleteClusterSetOfCluster(clusterName, clusterSetName string) {
	for i := 0; i < len(c.cluster2ClusterSets[clusterName]); i++ {
		if c.cluster2ClusterSets[clusterName][i] == clusterSetName {
			c.cluster2ClusterSets[clusterName] = append(c.cluster2ClusterSets[clusterName][:i], c.cluster2ClusterSets[clusterName][i+1:]...)
			break
		}
	}
	if len(c.cluster2ClusterSets[clusterName]) == 0 {
		delete(c.cluster2ClusterSets, clusterName)
	}
	return
}
