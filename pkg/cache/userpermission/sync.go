package userpermission

import (
	"crypto/sha256"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

// permissionProcessor processes permissions and populates the permission store
type permissionProcessor interface {
	// sync adds permissions to the permission store
	sync(store *permissionStore) error
	// getResourceVersionHash calculates a hash of the resources this processor is interested in
	getResourceVersionHash() (string, error)
}

// synchronize runs a full synchronization of the cache
func (c *Cache) synchronize() {
	startTime := time.Now()

	// Calculate hash of all processors' resource versions before acquiring the lock
	newHash, err := c.calculateResourceVersionHash()
	if err != nil {
		klog.Errorf("Failed to calculate resource version hash: %v", err)
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	// Check if resources have changed
	if c.resourceVersionHash == newHash {
		klog.V(4).Infof("No changes detected in resources, skipping synchronization, took %v", time.Since(startTime))
		return
	}

	if c.resourceVersionHash == "" {
		klog.V(2).Info("Initial synchronization of UserPermissionCache")
	} else {
		klog.V(2).Infof(
			"Resource changes detected (hash changed from %s to %s), synchronizing cache",
			c.resourceVersionHash[:8], newHash[:8])
	}

	// Build permission store
	store := newPermissionStore()

	// Process permissions using all processors
	for _, processor := range c.processors {
		if err := processor.sync(store); err != nil {
			klog.Errorf("Failed to sync permissions: %v", err)
			return
		}
	}

	// Replace the cache stores with the new ones
	c.permissionStore = store

	// Update the resource version hash after successful synchronization
	c.resourceVersionHash = newHash

	klog.V(2).Infof("UserPermissionCache synchronized: %d users, %d groups, %d discoverable roles, took %v",
		len(store.userStore.List()), len(store.groupStore.List()), len(store.getDiscoverableRoles()),
		time.Since(startTime))
}

// calculateResourceVersionHash computes a hash from all processors
func (c *Cache) calculateResourceVersionHash() (string, error) {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("calculateResourceVersionHash took %v", time.Since(startTime))
	}()

	h := sha256.New()

	// Get hash from each processor and combine them
	for i, processor := range c.processors {
		processorHash, err := processor.getResourceVersionHash()
		if err != nil {
			return "", fmt.Errorf("failed to get resource version hash from processor %d: %w", i, err)
		}
		_, _ = h.Write([]byte(processorHash))
		_, _ = h.Write([]byte("\n"))
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
