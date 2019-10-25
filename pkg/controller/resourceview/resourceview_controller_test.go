// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package resourceview

import (
	"testing"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm"
	v1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	clusterfake "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	informers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

type resourceViewController struct {
	*ResourceViewController

	clusterStore cache.Store
	viewStore    cache.Store
	workStore    cache.Store
}

var (
	alwaysReady = func() bool { return true }
)

func newTestController(initialObjects ...runtime.Object) (*resourceViewController, *fake.Clientset, *clusterfake.Clientset, error) {
	clientset := fake.NewSimpleClientset(initialObjects...)
	informerFactory := informers.NewSharedInformerFactory(clientset, time.Minute*10)

	clusterFakeClient := clusterfake.NewSimpleClientset(initialObjects...)
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterFakeClient, time.Minute*10)

	rvc := NewResourceViewController(clientset, nil, informerFactory, clusterInformerFactory, true, nil)
	rvc.clusterSynced = alwaysReady
	rvc.workSynced = alwaysReady
	rvc.viewSynced = alwaysReady

	return &resourceViewController{
		rvc,
		clusterInformerFactory.Clusterregistry().V1alpha1().Clusters().Informer().GetStore(),
		informerFactory.Mcm().V1alpha1().ResourceViews().Informer().GetStore(),
		informerFactory.Mcm().V1alpha1().Works().Informer().GetStore(),
	}, clientset, clusterFakeClient, nil
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
				clusterv1alpha1.ClusterCondition{
					Type: status,
				},
			},
		},
	}
}

func newView(name, namespace string) *v1alpha1.ResourceView {
	return &v1alpha1.ResourceView{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ResourceViewSpec{},
	}
}

func newWork(name, namespace, cluster, view, viewNamespace string) *v1alpha1.Work {
	return &v1alpha1.Work{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				mcm.ViewLabel: viewNamespace + "." + view,
			},
		},
		Spec: v1alpha1.WorkSpec{
			Cluster: corev1.LocalObjectReference{Name: cluster},
		},
	}
}

func syncAndValidateView(
	t *testing.T, manager *resourceViewController,
	client *fake.Clientset,
	view *v1alpha1.ResourceView,
	verb string,
	expectedCreates int,
) {
	key, err := cache.MetaNamespaceKeyFunc(view)
	if err != nil {
		t.Errorf("Could not get key for daemon.")
	}
	manager.processView(key)
	validateSyncViews(t, client, verb, expectedCreates)
}

func validateSyncViews(t *testing.T, client *fake.Clientset, verb string, expectedCreates int) {
	works := client.Actions()
	count := 0
	for _, work := range works {
		if work.Matches(verb, "works") {
			count = count + 1
		}
	}
	if count != expectedCreates {
		t.Errorf("Unexpected number of creates.  Expected %d, saw %d\n", expectedCreates, count)
	}
}

func TestFilterCluster(t *testing.T) {
	cluster1 := newCluster("cluster1", "cluster1", clusterv1alpha1.ClusterOK, map[string]string{})
	cluster2 := newCluster("cluster2", "cluster2", "", map[string]string{})
	view := newView("view1", "view1")

	manager, client, _, err := newTestController()
	if err != nil {
		t.Fatalf("error creating resource view controller: %v", err)
	}

	manager.clusterStore.Add(cluster1)
	manager.clusterStore.Add(cluster2)
	manager.viewStore.Add(view)

	syncAndValidateView(t, manager, client, view, "create", 1)
	work1 := newWork("work1", "cluster1", "cluster1", "view1", "view1")
	manager.workStore.Add(work1)

	// Add a cluster
	cluster3 := newCluster("cluster3", "cluster3", clusterv1alpha1.ClusterOK, map[string]string{})
	manager.clusterStore.Add(cluster3)
	syncAndValidateView(t, manager, client, view, "create", 2)
	work3 := newWork("work3", "cluster3", "cluster3", "view1", "view1")
	manager.workStore.Add(work3)

	view.Spec.SummaryOnly = true
	manager.viewStore.Update(view)
	syncAndValidateView(t, manager, client, view, "update", 2)
}

