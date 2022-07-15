package useridentity

import (
	"testing"

	v1 "k8s.io/api/authentication/v1"
)

//TODO: Adding more testing cases
func TestMergeUserIdentityToAnnotations(t *testing.T) {
	MergeUserIdentityToAnnotations(v1.UserInfo{Groups: []string{"a:b:c", "d:e"}}, nil, "", nil)
}
