package utils

import (
	"github.com/openshift/library-go/pkg/authorization/authorizationutil"
	"github.com/stolostron/multicloud-operators-foundation/pkg/cache"
	"github.com/stolostron/multicloud-operators-foundation/pkg/helpers"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
)

const (
	ClusterSetRole              string = "cluster.open-cluster-management.io/role"
	ClustersetRoleFinalizerName string = "cluster.open-cluster-management.io/managedclusterset-clusterrole"
	managedclusterGroup         string = "cluster.open-cluster-management.io"
	hiveGroup                   string = "hive.openshift.io"
	managedClusterViewGroup     string = "clusterview.open-cluster-management.io"
	registerGroup               string = "register.open-cluster-management.io"
	DefaultSetName              string = "default"
	GlobalSetName               string = "global"
	GlobalSetNameSpace          string = "open-cluster-management-global-set"
	GlobalPlacementName         string = "global"
)

// These subjects are excluded for the following reasons:
//  1. System admin groups (system:cluster-admins, system:masters) already have cluster-level full permissions
//  2. System reader group (system:cluster-readers) already has cluster-level read permissions
//  3. System users (system:admin, system:kube-controller-manager) are Kubernetes system components
//     that should not receive permissions through business logic to avoid interfering with system operations
//  4. Avoiding redundant permissions and potential security risks
//  5. Reducing unnecessary RoleBinding creation for better performance
var (
	ignoreGroup = sets.New("system:cluster-admins", "system:masters", "system:cluster-readers")
	ignoreUser  = sets.New("system:admin", "system:kube-controller-manager")
)

var GlobalSet = &clusterv1beta2.ManagedClusterSet{
	ObjectMeta: metav1.ObjectMeta{
		Name: GlobalSetName,
	},
	Spec: clusterv1beta2.ManagedClusterSetSpec{
		ClusterSelector: clusterv1beta2.ManagedClusterSelector{
			SelectorType:  clusterv1beta2.LabelSelector,
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
		subjects := filterSubjects(authorizationutil.BuildRBACSubjects(users, clustersetToGroups[set]))
		if len(subjects) > 0 {
			clustersetToSubjects[set] = subjects
		}
	}

	for set, groups := range clustersetToGroups {
		if _, ok := clustersetToUsers[set]; ok {
			continue
		}
		var nullUsers []string
		subjects := filterSubjects(authorizationutil.BuildRBACSubjects(nullUsers, groups))
		if len(subjects) > 0 {
			clustersetToSubjects[set] = subjects
		}
	}

	return clustersetToSubjects
}

// filterSubjects filters subject with:
// 1. ignore all subject with specified user and group
// 2. ignore all subject with sa kind
// This works since we do not need to create bindings for cluster-admin users
// and we do not need to create bindings for any service account. The feature
// is only used to control the user access automatically.
// TODO consider an explicit sa list rather than wildcard
func filterSubjects(subjects []rbacv1.Subject) []rbacv1.Subject {
	output := []rbacv1.Subject{}
	for _, subject := range subjects {
		switch subject.Kind {
		case rbacv1.GroupKind:
			if ignoreGroup.Has(subject.Name) {
				continue
			}
		case rbacv1.UserKind:
			if ignoreUser.Has(subject.Name) {
				continue
			}
		case rbacv1.ServiceAccountKind:
			continue
		}
		output = append(output, subject)
	}

	return output
}

// BuildAdminRole builds the admin clusterrole for the clusterset.
// The users with this clusterrole has admin permission(get/update/join/bind...) for the clusterset.
func BuildAdminRole(clustersetName string) *rbacv1.ClusterRole {
	adminroleName := utils.GenerateClustersetClusterroleName(clustersetName, "admin")
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: adminroleName,
			Labels: map[string]string{
				clusterv1beta2.ClusterSetLabel: clustersetName,
				ClusterSetRole:                 "admin",
			},
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("get", "update").
				Groups(managedclusterGroup).
				Resources("managedclustersets").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclustersets/join").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclustersets/bind").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclusters").
				RuleOrDie(),
			//TODO
			// We will restrict the update permission only for authenticated clusterset in another pr
			helpers.NewRule("update").
				Groups(registerGroup).
				Resources("managedclusters/accept").
				RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclustersets").
				RuleOrDie(),
		},
	}
}

// BuildViewRole builds the view clusterrole for the clusterset.
// The users with this clusterrole has view permission(get) for the clusterset.
func BuildViewRole(clustersetName string) *rbacv1.ClusterRole {
	viewroleName := utils.GenerateClustersetClusterroleName(clustersetName, "view")
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: viewroleName,
			Labels: map[string]string{
				clusterv1beta2.ClusterSetLabel: clustersetName,
				ClusterSetRole:                 "view",
			},
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("get").
				Groups(managedclusterGroup).
				Resources("managedclustersets").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclustersets").
				RuleOrDie(),
		},
	}
}

// BuildBindRole builds the bind clusterrole for the clusterset.
// The users with this clusterrole has bind and view permission for the clusterset.
func BuildBindRole(clustersetName string) *rbacv1.ClusterRole {
	bindroleName := utils.GenerateClustersetClusterroleName(clustersetName, "bind")
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindroleName,
			Labels: map[string]string{
				clusterv1beta2.ClusterSetLabel: clustersetName,
				ClusterSetRole:                 "bind",
			},
		},
		Rules: []rbacv1.PolicyRule{
			helpers.NewRule("create").
				Groups(managedclusterGroup).
				Resources("managedclustersets/bind").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("get").
				Groups(managedclusterGroup).
				Resources("managedclustersets").
				Names(clustersetName).
				RuleOrDie(),
			helpers.NewRule("get", "list", "watch").
				Groups(managedClusterViewGroup).
				Resources("managedclustersets").
				RuleOrDie(),
		},
	}
}
