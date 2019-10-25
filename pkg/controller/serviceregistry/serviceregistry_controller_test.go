// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package serviceregistry

import (
	"testing"
	"time"

	clusterfake "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset/fake"
	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	clusterv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	cluster "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func newRegisterEndpoint(namespace, name, annotations string) *corev1.Endpoints {
	endpoint := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				clusterLabel:    namespace,
				"customerlabel": "customerlabel",
			},
			Annotations: map[string]string{
				discoveryAnnotationKey: annotations,
			},
		},
		Subsets: []corev1.EndpointSubset{
			corev1.EndpointSubset{
				Addresses: []corev1.EndpointAddress{
					corev1.EndpointAddress{
						IP: "1.2.3.4",
					},
				},
			},
		},
	}
	return endpoint
}

func newCluster(name string, ready bool) *cluster.Cluster {
	var clusterType cluster.ClusterConditionType
	if ready {
		clusterType = cluster.ClusterOK
	} else {
		clusterType = "Offline"
	}
	return &cluster.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: name,
		},
		Status: cluster.ClusterStatus{
			Conditions: []cluster.ClusterCondition{
				cluster.ClusterCondition{
					Type: clusterType,
				},
			},
		},
	}
}

func newController(clusterLister clusterv1alpha1.ClusterLister) *Controller {
	return &Controller{
		kubeclientset:      kubefake.NewSimpleClientset(),
		clusterLister:      clusterLister,
		registerEndpoints:  map[string]*endpointNode{},
		discoveryEndpoints: map[string]*endpointNode{},
	}
}

func newClusterLister() clusterv1alpha1.ClusterLister {
	cluster1 := newCluster("cluster1", true)
	cluster2 := newCluster("cluster2", true)
	cluster3 := newCluster("cluster3", true)
	cluster4 := newCluster("cluster4", false)
	clusterFakeClient := clusterfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterFakeClient, time.Minute*10)
	store := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters().Informer().GetStore()
	store.Add(cluster1)
	store.Add(cluster2)
	store.Add(cluster3)
	store.Add(cluster4)
	store.Add(&cluster.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "cluster5", Namespace: "cluster5"}})
	return clusterInformerFactory.Clusterregistry().V1alpha1().Clusters().Lister()
}

