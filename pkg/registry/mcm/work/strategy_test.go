// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package work

import (
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
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

func TestWorkStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	if !Strategy.NamespaceScoped() {
		t.Errorf("Work must be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("Work should not allow create on update")
	}
	if !Strategy.AllowUnconditionalUpdate() {
		t.Errorf("Work should not allow unconditional update")
	}
	cfg := &mcm.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "work1",
			Namespace: "work1",
		},
		Spec: mcm.WorkSpec{
			Type:  mcm.ResourceWorkType,
			Scope: mcm.ResourceFilter{},
		},
	}

	Strategy.PrepareForCreate(ctx, cfg)

	newCfg := &mcm.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "work1",
			Namespace: "work1",
		},
		Spec: mcm.WorkSpec{
			Type:  mcm.ActionWorkType,
			Scope: mcm.ResourceFilter{},
		},
	}

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs := Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) == 0 {
		t.Errorf("Validation error")
	}
}

func TestWorkStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	cfg := &mcm.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "work1",
			Namespace: "work1",
		},
		Spec: mcm.WorkSpec{
			Type:  mcm.ResourceWorkType,
			Scope: mcm.ResourceFilter{},
		},
	}

	StatusStrategy.PrepareForCreate(ctx, cfg)

	newCfg := &mcm.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "work1",
			Namespace: "work1",
		},
		Spec: mcm.WorkSpec{
			Type:  mcm.ActionWorkType,
			Scope: mcm.ResourceFilter{},
		},
	}

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs := StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}
