// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EqualLabelSelector check if label selector are equal
func EqualLabelSelector(selector1, selector2 *metav1.LabelSelector) bool {
	return reflect.DeepEqual(selector1, selector2)
}

func EqualResourceList(rl1, rl2 corev1.ResourceList) bool {
	if len(rl1) != len(rl2) {
		return false
	}

	for key, rs1 := range rl1 {
		rs2, ok := rl2[key]
		if !ok {
			return false
		}
		if rs1.Value() != rs2.Value() {
			return false
		}
	}

	return true
}

func EqualEndpointAddresses(es1, es2 []corev1.EndpointAddress) bool {
	if len(es1) != len(es2) {
		return false
	}

	for idx, e := range es1 {
		cure := e
		if !EqualEndpointAddress(&cure, &es2[idx]) {
			return false
		}
	}

	return true
}

// EqualEndpointAddress compares the two endpoint address
func EqualEndpointAddress(e1, e2 *corev1.EndpointAddress) bool {
	if e1 == e2 {
		return true
	}
	if e1 == nil || e2 == nil {
		return false
	}
	if e1.Hostname != e2.Hostname {
		return false
	}
	if e1.IP != e2.IP {
		return false
	}

	return true
}
