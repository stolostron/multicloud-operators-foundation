/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package placementpolicy

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

var replica int32
var newReplica int32

func TestPlacementPolicyStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	if !Strategy.NamespaceScoped() {
		t.Errorf("PlacementPolicy must be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("PlacementPolicy should not allow create on update")
	}
	if !Strategy.AllowUnconditionalUpdate() {
		t.Errorf("PlacementPolicy should not allow unconditional update")
	}
	replica = 2
	cfg := &mcm.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "PlacementPolicy1",
		},
		Spec: mcm.PlacementPolicySpec{
			Replicas: &replica,
		},
	}

	Strategy.PrepareForCreate(ctx, cfg)

	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}
	newReplica = 3
	newCfg := &mcm.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "PlacementPolicy1",
		},
		Spec: mcm.PlacementPolicySpec{
			Replicas: &newReplica,
		},
	}

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestPlacementPolicyStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	replica = 2
	cfg := &mcm.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "PlacementPolicy1",
		},
		Spec: mcm.PlacementPolicySpec{
			Replicas: &replica,
		},
	}

	StatusStrategy.PrepareForCreate(ctx, cfg)

	errs := StatusStrategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newReplica = 3
	newCfg := &mcm.PlacementPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "PlacementPolicy1",
		},
		Spec: mcm.PlacementPolicySpec{
			Replicas: &newReplica,
		},
	}

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}
