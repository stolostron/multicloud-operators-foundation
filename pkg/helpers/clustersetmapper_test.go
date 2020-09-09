package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestClusterSetMapper(t *testing.T) {
	var clusterSetMapper = NewClusterSetMapper()

	// TestCase: update clusterSets
	inputs := map[string]sets.String{
		"clusterSet1":  {"cluster11": {}, "cluster12": {}, "cluster13": {}},
		"clusterSet2":  {"cluster21": {}, "cluster22": {}, "cluster23": {}},
		"clusterSet12": {"cluster11": {}, "cluster12": {}, "cluster13": {}, "cluster21": {}, "cluster22": {}, "cluster23": {}},
	}
	expectedAllClusterToClusterSets := map[string]sets.String{
		"cluster11": {"clusterSet1": {}, "clusterSet12": {}},
		"cluster12": {"clusterSet1": {}, "clusterSet12": {}},
		"cluster13": {"clusterSet1": {}, "clusterSet12": {}},
		"cluster21": {"clusterSet2": {}, "clusterSet12": {}},
		"cluster22": {"clusterSet2": {}, "clusterSet12": {}},
		"cluster23": {"clusterSet2": {}, "clusterSet12": {}},
	}

	for clusterSet, clusters := range inputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		assert.True(t, clusterSetMapper.GetClustersOfClusterSet(clusterSet).Equal(clusters))
	}

	allClusterSetToClusters := clusterSetMapper.GetAllClusterSetToClusters()
	assert.Equal(t, len(allClusterSetToClusters), len(inputs))
	for clusterSet, clusters := range allClusterSetToClusters {
		assert.True(t, clusters.Equal(inputs[clusterSet]))
	}

	for cluster, clusterSets := range expectedAllClusterToClusterSets {
		assert.True(t, clusterSetMapper.GetClusterSetsOfCluster(cluster).Equal(clusterSets))
	}

	allClusterToClusterSets := clusterSetMapper.GetAllClusterToClusterSets()
	assert.Equal(t, len(allClusterToClusterSets), len(expectedAllClusterToClusterSets))
	for cluster, clusterSets := range allClusterToClusterSets {
		assert.True(t, expectedAllClusterToClusterSets[cluster].Equal(clusterSets))
	}

	// TestCase: update existed clusterSets
	updateInputs := map[string]sets.String{
		"clusterSet3": {"cluster11": {}, "cluster12": {}, "cluster13": {}},
		"clusterSet2": {"cluster11": {}, "cluster22": {}},
	}
	expectedAllClusterSetToClusters := map[string]sets.String{
		"clusterSet1":  {"cluster11": {}, "cluster12": {}, "cluster13": {}},
		"clusterSet12": {"cluster11": {}, "cluster12": {}, "cluster13": {}, "cluster21": {}, "cluster22": {}, "cluster23": {}},
		"clusterSet3":  {"cluster11": {}, "cluster12": {}, "cluster13": {}},
		"clusterSet2":  {"cluster11": {}, "cluster22": {}},
	}
	expectedAllClusterToClusterSets = map[string]sets.String{
		"cluster11": {"clusterSet1": {}, "clusterSet2": {}, "clusterSet3": {}, "clusterSet12": {}},
		"cluster12": {"clusterSet1": {}, "clusterSet12": {}, "clusterSet3": {}},
		"cluster13": {"clusterSet1": {}, "clusterSet12": {}, "clusterSet3": {}},
		"cluster21": {"clusterSet12": {}},
		"cluster22": {"clusterSet2": {}, "clusterSet12": {}},
		"cluster23": {"clusterSet12": {}},
	}

	for clusterSet, clusters := range updateInputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		assert.True(t, clusterSetMapper.GetClustersOfClusterSet(clusterSet).Equal(clusters))
	}

	allClusterSetToClusters = clusterSetMapper.GetAllClusterSetToClusters()
	assert.Equal(t, len(allClusterSetToClusters), len(expectedAllClusterSetToClusters))
	for clusterSet, clusters := range allClusterSetToClusters {
		assert.True(t, expectedAllClusterSetToClusters[clusterSet].Equal(clusters))
	}

	for cluster, clusterSets := range expectedAllClusterToClusterSets {
		assert.True(t, clusterSetMapper.GetClusterSetsOfCluster(cluster).Equal(clusterSets))
	}

	allClusterToClusterSets = clusterSetMapper.GetAllClusterToClusterSets()
	assert.Equal(t, len(allClusterToClusterSets), len(expectedAllClusterToClusterSets))

	for cluster, clusterSets := range allClusterToClusterSets {
		assert.True(t, expectedAllClusterToClusterSets[cluster].Equal(clusterSets))
	}

	// TestCase: Delete clusterSets
	deleteClusterSet := "clusterSet12"
	expectedAllClusterSetToClusters = map[string]sets.String{
		"clusterSet1": {"cluster11": {}, "cluster12": {}, "cluster13": {}},
		"clusterSet2": {"cluster11": {}, "cluster22": {}},
		"clusterSet3": {"cluster11": {}, "cluster12": {}, "cluster13": {}},
	}
	expectedAllClusterToClusterSets = map[string]sets.String{
		"cluster11": {"clusterSet1": {}, "clusterSet2": {}, "clusterSet3": {}},
		"cluster12": {"clusterSet1": {}, "clusterSet3": {}},
		"cluster13": {"clusterSet1": {}, "clusterSet3": {}},
		"cluster22": {"clusterSet2": {}},
	}

	clusterSetMapper.DeleteClusterSet(deleteClusterSet)
	for clusterSet, clusters := range updateInputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		assert.True(t, clusterSetMapper.GetClustersOfClusterSet(clusterSet).Equal(clusters))
	}

	allClusterSetToClusters = clusterSetMapper.GetAllClusterSetToClusters()
	assert.Equal(t, len(allClusterSetToClusters), len(expectedAllClusterSetToClusters))
	for clusterSet, clusters := range allClusterSetToClusters {
		assert.True(t, expectedAllClusterSetToClusters[clusterSet].Equal(clusters))
	}

	for cluster, clusterSets := range expectedAllClusterToClusterSets {
		assert.True(t, clusterSetMapper.GetClusterSetsOfCluster(cluster).Equal(clusterSets))
	}

	allClusterToClusterSets = clusterSetMapper.GetAllClusterToClusterSets()
	assert.Equal(t, len(allClusterToClusterSets), len(expectedAllClusterToClusterSets))

	for cluster, clusterSets := range allClusterToClusterSets {
		assert.True(t, expectedAllClusterToClusterSets[cluster].Equal(clusterSets))
	}

	// Test cases: invalid inputs
	assert.Equal(t, clusterSetMapper.GetClustersOfClusterSet("abc").Len(), 0)
	assert.Equal(t, clusterSetMapper.GetClusterSetsOfCluster("abc").Len(), 0)

}
