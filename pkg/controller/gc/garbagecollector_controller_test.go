// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package gc

import (
	"testing"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	hcmfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset/fake"
	clientfake "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	hcminformers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"
)

var cluster1 = &clusterregistry.Cluster{
	TypeMeta: v1.TypeMeta{},
	ObjectMeta: v1.ObjectMeta{
		Name: "cluster1",
	},
	Spec:   clusterregistry.ClusterSpec{},
	Status: clusterregistry.ClusterStatus{},
}

var work1 = &mcm.Work{
	TypeMeta: v1.TypeMeta{},
	ObjectMeta: v1.ObjectMeta{
		Name: "work1",
		Annotations: map[string]string{
			mcm.OwnersLabel: "works.v1beta1.mcm.ibm.com",
		},
	},
}

var resourceView1 = &mcm.ResourceView{
	TypeMeta: v1.TypeMeta{},
	ObjectMeta: v1.ObjectMeta{
		Name: "resourceView1",
		Annotations: map[string]string{
			mcm.OwnersLabel: "resourceviews.v1beta1.mcm.ibm.com",
		},
	},
	Spec:   mcm.ResourceViewSpec{},
	Status: mcm.ResourceViewStatus{},
}

var workSet1 = &mcm.WorkSet{
	TypeMeta: v1.TypeMeta{},
	ObjectMeta: v1.ObjectMeta{
		Name: "workSet1",
		Annotations: map[string]string{
			mcm.OwnersLabel: "worksets.v1beta1.mcm.ibm.com",
		},
	},
	Spec:   mcm.WorkSetSpec{},
	Status: mcm.WorkSetStatus{},
}

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

func newGarbageCollectorController() *GarbageCollectorController {
	scheme := runtime.NewScheme()
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme,
		newUnstructured("group/version", "TheKind", "ns-foo", "name-foo"))

	clusterClient := clientfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterClient, 10*time.Minute)

	hcmClient := hcmfake.NewSimpleClientset()
	informerFactory := hcminformers.NewSharedInformerFactory(hcmClient, time.Minute*10)

	informers := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters()
	informers.Informer().GetIndexer().Add(cluster1)

	return NewGarbageCollectorController(
		dynamicClient, clusterInformerFactory, informerFactory, 60*time.Second, make(chan struct{}))
}

func TestGarbageCollect(t *testing.T) {
	g := newGarbageCollectorController()

	for _, gvr := range watchedResources {
		_, store, err := g.controllerFor(gvr)
		if err != nil {
			klog.Errorf("Failed to monitor: %v", err)
			continue
		}

		if gvr.Resource == "works" {
			store.Add(work1)
		}
		if gvr.Resource == "resourceviews" {
			store.Add(resourceView1)
		}
		if gvr.Resource == "worksets" {
			store.Add(workSet1)
		}
		g.storeMap[gvr] = store
	}

	g.syncResource()
	g.cleanExpiredObject()
}
