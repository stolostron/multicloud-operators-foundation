// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package workset

import (
	"fmt"
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	clusterfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type workSetController struct {
	*Controller

	clusterStore cache.Store
	worksetStore cache.Store
	workStore    cache.Store
}

var (
	alwaysReady = func() bool { return true }
)

func newTestController() (*workSetController, *fake.Clientset) {
	clientset := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Minute*10)

	clusterFakeClient := clusterfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterFakeClient, time.Minute*10)

	rvc := NewController(clientset, nil, informerFactory, clusterInformerFactory, true, nil)
	rvc.clusterSynced = alwaysReady
	rvc.workSynced = alwaysReady
	rvc.worksetSynced = alwaysReady

	return &workSetController{
		rvc,
		clusterInformerFactory.Clusterregistry().V1alpha1().Clusters().Informer().GetStore(),
		informerFactory.Mcm().V1beta1().WorkSets().Informer().GetStore(),
		informerFactory.Mcm().V1beta1().Works().Informer().GetStore(),
	}, clientset
}

func newCluster(name, namespace string, status clusterv1alpha1.ClusterConditionType, labels map[string]string) *clusterv1alpha1.Cluster {
	return &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    labels,
			Namespace: namespace,
		},
		Status: clusterv1alpha1.ClusterStatus{
			Conditions: []clusterv1alpha1.ClusterCondition{
				{
					Type: status,
				},
			},
		},
	}
}

func newWorkset(name, namespace string) *v1beta1.WorkSet {
	return &v1beta1.WorkSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.WorkSetSpec{
			Template: v1beta1.WorkTemplateSpec{
				Spec: v1beta1.WorkSpec{
					Type: v1beta1.ActionWorkType,
				},
			},
		},
	}
}

func newWork(name, namespace, cluster, worksetName, worksetNamespace string) *v1beta1.Work {
	return &v1beta1.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				mcm.WorkSetLabel: worksetNamespace + "." + worksetName,
			},
		},
		Spec: v1beta1.WorkSpec{
			Type:    v1beta1.ActionWorkType,
			Cluster: corev1.LocalObjectReference{Name: cluster},
		},
	}
}

func syncAndValidateWorkset(
	t *testing.T, manager *workSetController,
	client *fake.Clientset,
	workset *v1beta1.WorkSet,
	verb string,
	expectedCreates int,
) {
	key, err := cache.MetaNamespaceKeyFunc(workset)
	if err != nil {
		t.Errorf("Could not get key for daemon.")
	}
	manager.processWorkSet(key)
	validateSyncWorksets(t, client, verb, expectedCreates)
}

func validateSyncWorksets(t *testing.T, client *fake.Clientset, verb string, expectedCreates int) {
	works := client.Actions()
	count := 0
	for _, work := range works {
		if work.Matches(verb, "works") {
			count++
		}
	}
	if count != expectedCreates {
		t.Errorf("Unexpected number of creates.  Expected %d, saw %d\n", expectedCreates, count)
	}
}

func TestFilterCluster(t *testing.T) {
	cluster1 := newCluster("cluster1", "cluster1", clusterv1alpha1.ClusterOK, map[string]string{})
	cluster2 := newCluster("cluster2", "cluster2", "", map[string]string{})
	workset := newWorkset("workset1", "workset1")

	manager, client := newTestController()

	manager.clusterStore.Add(cluster1)
	manager.clusterStore.Add(cluster2)
	manager.worksetStore.Add(workset)

	syncAndValidateWorkset(t, manager, client, workset, "create", 1)
	work1 := newWork("work1", "cluster1", "cluster1", "workset1", "workset1")
	manager.workStore.Add(work1)

	// Add a cluster
	fmt.Printf("add clusters\n")
	cluster3 := newCluster("cluster3", "cluster3", clusterv1alpha1.ClusterOK, map[string]string{})
	manager.clusterStore.Add(cluster3)
	syncAndValidateWorkset(t, manager, client, workset, "create", 2)
	work3 := newWork("work3", "cluster3", "cluster3", "workset1", "workset1")
	manager.workStore.Add(work3)

	fmt.Printf("update workset\n")
	workset.Spec.Template.Spec.ActionType = "CREATE"
	manager.worksetStore.Update(workset)
	syncAndValidateWorkset(t, manager, client, workset, "update", 2)
}

