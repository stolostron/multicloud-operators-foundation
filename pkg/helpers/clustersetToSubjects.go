package helpers

import (
	"sync"

	rbacv1 "k8s.io/api/rbac/v1"
)

type ClustersetSubjectsMapper struct {
	mutex sync.RWMutex
	// mapping: ClusterSet - Subjects
	clustersetToSubjects map[string][]rbacv1.Subject
}

func NewClustersetSubjectsMapper() *ClustersetSubjectsMapper {
	return &ClustersetSubjectsMapper{
		clustersetToSubjects: make(map[string][]rbacv1.Subject),
	}
}

func (c *ClustersetSubjectsMapper) SetMap(m map[string][]rbacv1.Subject) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clustersetToSubjects = m
}

func (c *ClustersetSubjectsMapper) GetMap() map[string][]rbacv1.Subject {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.clustersetToSubjects
}

func (c *ClustersetSubjectsMapper) Get(k string) []rbacv1.Subject {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.clustersetToSubjects[k]
}
