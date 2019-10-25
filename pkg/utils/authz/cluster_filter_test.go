// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package authz

import (
	"encoding/base64"
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	v1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func newCluster(name, namespace string, status clusterv1alpha1.ClusterConditionType, labels map[string]string) *clusterv1alpha1.Cluster {
	return &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: namespace,
		},
		Status: clusterv1alpha1.ClusterStatus{
			Conditions: []clusterv1alpha1.ClusterCondition{
				clusterv1alpha1.ClusterCondition{
					Type: status,
				},
			},
		},
	}
}

func newClusterList(numNodes int, label map[string]string) (clusters []*clusterv1alpha1.Cluster) {
	for i := 0; i < numNodes; i++ {
		clusters = append(clusters, newCluster(fmt.Sprintf("cluster-%d", i), fmt.Sprintf("node-%d", i), clusterv1alpha1.ClusterOK, label))
	}

	return clusters
}

func newView(name, namespace string) *v1alpha1.ResourceView {
	return &v1alpha1.ResourceView{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ResourceViewSpec{},
	}
}

func TestFilterClusterByUserIdentity(t *testing.T) {
	clusters := newClusterList(4, map[string]string{})
	view := newView("test", "test")
	clientset := kubernetes.NewForConfigOrDie(&restclient.Config{Host: "", ContentConfig: restclient.ContentConfig{GroupVersion: &schema.GroupVersion{Group: "", Version: "v1"}}})

	view1 := newView("test1", "test1")
	view1.Annotations = map[string]string{
		v1alpha1.UserIdentityAnnotation: base64.StdEncoding.EncodeToString([]byte("user1")),
		v1alpha1.UserGroupAnnotation:    base64.StdEncoding.EncodeToString([]byte("group1")),
	}

	type args struct {
		view       *v1alpha1.ResourceView
		kubeclient kubernetes.Interface
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{
			"case1:",
			args{
				view,
				nil,
			},
			4,
		},
		{
			"case2:",
			args{
				view,
				clientset,
			},
			4,
		},
		{
			"case3:",
			args{
				view1,
				clientset,
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterClusterByUserIdentity(tt.args.view, clusters, tt.args.kubeclient, "works", "create")
			if len(filtered) != tt.want {
				t.Errorf("updateWork() = %v, want %v", len(filtered), tt.want)
			}
		})
	}

}

func TestExtractUserAndGroup(t *testing.T) {
	type args struct {
		annotation map[string]string
	}

	type want struct {
		user   string
		groups []string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"case1:",
			args{
				annotation: map[string]string{
					v1alpha1.UserIdentityAnnotation: base64.StdEncoding.EncodeToString([]byte("user1")),
					v1alpha1.UserGroupAnnotation:    base64.StdEncoding.EncodeToString([]byte("group1,group2")),
				},
			},
			want{
				user:   "user1",
				groups: []string{"group1", "group2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, groups := extractUserAndGroup(tt.args.annotation)
			if user != tt.want.user {
				t.Errorf("extractUserAndGroup() = %v, want %v", user, tt.want.user)
			}

			if len(groups) != len(tt.want.groups) {
				t.Errorf("extractUserAndGroup() = %v, want %v", groups, tt.want.groups)
			}
		})
	}
}
