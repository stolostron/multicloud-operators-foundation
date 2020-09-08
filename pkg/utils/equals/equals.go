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

// EqualStringSlice compares 2 slices
func EqualStringSlice(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for _, i := range s1 {
		contained := false
		for _, j := range s2 {
			if i == j {
				contained = true
				break
			}
		}
		if !contained {
			return false
		}
	}

	return true
}
