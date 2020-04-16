// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package workset

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

func newWorkSet(name string, namespace string, workname string) runtime.Object {
	return &mcm.WorkSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: mcm.WorkSetSpec{
			Template: mcm.WorkTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: workname,
				},
			},
		},
	}
}

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
	cfg := newWorkSet("ws1", "ws1", "w1")

	Strategy.PrepareForCreate(ctx, cfg)
	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}
	newCfg := newWorkSet("ws1", "ws1", "w2")

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestWorkSetStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	cfg := newWorkSet("ws2", "ws2", "w1")

	StatusStrategy.PrepareForCreate(ctx, cfg)

	newCfg := newWorkSet("ws2", "ws2", "w2")

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

func TestGetAttrs(t *testing.T) {
	rv1 := newWorkSet("ws1", "ws1", "w2")
	MatchWorkSet(nil, nil)
	_, _, err := GetAttrs(rv1)
	if err != nil {
		t.Errorf("error in GetAttrs")
	}
	_, _, err = GetAttrs(nil)
	if err == nil {
		t.Errorf("error in GetAttrs")
	}
}
