// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package useridentity

import (
	"encoding/base64"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	rbaclisters "k8s.io/client-go/listers/rbac/v1"
)

const (
	// TODO: need to confirm that name of user/group annotations
	// UserIdentityAnnotation is identity annotation
	UserIdentityAnnotation = "open-cluster-management.io/user-identity"

	// UserGroupAnnotation is user group annotation
	UserGroupAnnotation = "open-cluster-management.io/user-group"
)

func MergeUserIdentityToAnnotations(
	userInfo authenticationv1.UserInfo,
	annotations map[string]string,
	namespace string,
	listers rbaclisters.RoleBindingLister,
) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	user := userInfo.Username

	filteredGroups := []string{}
	for _, group := range userInfo.Groups {
		groupArray := strings.Split(group, ":")

		// add group not created by iam
		if len(groupArray) != 3 {
			filteredGroups = append(filteredGroups, group)
			continue
		}

		// add group not from icp
		if groupArray[0] != "icp" {
			filteredGroups = append(filteredGroups, group)
			continue
		}

		// add iam default group
		if groupArray[1] == "default" {
			filteredGroups = append(filteredGroups, group)
			continue
		}

		_, err := listers.RoleBindings(namespace).Get(group)
		if err == nil {
			filteredGroups = append(filteredGroups, group)
		}
	}

	group := strings.Join(filteredGroups, ",")

	annotations[UserIdentityAnnotation] = base64.StdEncoding.EncodeToString([]byte(user))
	annotations[UserGroupAnnotation] = base64.StdEncoding.EncodeToString([]byte(group))
	return annotations
}