func TestUpdateWorkByView(t *testing.T) {
	manager, _, _, _ := newTestController()

	type args struct {
		view *v1alpha1.ResourceView
		work *v1alpha1.Work
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"case1:",
			args{
				&v1alpha1.ResourceView{},
				&v1alpha1.Work{},
			},
			false,
		},
		{
			"case2:",
			args{
				&v1alpha1.ResourceView{Spec: v1alpha1.ResourceViewSpec{Mode: v1alpha1.PeriodicResourceUpdate}},
				&v1alpha1.Work{},
			},
			true,
		},
		{
			"case3:",
			args{
				&v1alpha1.ResourceView{Spec: v1alpha1.ResourceViewSpec{Scope: v1alpha1.ViewFilter{Resource: "pods", ResourceName: "pod1", NameSpace: "default"}}},
				&v1alpha1.Work{Spec: v1alpha1.WorkSpec{Scope: v1alpha1.ResourceFilter{ResourceType: "nodes", Name: "node1"}}},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, update := manager.updateWorkByView(tt.args.view, tt.args.work)
			if update != tt.want {
				t.Errorf("updateWork() = %v, want %v", update, tt.want)
			}
		})
	}
}

func TestNeedUpdate(t *testing.T) {
	manager, _, _, _ := newTestController()

	type args struct {
		view1 *v1alpha1.ResourceView
		view2 *v1alpha1.ResourceView
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"case1:",
			args{
				&v1alpha1.ResourceView{},
				&v1alpha1.ResourceView{},
			},
			false,
		},
		{
			"case2:",
			args{
				&v1alpha1.ResourceView{Spec: v1alpha1.ResourceViewSpec{Mode: v1alpha1.PeriodicResourceUpdate}},
				&v1alpha1.ResourceView{},
			},
			true,
		},
		{
			"case3:",
			args{
				&v1alpha1.ResourceView{Spec: v1alpha1.ResourceViewSpec{Scope: v1alpha1.ViewFilter{Resource: "pods", NameSpace: "ns1"}}},
				&v1alpha1.ResourceView{Spec: v1alpha1.ResourceViewSpec{Scope: v1alpha1.ViewFilter{Resource: "pods", NameSpace: "ns2"}}},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := manager.needsUpdate(tt.args.view1, tt.args.view2)
			if update != tt.want {
				t.Errorf("updateWork() = %v, want %v", update, tt.want)
			}
		})
	}

}

func TestEqualResourceViewSpec(t *testing.T) {
	type args struct {
		spec1 v1alpha1.ViewFilter
		spec2 v1alpha1.ViewFilter
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"case1:",
			args{
				v1alpha1.ViewFilter{},
				v1alpha1.ViewFilter{},
			},
			true,
		},
		{
			"case2:",
			args{
				v1alpha1.ViewFilter{APIGroup: "v1alph1"},
				v1alpha1.ViewFilter{APIGroup: "v1alph2"},
			},
			false,
		},
		{
			"case3:",
			args{
				v1alpha1.ViewFilter{Resource: "pods"},
				v1alpha1.ViewFilter{Resource: "secrets"},
			},
			false,
		},
		{
			"case4:",
			args{
				v1alpha1.ViewFilter{ResourceName: "pod1"},
				v1alpha1.ViewFilter{ResourceName: "pod2"},
			},
			false,
		},
		{
			"case5:",
			args{
				v1alpha1.ViewFilter{NameSpace: "namespace1"},
				v1alpha1.ViewFilter{NameSpace: "namespace2"},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			equal := equalResourceViewSpec(tt.args.spec1, tt.args.spec2)
			if equal != tt.want {
				t.Errorf("equalResourceViewSpec() = %v, want %v", equal, tt.want)
			}
		})
	}
}

func TestGetViewConditionType(t *testing.T) {
	type args struct {
		view v1alpha1.ResourceView
	}

	tests := []struct {
		name string
		args args
		want v1alpha1.WorkStatusType
	}{
		{
			"case1:",
			args{
				v1alpha1.ResourceView{Status: v1alpha1.ResourceViewStatus{Conditions: []v1alpha1.ViewCondition{}}},
			},
			"",
		},
		{
			"case2:",
			args{
				v1alpha1.ResourceView{},
			},
			"",
		},
		{
			"case1:",
			args{
				v1alpha1.ResourceView{
					Status: v1alpha1.ResourceViewStatus{
						Conditions: []v1alpha1.ViewCondition{
							v1alpha1.ViewCondition{
								Type: v1alpha1.WorkProcessing,
							},
							v1alpha1.ViewCondition{
								Type: v1alpha1.WorkCompleted,
							},
						},
					},
				},
			},
			v1alpha1.WorkCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getViewCondition(&tt.args.view).Type
			if string(status) != string(tt.want) {
				t.Errorf("getViewConditionType() = %v, want %v", status, tt.want)
			}
		})
	}
}

