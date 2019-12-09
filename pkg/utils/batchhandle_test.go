// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"testing"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
)

func TestBatchHandle(t *testing.T) {
	count := []int{1, 2, 3, 9, 10, 11, 14, 20, 80, 99, 100, 101, 200, 134, 500}

	for _, cnt := range count {
		workList := makeWorkList(cnt)

		handle := func(i int) {
			workList[i].Status.Type = v1alpha1.WorkCompleted
		}

		BatchHandle(len(workList), handle)

		if !checkResult(workList) {
			t.Fatalf("Handle error %v", cnt)
		}
	}
}

func makeWorkList(n int) []v1alpha1.Work {
	return make([]v1alpha1.Work, n)
}

func checkResult(workList []v1alpha1.Work) bool {
	for _, work := range workList {
		if work.Status.Type != v1alpha1.WorkCompleted {
			return false
		}
	}
	return true
}
