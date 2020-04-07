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

package clusterjoinrequest

import (
	"testing"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	certificates "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

var clientCertUsage = []certificates.KeyUsage{
	certificates.UsageDigitalSignature,
	certificates.UsageKeyEncipherment,
	certificates.UsageClientAuth,
}

func TestClusterJoinRequestStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	if Strategy.NamespaceScoped() {
		t.Errorf("ClusterJoinRequest must not be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("ClusterJoinRequest should not allow create on update")
	}
	if !Strategy.AllowUnconditionalUpdate() {
		t.Errorf("ClusterJoinRequest should not allow unconditional update")
	}
	cfg := &mcm.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acmjoinName",
		},
		Spec: mcm.ClusterJoinRequestSpec{
			ClusterName:      "clustername1",
			ClusterNamespace: "clusternamespace1",
			CSR: certificates.CertificateSigningRequestSpec{
				Request: []byte(""),
				Usages:  clientCertUsage,
			},
		},
	}

	Strategy.PrepareForCreate(ctx, cfg)

	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := &mcm.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acmjoinName",
		},
		Spec: mcm.ClusterJoinRequestSpec{
			ClusterName:      "clustername2",
			ClusterNamespace: "clusternamespace2",
			CSR: certificates.CertificateSigningRequestSpec{
				Request: []byte("update"),
				Usages:  clientCertUsage,
			},
		},
	}

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestClusterJoinRequestStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()

	cfg := &mcm.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acmjoinName",
		},
		Spec: mcm.ClusterJoinRequestSpec{
			ClusterName:      "clustername1",
			ClusterNamespace: "clusternamespace1",
			CSR: certificates.CertificateSigningRequestSpec{
				Request: []byte(""),
				Usages:  clientCertUsage,
			},
		},
	}

	StatusStrategy.PrepareForCreate(ctx, cfg)

	newCfg := &mcm.ClusterJoinRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: "acmjoinName",
		},
		Spec: mcm.ClusterJoinRequestSpec{
			ClusterName:      "clustername2",
			ClusterNamespace: "clusternamespace2",
			CSR: certificates.CertificateSigningRequestSpec{
				Request: []byte("update"),
				Usages:  clientCertUsage,
			},
		},
	}

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs := StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}
