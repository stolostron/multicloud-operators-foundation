// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package authz

import (
	"encoding/base64"
	"strings"

	v1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	rbacv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// FilterClusterByUserIdentity filters cluster by checking if user can act on on resources
func FilterClusterByUserIdentity(
	obj runtime.Object,
	clusters []*clusterv1alpha1.Cluster,
	kubeclient kubernetes.Interface,
	resource, verb string,
) []*clusterv1alpha1.Cluster {
	if kubeclient == nil {
		return clusters
	}

	accessor, err := meta.Accessor(obj)
	if err != nil {
		return clusters
	}
	annotations := accessor.GetAnnotations()
	if annotations == nil {
		return clusters
	}

	filteredClusters := []*clusterv1alpha1.Cluster{}
	for _, cluster := range clusters {
		user, groups := extractUserAndGroup(annotations)
		sar := &rbacv1.SubjectAccessReview{
			Spec: rbacv1.SubjectAccessReviewSpec{
				ResourceAttributes: &rbacv1.ResourceAttributes{
					Namespace: cluster.Namespace,
					Group:     v1alpha1.GroupName,
					Verb:      verb,
					Resource:  resource,
				},
				User:   user,
				Groups: groups,
			},
		}
		result, err := kubeclient.AuthorizationV1().SubjectAccessReviews().Create(sar)
		if err != nil {
			continue
		}
		if !result.Status.Allowed {
			continue
		}

		filteredClusters = append(filteredClusters, cluster)
	}

	return filteredClusters
}

func extractUserAndGroup(annotations map[string]string) (string, []string) {
	var user string
	var groups []string

	encodedUser, ok := annotations[v1alpha1.UserIdentityAnnotation]
	if ok {
		decodedUser, err := base64.StdEncoding.DecodeString(encodedUser)
		if err == nil {
			user = string(decodedUser)
		}
	}

	encodedGroups, ok := annotations[v1alpha1.UserGroupAnnotation]
	if ok {
		decodedGroup, err := base64.StdEncoding.DecodeString(encodedGroups)
		if err == nil {
			groups = strings.Split(string(decodedGroup), ",")
		}
	}

	return user, groups
}
