// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package workset

import (
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkSetStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	if !Strategy.NamespaceScoped() {
		t.Errorf("WorkSet must be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("WorkSet should not allow create on update")
	}
	if !Strategy.AllowUnconditionalUpdate() {
		t.Errorf("WorkSet should not allow unconditional update")
	}
	cfg := &mcm.WorkSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "WorkSet1",
			Namespace: "WorkSet1",
		},
		Spec: mcm.WorkSetSpec{
			Template: mcm.WorkTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Work1",
				},
			},
		},
	}

	Strategy.PrepareForCreate(ctx, cfg)
	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}
	newCfg := &mcm.WorkSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "WorkSet1",
			Namespace: "WorkSet1",
		},
		Spec: mcm.WorkSetSpec{
			Template: mcm.WorkTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Work2",
				},
			},
		},
	}

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestWorkSetStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	cfg := &mcm.WorkSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "WorkSet1",
			Namespace: "WorkSet1",
		},
		Spec: mcm.WorkSetSpec{
			Template: mcm.WorkTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Work1",
				},
			},
		},
	}

	StatusStrategy.PrepareForCreate(ctx, cfg)

	newCfg := &mcm.WorkSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "WorkSet1",
			Namespace: "WorkSet1",
		},
		Spec: mcm.WorkSetSpec{
			Template: mcm.WorkTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "Work2",
				},
			},
		},
	}

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)
	errs := StatusStrategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}
	errs = StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}
