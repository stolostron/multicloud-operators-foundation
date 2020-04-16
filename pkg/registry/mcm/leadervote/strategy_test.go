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
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

func newLeaderVote(name string, vote int32) runtime.Object {
	return &mcm.LeaderVote{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: mcm.LeaderVoteSpec{
			Vote: vote,
		},
	}
}
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
	cfg := newLeaderVote("leadervote1", 1)

	Strategy.PrepareForCreate(ctx, cfg)

	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := newLeaderVote("leadervote1", 2)

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestLeaderVoteStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	cfg := newLeaderVote("leadervote2", 1)

	StatusStrategy.PrepareForCreate(ctx, cfg)

	errs := StatusStrategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := newLeaderVote("leadervote2", 2)

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestGetAttrs(t *testing.T) {
	cjr1 := newLeaderVote("leadervote1", 2)
	MatchLeaderVote(nil, nil)
	_, _, err := GetAttrs(cjr1)
	if err != nil {
		t.Errorf("error in GetAttrs")
	}
	_, _, err = GetAttrs(nil)
	if err == nil {
		t.Errorf("error in GetAttrs")
	}
}
