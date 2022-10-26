package helpers

import (
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
)

type ClusterSetMapper struct {
	mutex sync.RWMutex
	// mapping: ClusterSet - Objects
	clusterSetToObjects map[string]sets.String
}

func NewClusterSetMapper() *ClusterSetMapper {
	return &ClusterSetMapper{
		clusterSetToObjects: make(map[string]sets.String),
	}
}

func (c *ClusterSetMapper) UpdateClusterSetByObjects(clusterSetName string, objects sets.String) {
	if clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if objects.Len() == 0 {
		delete(c.clusterSetToObjects, clusterSetName)
		return
	}
	c.clusterSetToObjects[clusterSetName] = objects
}

func (c *ClusterSetMapper) DeleteClusterSet(clusterSetName string) {
	if clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.clusterSetToObjects, clusterSetName)

	return
}

func (c *ClusterSetMapper) CopyClusterSetMapper(requiredMapper *ClusterSetMapper) {
	for set := range c.GetAllClusterSetToObjects() {
		c.DeleteClusterSet(set)
	}
	for requiredSet, requiredObjs := range requiredMapper.GetAllClusterSetToObjects() {
		c.UpdateClusterSetByObjects(requiredSet, requiredObjs)
	}
}

// DeleteObjectInClusterSet will delete cluster in all clusterset mapping
func (c *ClusterSetMapper) DeleteObjectInClusterSet(objectName string) {
	if objectName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for clusterset, objects := range c.clusterSetToObjects {
		if !objects.Has(objectName) {
			continue
		}
		objects.Delete(objectName)
		if len(objects) == 0 {
			delete(c.clusterSetToObjects, clusterset)
		}
	}

	return
}

// AddObjectInClusterSet add object to clusterset mapping. it only add the object to current clusterset,
// and will not delete the object in other clusterset.
func (c *ClusterSetMapper) AddObjectInClusterSet(objectName, clusterSetName string) {
	if objectName == "" || clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.clusterSetToObjects[clusterSetName]; !ok {
		object := sets.NewString(objectName)
		c.clusterSetToObjects[clusterSetName] = object
	} else {
		c.clusterSetToObjects[clusterSetName].Insert(objectName)
	}

	return
}

// UpdateObjectInClusterSet updates clusterset to cluster mapping.
// If a the clusterset of a object is changed, this func remove object from the previous mapping and add in new one.
func (c *ClusterSetMapper) UpdateObjectInClusterSet(objectName, clusterSetName string) {
	if objectName == "" || clusterSetName == "" {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.clusterSetToObjects[clusterSetName]; !ok {
		cluster := sets.NewString(objectName)
		c.clusterSetToObjects[clusterSetName] = cluster
	} else {
		c.clusterSetToObjects[clusterSetName].Insert(objectName)
	}

	for clusterset, objects := range c.clusterSetToObjects {
		if clusterSetName == clusterset {
			continue
		}
		if !objects.Has(objectName) {
			continue
		}
		objects.Delete(objectName)
		if len(objects) == 0 {
			delete(c.clusterSetToObjects, clusterset)
		}
	}

	return
}

func (c *ClusterSetMapper) GetObjectsOfClusterSet(clusterSetName string) sets.String {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.clusterSetToObjects[clusterSetName]
}

func (c *ClusterSetMapper) GetAllClusterSetToObjects() map[string]sets.String {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.clusterSetToObjects
}

// UnionObjectsInClusterSet merge the objects in current ClusterSetMapper and newClustersetToObjects when clusterset is same.
func (c *ClusterSetMapper) UnionObjectsInClusterSet(newClustersetToObjects *ClusterSetMapper) *ClusterSetMapper {
	curSetToObjMap := c.GetAllClusterSetToObjects()
	if len(curSetToObjMap) == 0 {
		return newClustersetToObjects
	}
	newSetToObjMap := newClustersetToObjects.GetAllClusterSetToObjects()
	if len(newSetToObjMap) == 0 {
		return c
	}

	unionSetToObjMapper := NewClusterSetMapper()
	for set, objs := range curSetToObjMap {
		if _, ok := newSetToObjMap[set]; ok {
			unionObjs := objs.Union(newSetToObjMap[set])
			unionSetToObjMapper.UpdateClusterSetByObjects(set, unionObjs)
			continue
		}
		unionSetToObjMapper.UpdateClusterSetByObjects(set, objs)
	}

	for newSet, newObjs := range newSetToObjMap {
		if _, ok := curSetToObjMap[newSet]; ok {
			continue
		}
		unionSetToObjMapper.UpdateClusterSetByObjects(newSet, newObjs)
	}
	return unionSetToObjMapper
}

func (c *ClusterSetMapper) GetObjectClusterset(objectName string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for set, objs := range c.clusterSetToObjects {
		if objs.Has(objectName) {
			return set
		}
	}
	return ""
}
