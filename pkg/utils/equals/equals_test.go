// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
