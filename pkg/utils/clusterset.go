package utils

import (
	"fmt"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/cache"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/helpers"
	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	ClusterSetLabel string = "clusterset.cluster.open-cluster-management.io"
)

// GenerateObjectSubjectMap generate the map which key is object and value is subjects, which means these users/groups in subjects has permission for this object.
func GenerateObjectSubjectMap(clustersetToObjects *helpers.ClusterSetMapper, clustersetToSubject map[string][]rbacv1.Subject) map[string][]rbacv1.Subject {
	var objectToSubject = make(map[string][]rbacv1.Subject)

	for clusterset, subjects := range clustersetToSubject {
		if clusterset == "*" {
			continue
		}
		objects := clustersetToObjects.GetObjectsOfClusterSet(clusterset)
		for _, object := range objects.List() {
			objectToSubject[object] = Mergesubjects(objectToSubject[object], subjects)
		}
	}
	if len(clustersetToSubject["*"]) == 0 {
		return objectToSubject
	}
	//if clusterset is "*", should map this subjects to all namespace
	allClustersetToObjects := clustersetToObjects.GetAllClusterSetToObjects()
	for _, objs := range allClustersetToObjects {
		subjects := clustersetToSubject["*"]
		for _, obj := range objs.List() {
			objectToSubject[obj] = Mergesubjects(objectToSubject[obj], subjects)
		}
	}
	return objectToSubject
}

func GenerateClustersetSubjects(cache *cache.AuthCache) map[string][]rbacv1.Subject {
	clustersetToSubjects := make(map[string][]rbacv1.Subject)

	clustersetToUsers := make(map[string][]string)
	clustersetToGroups := make(map[string][]string)

	subjectUserRecords := cache.GetUserSubjectRecord()
	for _, subjectRecord := range subjectUserRecords {
		for _, set := range subjectRecord.Names.List() {
			clustersetToUsers[set] = append(clustersetToUsers[set], subjectRecord.Subject)
		}
	}

	subjectGroupRecords := cache.GetGroupSubjectRecord()
	for _, subjectRecord := range subjectGroupRecords {
		for _, set := range subjectRecord.Names.List() {
			clustersetToGroups[set] = append(clustersetToGroups[set], subjectRecord.Subject)
		}
	}

	for set, users := range clustersetToUsers {
		subjects := authorizationutil.BuildRBACSubjects(users, clustersetToGroups[set])
		clustersetToSubjects[set] = subjects
	}

	for set, groups := range clustersetToGroups {
		if _, ok := clustersetToUsers[set]; ok {
			continue
		}
		var nullUsers []string
		subjects := authorizationutil.BuildRBACSubjects(nullUsers, groups)
		clustersetToSubjects[set] = subjects
	}

	return clustersetToSubjects
}

// getResourceNamespaceName input should be "<ResourceType>/<Namespace>/<Name>" and return (resourceType, Namespace, Name)
func getResourceNamespaceName(namespacedName string) (string, string, string) {
	splitNamespacedName := strings.Split(namespacedName, "/")
	if len(splitNamespacedName) != 3 {
		return "", "", ""
	}
	return splitNamespacedName[0], splitNamespacedName[1], splitNamespacedName[2]
}

func ConvertToClusterSetNamespaceMap(clustersetToObjects *helpers.ClusterSetMapper) (*helpers.ClusterSetMapper, []error) {
	errs := []error{}

	setToObjMap := clustersetToObjects.GetAllClusterSetToObjects()
	if len(setToObjMap) == 0 {
		return clustersetToObjects, errs
	}
	returnMap := helpers.NewClusterSetMapper()
	for clusterset, objs := range setToObjMap {
		for obj := range objs {
			_, ns, _ := getResourceNamespaceName(obj)
			if ns == "" {
				errs = append(errs, fmt.Errorf("failed to get namespace from %s", obj))
			}
			returnMap.AddObjectInClusterSet(ns, clusterset)
		}
	}
	return returnMap, errs
}
