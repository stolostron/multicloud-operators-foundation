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

package leadervote

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

func TestLeaderVoteStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	if Strategy.NamespaceScoped() {
		t.Errorf("LeaderVote must not be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("LeaderVote should not allow create on update")
	}
	if !Strategy.AllowUnconditionalUpdate() {
		t.Errorf("LeaderVote should not allow unconditional update")
	}
	cfg := &mcm.LeaderVote{
		ObjectMeta: metav1.ObjectMeta{
			Name: "LeaderVote1",
		},
		Spec: mcm.LeaderVoteSpec{
			Vote: 1,
		},
	}

	Strategy.PrepareForCreate(ctx, cfg)

	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := &mcm.LeaderVote{
		ObjectMeta: metav1.ObjectMeta{
			Name: "LeaderVote1",
		},
		Spec: mcm.LeaderVoteSpec{
			Vote: 2,
		},
	}

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestLeaderVoteStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	cfg := &mcm.LeaderVote{
		ObjectMeta: metav1.ObjectMeta{
			Name: "LeaderVote1",
		},
		Spec: mcm.LeaderVoteSpec{
			Vote: 1,
		},
	}

	StatusStrategy.PrepareForCreate(ctx, cfg)

	errs := StatusStrategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := &mcm.LeaderVote{
		ObjectMeta: metav1.ObjectMeta{
			Name: "LeaderVote1",
		},
		Spec: mcm.LeaderVoteSpec{
			Vote: 2,
		},
	}

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}
