// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package cluster

import (
	hcmfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	clientfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	hcminformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	"testing"

	"time"
)

var cluster = &clusterregistry.Cluster{
	TypeMeta: v1.TypeMeta{},
	ObjectMeta: v1.ObjectMeta{
		Name: "cluster1",
	},
	Spec: clusterregistry.ClusterSpec{},
	Status: clusterregistry.ClusterStatus{
		Conditions: []clusterregistry.ClusterCondition{
			{
				Type:              clusterregistry.ClusterOK,
				LastHeartbeatTime: metav1.Time{},
			},
		},
	},
}

func newController() *Controller {
	hcmClient := hcmfake.NewSimpleClientset()
	informerFactory := hcminformers.NewSharedInformerFactory(hcmClient, time.Minute*10)
	clusterClient := clientfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	informers := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters()

	informers.Informer().GetIndexer().Add(cluster)
	return NewController(hcmClient, informerFactory, clusterClient, clusterInformerFactory, 1*time.Minute, make(chan struct{}))
}

func TestClusterHealthCheck(t *testing.T) {
	controller := newController()
	controller.clusterHealthCheck()
	if cluster.Status.Conditions[0].Reason != offlineReason {
		t.Errorf("fail to update cluster")
	}
}
