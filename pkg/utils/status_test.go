// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestGetStatusFromCluster(t *testing.T) {
	var (
		//ServerAddressByClientCIDR1 = v1alpha1.ServerAddressByClientCIDR{ClientCIDR: "aaa", ServerAddress: "aaa"}
		//ServerAddressByClientCIDR2 = v1alpha1.ServerAddressByClientCIDR{ClientCIDR: "bbb", ServerAddress: "bbb"}
		//KubernetesAPIEndpoints1    = v1alpha1.KubernetesAPIEndpoints{ServerEndpoints: []v1alpha1.ServerAddressByClientCIDR{ServerAddressByClientCIDR1, ServerAddressByClientCIDR2}, CABundle: []byte("aaa")}
		//AuthInfo1                  = v1alpha1.AuthInfo{}
		//ClusterSpe1 = v1alpha1.ClusterSpec{KubernetesAPIEndpoints: KubernetesAPIEndpoints1, AuthInfo: AuthInfo1}

		ClusterConditionType1 = v1alpha1.ClusterConditionType("type1")
		ClusterConditionType2 = v1alpha1.ClusterConditionType("type2")
		Status1               = v1.ConditionStatus("ok")

		ClusterCondition1 = v1alpha1.ClusterCondition{Type: ClusterConditionType1, Status: Status1}
		ClusterCondition2 = v1alpha1.ClusterCondition{Type: ClusterConditionType2, Status: Status1}
		ClusterStatus1    = v1alpha1.ClusterStatus{Conditions: []v1alpha1.ClusterCondition{ClusterCondition1, ClusterCondition2}}

		cluster1 = v1alpha1.Cluster{Status: ClusterStatus1}
	)
	type args struct {
		cluster v1alpha1.Cluster
	}
	tests := []struct {
		name  string
		args  args
		want  v1alpha1.ClusterConditionType
		want1 metav1.Time
	}{
		{"case1", args{}, "", metav1.Time{}},
		{"case2", args{cluster: cluster1}, "type1", metav1.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetStatusFromCluster(tt.args.cluster)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStatusFromCluster() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("GetStatusFromCluster() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
