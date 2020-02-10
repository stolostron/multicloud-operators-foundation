// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package work

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateWork(t *testing.T) {
	work := &mcm.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "work1",
			Namespace: "work1",
		},
		Spec: mcm.WorkSpec{
			Type:  mcm.ResourceWorkType,
			Scope: mcm.ResourceFilter{},
		},
	}

	err := validateWork(work)
	if err == nil {
		t.Errorf("should failed to validate work")
	}

	work.Spec.Cluster.Name = "cluster1"
	err = validateWork(work)
	if err == nil {
		t.Errorf("should failed to validate work")
	}

	work.Spec.Scope.ResourceType = "pods"
	err = validateWork(work)
	if err != nil {
		t.Errorf("should be able to validate work")
	}

	work.Spec.Type = mcm.ActionWorkType
	err = validateWork(work)
	if err == nil {
		t.Errorf("should failed to validate work")
	}

	work.Spec.KubeWork = &mcm.KubeWorkSpec{}
	work.Spec.HelmWork = &mcm.HelmWorkSpec{}
	err = validateWork(work)
	if err == nil {
		t.Errorf("should failed to validate work")
	}
}
