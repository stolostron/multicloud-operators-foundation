package helpers

import (
	utils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/equals"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClusterSetMapper(t *testing.T) {
	var clusterSetMapper = NewClusterSetMapper()

	// TestCase: update clusterSets
	inputs := map[string][]string{
		"clusterSet1":  {"cluster11", "cluster12", "cluster13"},
		"clusterSet2":  {"cluster21", "cluster22", "cluster23"},
		"clusterSet12": {"cluster11", "cluster12", "cluster13", "cluster21", "cluster22", "cluster23"},
	}
	expectedAllCluster2ClusterSets := map[string][]string{
		"cluster11": {"clusterSet1", "clusterSet12"},
		"cluster12": {"clusterSet1", "clusterSet12"},
		"cluster13": {"clusterSet1", "clusterSet12"},
		"cluster21": {"clusterSet2", "clusterSet12"},
		"cluster22": {"clusterSet2", "clusterSet12"},
		"cluster23": {"clusterSet2", "clusterSet12"},
	}

	for clusterSet, clusters := range inputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		clustersOfClusterSet := clusterSetMapper.GetClustersOfClusterSet(clusterSet)
		assert.Equal(t, clustersOfClusterSet, clusters)
	}

	allClusterSet2Clusters := clusterSetMapper.GetAllClusterSet2Clusters()
	assert.Equal(t, len(allClusterSet2Clusters), len(inputs))
	for clusterSet, clusters := range allClusterSet2Clusters {
		assert.Equal(t, clusters, inputs[clusterSet])
	}

	for cluster, clusterSets := range expectedAllCluster2ClusterSets {
		assert.True(t, utils.EqualStringSlice(clusterSets, clusterSetMapper.GetClusterSetsOfCluster(cluster)))
	}

	allCluster2ClusterSets := clusterSetMapper.GetAllCluster2ClusterSets()
	assert.Equal(t, len(allCluster2ClusterSets), len(expectedAllCluster2ClusterSets))

	for cluster, clusterSets := range allCluster2ClusterSets {
		assert.True(t, utils.EqualStringSlice(clusterSets, expectedAllCluster2ClusterSets[cluster]))
	}

	// TestCase: update existed clusterSets
	updateInputs := map[string][]string{
		"clusterSet3": {"cluster11", "cluster12", "cluster13"},
		"clusterSet2": {"cluster11", "cluster22"},
	}
	expectedAllClusterSet2Clusters := map[string][]string{
		"clusterSet1":  {"cluster11", "cluster12", "cluster13"},
		"clusterSet12": {"cluster11", "cluster12", "cluster13", "cluster21", "cluster22", "cluster23"},
		"clusterSet3":  {"cluster11", "cluster12", "cluster13"},
		"clusterSet2":  {"cluster11", "cluster22"},
	}
	expectedAllCluster2ClusterSets = map[string][]string{
		"cluster11": {"clusterSet1", "clusterSet2", "clusterSet3", "clusterSet12"},
		"cluster12": {"clusterSet1", "clusterSet12", "clusterSet3"},
		"cluster13": {"clusterSet1", "clusterSet12", "clusterSet3"},
		"cluster21": {"clusterSet12"},
		"cluster22": {"clusterSet2", "clusterSet12"},
		"cluster23": {"clusterSet12"},
	}

	for clusterSet, clusters := range updateInputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		clustersOfClusterSet := clusterSetMapper.GetClustersOfClusterSet(clusterSet)
		assert.Equal(t, clustersOfClusterSet, clusters)
	}

	allClusterSet2Clusters = clusterSetMapper.GetAllClusterSet2Clusters()
	assert.Equal(t, len(allClusterSet2Clusters), len(expectedAllClusterSet2Clusters))
	for clusterSet, clusters := range allClusterSet2Clusters {
		assert.True(t, utils.EqualStringSlice(clusters, expectedAllClusterSet2Clusters[clusterSet]))
	}

	for cluster, clusterSets := range expectedAllCluster2ClusterSets {
		assert.True(t, utils.EqualStringSlice(clusterSets, clusterSetMapper.GetClusterSetsOfCluster(cluster)))
	}

	allCluster2ClusterSets = clusterSetMapper.GetAllCluster2ClusterSets()
	assert.Equal(t, len(allCluster2ClusterSets), len(expectedAllCluster2ClusterSets))

	for cluster, clusterSets := range allCluster2ClusterSets {
		assert.True(t, utils.EqualStringSlice(clusterSets, expectedAllCluster2ClusterSets[cluster]))
	}

	// TestCase: Delete clusterSets
	deleteClusterSet := "clusterSet12"
	expectedAllClusterSet2Clusters = map[string][]string{
		"clusterSet1": {"cluster11", "cluster12", "cluster13"},
		"clusterSet2": {"cluster11", "cluster22"},
		"clusterSet3": {"cluster11", "cluster12", "cluster13"},
	}
	expectedAllCluster2ClusterSets = map[string][]string{
		"cluster11": {"clusterSet1", "clusterSet2", "clusterSet3"},
		"cluster12": {"clusterSet1", "clusterSet3"},
		"cluster13": {"clusterSet1", "clusterSet3"},
		"cluster22": {"clusterSet2"},
	}

	clusterSetMapper.DeleteClusterSet(deleteClusterSet)
	for clusterSet, clusters := range updateInputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		clustersOfClusterSet := clusterSetMapper.GetClustersOfClusterSet(clusterSet)
		assert.Equal(t, clustersOfClusterSet, clusters)
	}

	allClusterSet2Clusters = clusterSetMapper.GetAllClusterSet2Clusters()
	assert.Equal(t, len(allClusterSet2Clusters), len(expectedAllClusterSet2Clusters))
	for clusterSet, clusters := range allClusterSet2Clusters {
		assert.True(t, utils.EqualStringSlice(clusters, expectedAllClusterSet2Clusters[clusterSet]))
	}

	for cluster, clusterSets := range expectedAllCluster2ClusterSets {
		assert.True(t, utils.EqualStringSlice(clusterSets, clusterSetMapper.GetClusterSetsOfCluster(cluster)))
	}

	allCluster2ClusterSets = clusterSetMapper.GetAllCluster2ClusterSets()
	assert.Equal(t, len(allCluster2ClusterSets), len(expectedAllCluster2ClusterSets))

	for cluster, clusterSets := range allCluster2ClusterSets {
		assert.True(t, utils.EqualStringSlice(clusterSets, expectedAllCluster2ClusterSets[cluster]))
	}
}