func TestGetClustersToWorks(t *testing.T) {
	work1 := newWork("work1", "work1", "cluster1", "view1", "view1")
	work2 := newWork("work2", "work2", "cluster2", "view1", "view1")
	view := newView("view1", "view1")
	manager, _, _, _ := newTestController()

	manager.workStore.Add(work1)
	manager.workStore.Add(work2)

	getClustersToWorks, err := manager.getClustersToWorks(view)
	if err != nil {
		t.Errorf("Failed to get cluster to works %v", err)
	}

	if len(getClustersToWorks) != 2 {
		t.Errorf("getClustersToWorks() = %v, want %v", len(getClustersToWorks), 2)
	}
}

func TestWorksShouldBeOnClusters(t *testing.T) {
	manager, _, _, _ := newTestController()

	cluster1 := newCluster("cluster1", "cluster1", clusterv1alpha1.ClusterOK, map[string]string{})
	cluster2 := newCluster("cluster2", "cluster2", clusterv1alpha1.ClusterOK, map[string]string{})
	clusters := []*clusterv1alpha1.Cluster{cluster1, cluster2}
	view := newView("view1", "view1")

	clusterToWorks := map[string][]*v1alpha1.Work{}
	clustersNeedingWorks, _, _ := manager.worksShouldBeOnClusters(view, clusterToWorks, clusters)

	if len(clustersNeedingWorks) != 2 {
		t.Errorf("getClustersToWorks() clustersNeedingWorks = %v, want %v", len(clustersNeedingWorks), 2)
	}

	work1 := newWork("work1", "cluster1", "cluster1", "view1", "view1")
	work1.Spec.Scope.ResourceType = "pods"
	work2 := newWork("work2", "cluster3", "cluster3", "view1", "view1")
	manager.workStore.Add(work1)
	manager.workStore.Add(work2)
	clusterToWorks, err := manager.getClustersToWorks(view)
	if err != nil {
		t.Errorf("Failed to get cluster to works %v", err)
	}

	if len(clusterToWorks) != 2 {
		t.Errorf("getClustersToWorks() = %v, want %v", len(clusterToWorks), 2)
	}

	clustersNeedingWorks, worksToDelete, workToUpdate := manager.worksShouldBeOnClusters(view, clusterToWorks, clusters)
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

func TestCreateViewCondition(t *testing.T) {
	view := newView("view1", "view1")
	conditions := createViewContidion(view, v1alpha1.WorkCompleted)
	if len(conditions) != 1 {
		t.Errorf("CreateViewCondition() conditions = %v, want %v", len(conditions), 1)
	}

	view.Status.Conditions = conditions
	conditions = createViewContidion(view, v1alpha1.WorkFailed)
	if len(conditions) != 2 {
		t.Errorf("CreateViewCondition() conditions = %v, want %v", len(conditions), 2)
	}

	view.Status.Conditions = conditions
	conditions = createViewContidion(view, v1alpha1.WorkFailed)
	if len(conditions) != 2 {
		t.Errorf("CreateViewCondition() conditions = %v, want %v", len(conditions), 2)
	}
}

func TestEnqueueViewFromWork(t *testing.T) {
	manager, _, _, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")

	manager.enqueueViewFromWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueViewFromWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}
}

func TestAddWork(t *testing.T) {
	manager, _, _, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")
	manager.addWork(work)
	if manager.workqueue.Len() != 0 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 0)
	}

	work.Status.Type = v1alpha1.WorkCompleted
	work.Spec.Type = v1alpha1.ResourceWorkType
	manager.addWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}
}

func TestUpdateWork(t *testing.T) {
	manager, _, _, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")
	manager.updateWork(work, work)
	if manager.workqueue.Len() != 0 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 0)
	}

	work.Spec.Type = v1alpha1.ResourceWorkType
	newwork := work.DeepCopy()
	newwork.Status.Type = v1alpha1.WorkCompleted
	manager.updateWork(work, newwork)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}

	work1 := newWork("work2", "work2", "cluster1", "view2", "view2")
	work1.Spec.Type = v1alpha1.ResourceWorkType
	work1.Status.Type = v1alpha1.WorkProcessing
	newwork1 := work1.DeepCopy()
	newwork1.Status.Type = v1alpha1.WorkProcessing
	manager.updateWork(work1, newwork1)
	if manager.workqueue.Len() != 2 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 2)
	}
}

func TestDeleteWork(t *testing.T) {
	manager, _, _, _ := newTestController()
	work := newWork("work1", "work1", "cluster1", "view1", "view1")
	work.Spec.Type = v1alpha1.ResourceWorkType
	manager.deleteWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}

	work.Status.Type = v1alpha1.WorkCompleted
	manager.addWork(work)
	if manager.workqueue.Len() != 1 {
		t.Errorf("EnqueueAddWork() queue len = %v, want %v", manager.workqueue.Len(), 1)
	}
}
