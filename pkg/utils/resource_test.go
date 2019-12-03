// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been
// deposited with the U.S. Copyright Office.

package utils

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	v1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestIsClusterExit(t *testing.T) {
	ServerAddressByClientCIDR1 := v1alpha1.ServerAddressByClientCIDR{ClientCIDR: "/opt"}
	ServerAddressByClientCIDR2 := v1alpha1.ServerAddressByClientCIDR{ClientCIDR: "/usr"}
	clusters1 := &v1alpha1.Cluster{Spec: v1alpha1.ClusterSpec{KubernetesAPIEndpoints: v1alpha1.KubernetesAPIEndpoints{
		ServerEndpoints: []v1alpha1.ServerAddressByClientCIDR{ServerAddressByClientCIDR1, ServerAddressByClientCIDR2}}}}

	type args struct {
		ClusterName string
		clusters    []*v1alpha1.Cluster
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1:", args{ClusterName: "clusterName1"}, false},
		{"case2:", args{ClusterName: "clusterName1", clusters: []*v1alpha1.Cluster{clusters1}}, false},
		{"case3:", args{ClusterName: "", clusters: nil}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClusterExit(tt.args.ClusterName, tt.args.clusters); got != tt.want {
				t.Errorf("IsClusterExit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatQuatityToMi(t *testing.T) {
	type args struct {
		q resource.Quantity
	}
	m1 := *resource.NewQuantity(int64(1024*1024), resource.BinarySI)
	m2 := *resource.NewQuantity(int64(1024*1024*2), resource.BinarySI)
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{"case1:", args{m1}, "1Mi"},
		{"case1:", args{m2}, "2Mi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuatityToMi(tt.args.q); !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("FormatQuatityToMi() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestFormatQuatityToGi(t *testing.T) {
	type args struct {
		q resource.Quantity
	}
	m1 := *resource.NewQuantity(int64(1024*1024*1024), resource.BinarySI)
	m2 := *resource.NewQuantity(int64(1024*1024*1024*2), resource.BinarySI)
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{"case1:", args{m1}, "1Gi"},
		{"case1:", args{m2}, "2Gi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatQuatityToGi(tt.args.q); !reflect.DeepEqual(got.String(), tt.want) {
				t.Errorf("FormatQuatityToMi() = %v, want %v", got, tt.want)
			}
		})
	}
}