func TestGetClustersToWorks(t *testing.T) {
	work1 := newWork("work1", "work1", "cluster1", "workset1", "workset1")
	work2 := newWork("work2", "work2", "cluster2", "workset1", "workset1")
	workset := newWorkset("workset1", "workset1")
	manager, _ := newTestController()

	manager.workStore.Add(work1)
	manager.workStore.Add(work2)

	getClustersToWorks, err := manager.getClustersToWorks(workset)
	if err != nil {
		t.Errorf("Failed to get cluster to works %v", err)
	}

	if len(getClustersToWorks) != 2 {
		t.Errorf("getClustersToWorks() = %v, want %v", len(getClustersToWorks), 2)
	}
}

func TestWorksShouldBeOnClusters(t *testing.T) {
	manager, _ := newTestController()

	cluster1 := newCluster("cluster1", "cluster1", clusterv1alpha1.ClusterOK, map[string]string{})
	cluster2 := newCluster("cluster2", "cluster2", clusterv1alpha1.ClusterOK, map[string]string{})
	clusters := []*clusterv1alpha1.Cluster{cluster1, cluster2}
	workset := newWorkset("workset1", "workset1")

	clusterToWorks := map[string][]*v1beta1.Work{}
	clustersNeedingWorks, _, _ := manager.worksShouldBeOnClusters(workset, clusterToWorks, clusters)

	if len(clustersNeedingWorks) != 2 {
		t.Errorf("getClustersToWorks() clustersNeedingWorks = %v, want %v", len(clustersNeedingWorks), 2)
	}

	work1 := newWork("work1", "cluster1", "cluster1", "workset1", "workset1")
	work1.Spec.Scope.ResourceType = "pods"
	work2 := newWork("work2", "cluster3", "cluster3", "workset1", "workset1")
	manager.workStore.Add(work1)
	manager.workStore.Add(work2)
	clusterToWorks, err := manager.getClustersToWorks(workset)
	if err != nil {
		t.Errorf("Failed to get cluster to works %v", err)
	}

	if len(clusterToWorks) != 2 {
		t.Errorf("getClustersToWorks() = %v, want %v", len(clusterToWorks), 2)
	}

	clustersNeedingWorks, worksToDelete, workToUpdate := manager.worksShouldBeOnClusters(workset, clusterToWorks, clusters)
	if len(clustersNeedingWorks) != 1 {
		t.Errorf("getClustersToWorks() clustersNeedingWorks = %v, want %v", len(clustersNeedingWorks), 1)
	}
	if len(worksToDelete) != 1 {
		t.Errorf("getClustersToWorks() worksToDelete = %v, want %v", len(worksToDelete), 1)
	}
	if len(workToUpdate) != 1 {
		t.Errorf("getClustersToWorks() workToUpdate = %v, want %v", len(workToUpdate), 1)
	}
}

func TestAddWork(t *testing.T) {
	manager, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")
	manager.addWork(work)
	if manager.workqueue.Len() != 0 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 0)
	}

	work.Status.Type = v1beta1.WorkCompleted
	manager.addWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}
}

func TestUpdateWork(t *testing.T) {
	manager, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")
	manager.updateWork(work, work)
	if manager.workqueue.Len() != 0 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 0)
	}

	newwork := work.DeepCopy()
	newwork.Status.Type = v1beta1.WorkCompleted
	manager.updateWork(work, newwork)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}
	work2 := newWork("work2", "work2", "cluster2", "view2", "view2")
	newwork2 := work2.DeepCopy()
	newwork2.Status.Type = v1beta1.WorkFailed
	manager.updateWork(work2, newwork2)
	if manager.workqueue.Len() != 2 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 2)
	}
}

func TestDeleteWork(t *testing.T) {
	manager, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")
	manager.deleteWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}

	work.Status.Type = v1beta1.WorkCompleted
	manager.addWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}
}
