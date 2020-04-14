// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package cluster

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func newCluster(name string, namespace string, serverAddress string) runtime.Object {
	return &clusterregistry.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{},
		},
		Spec: clusterregistry.ClusterSpec{
			KubernetesAPIEndpoints: clusterregistry.KubernetesAPIEndpoints{
				ServerEndpoints: []clusterregistry.ServerAddressByClientCIDR{
					{
						ServerAddress: serverAddress,
					},
				},
			},
		},
		Status: clusterregistry.ClusterStatus{
			Conditions: []clusterregistry.ClusterCondition{
				{
					Type:   clusterregistry.ClusterOK,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}
func TestClusterStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	if !Strategy.NamespaceScoped() {
		t.Errorf("Cluster must not be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("Cluster should not allow create on update")
	}
	if !Strategy.AllowUnconditionalUpdate() {
		t.Errorf("Cluster should not allow unconditional update")
	}
	cfg := newCluster("cluster1", "clusternamespacec1", "127.0.0.1:8443")

	Strategy.PrepareForCreate(ctx, cfg)

	errs := Strategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := newCluster("cluster1", "clusternamespacec1", "127.0.0.2:8443")

	Strategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = Strategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestClusterStatusStrategy(t *testing.T) {
	ctx := genericapirequest.NewDefaultContext()
	cfg := newCluster("cluster2", "clusternamespacec2", "127.0.0.1:8443")

	StatusStrategy.PrepareForCreate(ctx, cfg)

	errs := StatusStrategy.Validate(ctx, cfg)
	if len(errs) != 0 {
		t.Errorf("unexpected error validating %v", errs)
	}

	newCfg := newCluster("cluster2", "clusternamespacec2", "127.0.0.2:8443")

	StatusStrategy.PrepareForUpdate(ctx, newCfg, cfg)

	errs = StatusStrategy.ValidateUpdate(ctx, newCfg, cfg)
	if len(errs) != 0 {
		t.Errorf("Validation error")
	}
}

func TestGetAttrs(t *testing.T) {
	cluster1 := newCluster("cluster1", "clusternamespacec1", "127.0.0.2:8443")
	MatchCluster(nil, nil)
	_, _, err := GetAttrs(cluster1)
	if err != nil {
		t.Errorf("error in GetAttrs")
	}
	_, _, err = GetAttrs(nil)
	if err == nil {
		t.Errorf("error in GetAttrs")
	}
}
