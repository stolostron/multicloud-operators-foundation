package helpers

import (
	"sync"

	rbacv1 "k8s.io/api/rbac/v1"
)

type ClustersetSubjectsMapper struct {
	mutex sync.RWMutex
	// mapping: ClusterSet - Subjects
	clusterset2Subjects map[string][]rbacv1.Subject
}

func NewClustersetSubjectsMapper() *ClustersetSubjectsMapper {
	return &ClustersetSubjectsMapper{
		clusterset2Subjects: make(map[string][]rbacv1.Subject),
	}
}

func (c *ClustersetSubjectsMapper) Set(key string, value []rbacv1.Subject) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clusterset2Subjects[key] = value
}

func (c *ClustersetSubjectsMapper) GetMap() map[string][]rbacv1.Subject {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.clusterset2Subjects
}
