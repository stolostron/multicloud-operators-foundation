package userpermission

import (
	"fmt"

	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

const (
	// Resource version format constants
	ClusterRoleVersionFormat = "cr:%s:%s"
	ClusterPermissionFormat  = "cp:%s:%s"
	ClusterRoleBindingFormat = "crb:%s:%s"
	RoleFormat               = "r:%s:%s"
	RoleBindingFormat        = "rb:%s/%s:%s"

	// API group constants
	ActionAPIGroup = "action.open-cluster-management.io"
	ViewAPIGroup   = "view.open-cluster-management.io"

	// Resource constants
	ManagedClusterActionsResource = "managedclusteractions"
	ManagedClusterViewsResource   = "managedclusterviews"
)

// UserPermissionRecord represents permissions for a single user or group
type UserPermissionRecord struct {
	Subject     string
	Permissions map[string][]clusterviewv1alpha1.ClusterBinding // map[ClusterRoleName][]ClusterBinding
}

// userPermissionRecordKeyFn is a key func for UserPermissionRecord objects
func userPermissionRecordKeyFn(obj interface{}) (string, error) {
	record, ok := obj.(*UserPermissionRecord)
	if !ok {
		return "", fmt.Errorf("expected UserPermissionRecord")
	}
	return record.Subject, nil
}
