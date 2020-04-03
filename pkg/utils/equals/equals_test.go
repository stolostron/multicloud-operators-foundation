// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEqualWorkSpec(t *testing.T) {
	WorkSpecNil := &v1beta1.WorkSpec{}
	WorkSpecP1 := &v1beta1.WorkSpec{Type: v1beta1.WorkType("testing")}
	WorkSpecP2 := &v1beta1.WorkSpec{Type: v1beta1.WorkType("testing1")}
	WorkSpecP3 := &v1beta1.WorkSpec{Scope: v1beta1.ResourceFilter{Name: "name1"}}
	WorkSpecP4 := &v1beta1.WorkSpec{Scope: v1beta1.ResourceFilter{Name: "name2"}}
	WorkSpecP5 := &v1beta1.WorkSpec{KubeWork: &v1beta1.KubeWorkSpec{Namespace: "test1"}}
	WorkSpecP6 := &v1beta1.WorkSpec{KubeWork: &v1beta1.KubeWorkSpec{Namespace: "test1"}}
	WorkSpecP7 := &v1beta1.WorkSpec{KubeWork: &v1beta1.KubeWorkSpec{Namespace: "test3"}}

	type args struct {
		spec1 *v1beta1.WorkSpec
		spec2 *v1beta1.WorkSpec
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1:", args{spec1: WorkSpecP1, spec2: WorkSpecP1}, true},
		{"case2:", args{spec1: WorkSpecNil, spec2: WorkSpecP1}, false},
		{"case3:", args{spec1: WorkSpecP1, spec2: WorkSpecP2}, false},
		{"case4:", args{spec1: WorkSpecP3, spec2: WorkSpecP4}, false},
		{"case8:", args{spec1: WorkSpecP5, spec2: WorkSpecP6}, true},
		{"case9:", args{spec1: WorkSpecP6, spec2: WorkSpecP7}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualWorkSpec(tt.args.spec1, tt.args.spec2); got != tt.want {
				t.Errorf("EqualWorkSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqualWorkScope(t *testing.T) {
	ResourceFilterNil := &v1beta1.ResourceFilter{}
	ResourceFilterTpye1 := &v1beta1.ResourceFilter{ResourceType: "name1"}
	ResourceFilterTpye2 := &v1beta1.ResourceFilter{ResourceType: "name2"}
	ResourceFilterNamespace1 := &v1beta1.ResourceFilter{NameSpace: "namespace1"}
	ResourceFilterNamespace2 := &v1beta1.ResourceFilter{NameSpace: "namespace2"}
	ResourceFilterName1 := &v1beta1.ResourceFilter{Name: "names1"}
	ResourceFilterName2 := &v1beta1.ResourceFilter{Name: "names2"}
	ResourceFilterVersion1 := &v1beta1.ResourceFilter{Version: "version1"}
	ResourceFilterVersion2 := &v1beta1.ResourceFilter{Version: "version2"}
	MatchLabels1 := &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "value1", "label2": "value2"}}
	ResourceFilterLabelSelector1 := &v1beta1.ResourceFilter{LabelSelector: MatchLabels1}
	MatchLabels2 := &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "value1", "label3": "value3"}}
	ResourceFilterLabelSelector2 := &v1beta1.ResourceFilter{LabelSelector: MatchLabels2}

	type args struct {
		f1 *v1beta1.ResourceFilter
		f2 *v1beta1.ResourceFilter
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1:", args{f1: ResourceFilterTpye1, f2: ResourceFilterTpye1}, true},
		{"case2:", args{f1: ResourceFilterNil, f2: ResourceFilterTpye1}, false},
		{"case3:", args{f1: ResourceFilterTpye1, f2: ResourceFilterTpye2}, false},
		{"case4:", args{f1: ResourceFilterNamespace1, f2: ResourceFilterNamespace2}, false},
		{"case5:", args{f1: ResourceFilterName1, f2: ResourceFilterName2}, false},
		{"case6:", args{f1: ResourceFilterVersion1, f2: ResourceFilterVersion2}, false},
		{"case7:", args{f1: ResourceFilterLabelSelector1, f2: ResourceFilterLabelSelector2}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualWorkScope(tt.args.f1, tt.args.f2); got != tt.want {
				t.Errorf("EqualWorkScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEqualLabelSelector(t *testing.T) {
	type args struct {
		selector1 *metav1.LabelSelector
		selector2 *metav1.LabelSelector
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1:", args{selector1: nil, selector2: nil}, true},
		{"case2:", args{
			selector1: nil,
			selector2: &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "va1", "label2": "va2"}}},
			false},
		{"case3:", args{
			selector1: &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "va1", "label2": "va2"}},
			selector2: &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "va1", "label2": "va2"}}},
			true},
		{"case3:", args{
			selector1: &metav1.LabelSelector{MatchLabels: map[string]string{"label2": "va2"}},
			selector2: &metav1.LabelSelector{MatchLabels: map[string]string{"label1": "va1", "label2": "va2"}}},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualLabelSelector(tt.args.selector1, tt.args.selector2); got != tt.want {
				t.Errorf("MatchLabelForLabelSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_EqualResourceList(t *testing.T) {
	testCases := []struct {
		name          string
		resourceList1 corev1.ResourceList
		resourceList2 corev1.ResourceList
		rst           bool
	}{
		{
			name: "case1",
			resourceList1: corev1.ResourceList{
				"resource1": resource.Quantity{},
			},
			resourceList2: corev1.ResourceList{},
			rst:           false,
		},
		{
			name: "case2",
			resourceList1: corev1.ResourceList{
				"resource1": resource.Quantity{},
			},
			resourceList2: corev1.ResourceList{
				"resource2": resource.Quantity{},
			},
			rst: false,
		},
	}

	for _, testCase := range testCases {
		rst := EqualResourceList(testCase.resourceList1, testCase.resourceList2)
		if rst != testCase.rst {
			t.Errorf("test case %s fails", testCase.name)
		}
	}
}

func Test_EqualEndpointAddresses(t *testing.T) {
	testCases := []struct {
		name string
		es1  []corev1.EndpointAddress
		es2  []corev1.EndpointAddress
		rst  bool
	}{
		{
			name: "case1",
			es1: []corev1.EndpointAddress{
				{
					IP: "1.1.1.1",
				},
			},
			es2: []corev1.EndpointAddress{},
			rst: false,
		},
		{
			name: "case2",
			es1: []corev1.EndpointAddress{
				{
					IP:       "1.1.1.1",
					Hostname: "host1",
				},
			},
			es2: []corev1.EndpointAddress{
				{
					IP:       "1.1.1.1",
					Hostname: "host2",
				},
			},
			rst: false,
		},
		{
			name: "case3",
			es1: []corev1.EndpointAddress{
				{
					IP:       "1.1.1.1",
					Hostname: "host1",
				},
			},
			es2: []corev1.EndpointAddress{
				{
					IP:       "1.1.1.1",
					Hostname: "host1",
				},
			},
			rst: true,
		},
	}

	for _, testCase := range testCases {
		rst := EqualEndpointAddresses(testCase.es1, testCase.es2)
		if rst != testCase.rst {
			t.Errorf("test case %s fails", testCase.name)
		}
	}
}
