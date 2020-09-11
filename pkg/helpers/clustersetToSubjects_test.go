package helpers

import (
	"reflect"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestClustersetSubjectsMapper_Get(t *testing.T) {
	clusterSubjects := NewClustersetSubjectsMapper()
	var requiredMap = make(map[string][]rbacv1.Subject)
	requiedSubject := []rbacv1.Subject{{Kind: "R1", APIGroup: "G1", Name: "N1"}}
	requiredMap["c1"] = requiedSubject
	clusterSubjects.SetMap(requiredMap)
	returnSubject := clusterSubjects.Get("c1")
	if !reflect.DeepEqual(returnSubject, requiedSubject) {
		t.Errorf("Failed to get cluster subjects")
	}
	returnMap := clusterSubjects.GetMap()
	if !reflect.DeepEqual(returnMap, requiredMap) {
		t.Errorf("Failed to get cluster subjects")
	}

	//Do not exist
	requirdNil := clusterSubjects.Get("c2")
	if requirdNil != nil {
		t.Errorf("Failed to get cluster subjects")
	}
}