func TestRegisterServiceToAllClusters(t *testing.T) {
	c := newController(newClusterLister())
	ep := newRegisterEndpoint("test", "test", "{}")
	c.addNode(c.newEndpointNode(ep))
	toCreate, toUpdate, toDelete := c.syncEndpoints()
	if len(toCreate) != 3 {
		t.Fatalf("expect 3 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
}

func TestRegisterServiceToSpecifiedCluster(t *testing.T) {
	c := newController(newClusterLister())
	ep := newRegisterEndpoint("test", "test", "{\"target-clusters\": [\"cluster1\", \"cluster2\"]}")
	c.addNode(c.newEndpointNode(ep))
	toCreate, toUpdate, toDelete := c.syncEndpoints()
	if len(toCreate) != 2 {
		t.Fatalf("expect 2 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
}

func TestRegisterServiceToNotReadyCluster(t *testing.T) {
	c := newController(newClusterLister())
	ep := newRegisterEndpoint("test", "test", "{\"target-clusters\": [\"cluster4\", \"cluster5\", \"cluster6\"]}")
	c.addNode(c.newEndpointNode(ep))
	toCreate, toUpdate, toDelete := c.syncEndpoints()
	if len(toCreate) != 0 {
		t.Fatalf("expect 0 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
}

func TestDeleteRegisteredService(t *testing.T) {
	c := newController(newClusterLister())
	ep1 := newRegisterEndpoint("test1", "test1", "{}")
	ep2 := newRegisterEndpoint("test2", "test2", "{\"target-clusters\": [\"cluster1\", \"cluster2\"]}")
	c.addNode(c.newEndpointNode(ep1))
	c.addNode(c.newEndpointNode(ep2))
	toCreate, toUpdate, toDelete := c.syncEndpoints()
	if len(toCreate) != 5 {
		t.Fatalf("expect 5 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
	for _, ep := range toCreate {
		c.addNode(c.newEndpointNode(ep))
	}

	c.deleteNode(c.newEndpointNode(cache.DeletedFinalStateUnknown{Obj: ep2}))
	toCreate, toUpdate, toDelete = c.syncEndpoints()
	if len(toCreate) != 0 {
		t.Fatalf("expect 0 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 2 {
		t.Fatalf("expect 2 but %d", len(toDelete))
	}
}

func TestRecoverRegisteredService(t *testing.T) {
	c := newController(newClusterLister())
	ep1 := newRegisterEndpoint("test1", "test1", "{}")
	ep2 := newRegisterEndpoint("test2", "test2", "{\"target-clusters\": [\"cluster1\", \"cluster2\"]}")
	c.addNode(c.newEndpointNode(ep1))
	c.addNode(c.newEndpointNode(ep2))
	toCreate, toUpdate, toDelete := c.syncEndpoints()
	if len(toCreate) != 5 {
		t.Fatalf("expect 5 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
	for _, ep := range toCreate {
		c.addNode(c.newEndpointNode(ep))
	}

	for i := 0; i < 2; i++ {
		c.deleteNode(c.newEndpointNode(cache.DeletedFinalStateUnknown{Obj: toCreate[i]}))
	}
	toCreate, toUpdate, toDelete = c.syncEndpoints()
	if len(toCreate) != 2 {
		t.Fatalf("expect 2 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
}

func TestUpdateRegisteredService(t *testing.T) {
	c := newController(newClusterLister())
	ep1 := newRegisterEndpoint("test1", "test1", "{}")
	c.addNode(c.newEndpointNode(ep1))
	toCreate, toUpdate, toDelete := c.syncEndpoints()
	if len(toCreate) != 3 {
		t.Fatalf("expect 3 but %d", len(toCreate))
	}
	if len(toUpdate) != 0 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
	for _, ep := range toCreate {
		c.addNode(c.newEndpointNode(ep))
	}

	ep1.Subsets[0].Addresses[0].IP = "5.6.7.8"
	c.addNode(c.newEndpointNode(ep1))
	toCreate, toUpdate, toDelete = c.syncEndpoints()
	if len(toCreate) != 0 {
		t.Fatalf("expect 3 but %d", len(toCreate))
	}
	if len(toUpdate) != 3 {
		t.Fatalf("expect 0 but %d", len(toUpdate))
	}
	if len(toDelete) != 0 {
		t.Fatalf("expect 0 but %d", len(toDelete))
	}
}

func TestRemoveDiscoveryAnnotation(t *testing.T) {
	c := newController(newClusterLister())
	ep := newRegisterEndpoint("test", "test", "{}")
	epNode := c.newEndpointNode(ep)
	c.addNode(epNode)
	if epNode.onlyDeletion {
		t.Fatalf("expect false but true")
	}

	ep.Annotations = map[string]string{}
	newEpNode := c.newEndpointNode(ep)
	if !newEpNode.onlyDeletion {
		t.Fatalf("expect true but false")
	}
}

func TestRunContoller(t *testing.T) {
	var stopCh <-chan struct{}

	registerep1 := newRegisterEndpoint("cluster2", "kube-service.test.http1", "{}")
	fakeKubeClientset := kubefake.NewSimpleClientset(registerep1)
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(fakeKubeClientset, time.Minute*10)

	clusterFakeClient := clusterfake.NewSimpleClientset()
	clusterInformerFactory := clusterinformers.NewSharedInformerFactory(clusterFakeClient, time.Minute*10)
	store := clusterInformerFactory.Clusterregistry().V1alpha1().Clusters().Informer().GetStore()
	store.Add(newCluster("cluster1", true))

	ctrl := NewServiceRegistryController(fakeKubeClientset, kubeInformerFactory, clusterInformerFactory, stopCh)
	ctrl.endpointSynced = func() bool { return true }

	go kubeInformerFactory.Start(stopCh)

	ctrl.processGraphChanges()
	ctrl.syncup()
}
