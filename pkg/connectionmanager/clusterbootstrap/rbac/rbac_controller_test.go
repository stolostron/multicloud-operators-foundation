// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package rbac

import (
	"testing"
	"time"

	clusterfake "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

var (
	alwaysReady = func() bool { return true }
)

func TestSyncCluster(t *testing.T) {
	cluster1 := newCluster("cluster1", "cluster1")
	cluster2 := newCluster("cluster2", "cluster2")
	fakeKubeClient := kubefake.NewSimpleClientset()
	clusterFakeClient := clusterfake.NewSimpleClientset()
	clusterinformer := clusterinformers.NewSharedInformerFactory(clusterFakeClient, 10*time.Minute)

	controller := NewClusterRBACController(fakeKubeClient, clusterinformer, nil)
	controller.clusterSyced = alwaysReady
	store := clusterinformer.Clusterregistry().V1alpha1().Clusters().Informer().GetStore()
	store.Add(cluster1)
	store.Add(cluster2)
	controller.syncCluster()
	count := actionCount("create", "roles", fakeKubeClient.Actions())
	if count != 2 {
		t.Errorf("failed to create expected number of clusters: actual %d; expected 2", count)
	}
}
