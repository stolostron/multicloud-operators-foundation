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

	for clusterSet, clusters := range inputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		assert.True(t, clusterSetMapper.GetClustersOfClusterSet(clusterSet).Equal(clusters))
	}

	allClusterSetToClusters := clusterSetMapper.GetAllClusterSetToClusters()
	assert.Equal(t, len(allClusterSetToClusters), len(inputs))
	for clusterSet, clusters := range allClusterSetToClusters {
		assert.True(t, clusters.Equal(inputs[clusterSet]))
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

	for clusterSet, clusters := range updateInputs {
		clusterSetMapper.UpdateClusterSetByClusters(clusterSet, clusters)
		assert.True(t, clusterSetMapper.GetClustersOfClusterSet(clusterSet).Equal(clusters))
	}

	allClusterSetToClusters = clusterSetMapper.GetAllClusterSetToClusters()
	assert.Equal(t, len(allClusterSetToClusters), len(expectedAllClusterSetToClusters))
	for clusterSet, clusters := range allClusterSetToClusters {
		assert.True(t, expectedAllClusterSetToClusters[clusterSet].Equal(clusters))
	}

	// TestCase: Delete clusterSets
	deleteClusterSet := "clusterSet12"
	expectedAllClusterSetToClusters = map[string]sets.String{
		"clusterSet1": {"cluster11": {}, "cluster12": {}, "cluster13": {}},
		"clusterSet2": {"cluster11": {}, "cluster22": {}},
		"clusterSet3": {"cluster11": {}, "cluster12": {}, "cluster13": {}},
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

}

func initClustersetmap(m map[string]sets.String) *ClusterSetMapper {
	var clusterSetMapper = NewClusterSetMapper()
	for clusterset, cluster := range m {
		clusterSetMapper.UpdateClusterSetByClusters(clusterset, cluster)
	}
	return clusterSetMapper
}

func TestClusterSetMapper_DeleteClusterInClusterSet(t *testing.T) {
	// Delete cluster in clusterset
	initMap := map[string]sets.String{
		"clusterSet4": {"cluster11": {}},
		"clusterSet1": {"cluster12": {}, "cluster13": {}},
		"clusterSet3": {"cluster12": {}, "cluster22": {}},
	}
	clusterSetMapper := initClustersetmap(initMap)
	expectClustermap := map[string]sets.String{
		"clusterSet1": {"cluster12": {}, "cluster13": {}},
		"clusterSet3": {"cluster12": {}},
	}
	clusterSetMapper.DeleteClusterInClusterSet("cluster11")
	clusterSetMapper.DeleteClusterInClusterSet("cluster22")
	assert.Equal(t, len(expectClustermap), len(initMap)-1)
}

func TestClusterSetMapper_UpdateClusterInClusterSet(t *testing.T) {
	initMap := map[string]sets.String{
		"clusterSet2": {"cluster11": {}},
		"clusterSet1": {"cluster12": {}, "cluster13": {}},
		"clusterSet3": {"cluster12": {}, "cluster22": {}},
	}
	clusterSetMapper := initClustersetmap(initMap)
	expectClustermap := map[string]sets.String{
		"clusterSet1": {"cluster12": {}, "cluster13": {}},
		"clusterSet3": {"cluster12": {}, "cluster11": {}},
		"clusterSet4": {"cluster22": {}},
	}
	clusterSetMapper.UpdateClusterInClusterSet("cluster11", "clusterSet3")
	clusterSetMapper.UpdateClusterInClusterSet("cluster22", "clusterSet4")
	assert.Equal(t, len(expectClustermap), len(initMap))

}
