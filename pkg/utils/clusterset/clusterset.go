package utils

import (
	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

const (
	ClusterSetRole string = "cluster.open-cluster-management.io/role"

	DefaultSetName = "default"
	GlobalSetName  = "global"
)

var GlobalSet = &clusterv1beta1.ManagedClusterSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: GlobalSetName,
	},
	Spec: clusterv1beta1.ManagedClusterSetSpec{
		ClusterSelector: clusterv1beta1.ManagedClusterSelector{
			SelectorType:  clusterv1beta1.LabelSelector,
			LabelSelector: &metav1.LabelSelector{},
		},
	},
}

// GenerateObjectSubjectMap generate the map which key is object and value is subjects, which means these users/groups in subjects has permission for this object.
func GenerateObjectSubjectMap(clustersetToObjects *helpers.ClusterSetMapper, clustersetToSubject map[string][]rbacv1.Subject) map[string][]rbacv1.Subject {
	var objectToSubject = make(map[string][]rbacv1.Subject)

	for clusterset, subjects := range clustersetToSubject {
		if clusterset == "*" {
			continue
		}
		objects := clustersetToObjects.GetObjectsOfClusterSet(clusterset)
		for _, object := range objects.List() {
			objectToSubject[object] = utils.Mergesubjects(objectToSubject[object], subjects)
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
			objectToSubject[obj] = utils.Mergesubjects(objectToSubject[obj], subjects)
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
