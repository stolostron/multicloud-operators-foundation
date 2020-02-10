// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"reflect"
	"testing"

	v1alpha1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func TestConvertLabels(t *testing.T) {
	type args struct {
		labelSelector *metav1.LabelSelector
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{"case1:", args{labelSelector: &metav1.LabelSelector{
			MatchLabels: nil, MatchExpressions: nil}}, &metav1.LabelSelector{MatchLabels: nil, MatchExpressions: nil}, false},
		{"case2:", args{labelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"label1": "value1", "label2": "value2"}}},
			&metav1.LabelSelector{MatchLabels: map[string]string{"label1": "value1", "label2": "value2"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertLabels(tt.args.labelSelector)
			wantnew, _ := metav1.LabelSelectorAsSelector(tt.want)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, wantnew) {
				t.Errorf("ConvertLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetWorkSetFromWork(t *testing.T) {
	selector1 := &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "value1", "label2": "value2"}}
	work1 := &v1alpha1.Work{Spec: v1alpha1.WorkSpec{Scope: v1alpha1.ResourceFilter{LabelSelector: selector1}}}
	workset1 := &v1alpha1.WorkSet{Spec: v1alpha1.WorkSetSpec{Selector: selector1},
		Status: v1alpha1.WorkSetStatus{Status: v1alpha1.WorkStatusType("running")}}

	type args struct {
		work     *v1alpha1.Work
		worksets []*v1alpha1.WorkSet
	}
	tests := []struct {
		name    string
		args    args
		want    []*v1alpha1.WorkSet
		wantErr bool
	}{
		{"case1:", args{work: work1, worksets: nil}, nil, true},
		{"case2:", args{work: work1, worksets: []*v1alpha1.WorkSet{workset1}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetWorkSetFromWork(tt.args.work, tt.args.worksets)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWorkSetFromWork() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWorkSetFromWork() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterHealthyClusters(t *testing.T) {
	ClusterCondition1 := clusterv1alpha1.ClusterCondition{Status: v1.ConditionStatus("running")}
	ClusterCondition2 := clusterv1alpha1.ClusterCondition{Status: v1.ConditionStatus("stopped")}
	cluster1 := &clusterv1alpha1.Cluster{Status: clusterv1alpha1.ClusterStatus{
		Conditions: []clusterv1alpha1.ClusterCondition{ClusterCondition1, ClusterCondition2}}}
	type args struct {
		clusters []*clusterv1alpha1.Cluster
	}
	tests := []struct {
		name string
		args args
		want []*clusterv1alpha1.Cluster
	}{

		{"case1:", args{clusters: []*clusterv1alpha1.Cluster{}}, []*clusterv1alpha1.Cluster{}},
		{"case2:", args{clusters: []*clusterv1alpha1.Cluster{cluster1}}, []*clusterv1alpha1.Cluster{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterHealthyClusters(tt.args.clusters); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterHealthyClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}
