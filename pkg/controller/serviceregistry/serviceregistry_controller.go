// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package serviceregistry

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	clusterinformers "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_informers_generated/externalversions"
	clusterv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/cluster_listers_generated/clusterregistry/v1alpha1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
)

const (
	discoveryLabel         = "mcm.ibm.com/auto-discovery"
	clusterLabel           = "mcm.ibm.com/cluster"
	ownersLabel            = "mcm.ibm.com/owners"
	discoveryAnnotationKey = "mcm.ibm.com/service-discovery"
)

// Controller is a type for service registry controller
type Controller struct {
	kubeclientset      kubernetes.Interface
	endpointSynced     cache.InformerSynced
	clusterLister      clusterv1alpha1.ClusterLister
	workqueue          workqueue.RateLimitingInterface
	stopCh             <-chan struct{}
	mu                 sync.RWMutex
	registerEndpoints  map[string]*endpointNode
	discoveryEndpoints map[string]*endpointNode
}

// NewServiceRegistryController create a new Controller
func NewServiceRegistryController(
	kubeclientset kubernetes.Interface,
	kubeinformerFactory kubeinformers.SharedInformerFactory,
	clusterInformerFactory clusterinformers.SharedInformerFactory,
	stopCh <-chan struct{}) *Controller {
	endpointInformer := kubeinformerFactory.Core().V1().Endpoints()
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "serviceregistryQueue")

	ctrl := &Controller{
		kubeclientset:      kubeclientset,
		endpointSynced:     endpointInformer.Informer().HasSynced,
		clusterLister:      clusterInformerFactory.Clusterregistry().V1alpha1().Clusters().Lister(),
		workqueue:          queue,
		stopCh:             stopCh,
		registerEndpoints:  map[string]*endpointNode{},
		discoveryEndpoints: map[string]*endpointNode{},
	}

	endpointInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.handleFunc(ctrl.newEndpointNode, addEvent),
		UpdateFunc: func(old, new interface{}) {
			oldep := old.(*corev1.Endpoints)
			newep := new.(*corev1.Endpoints)
			if !reflect.DeepEqual(oldep.Subsets, newep.Subsets) || !reflect.DeepEqual(oldep.Annotations, newep.Annotations) {
				ctrl.handleFunc(ctrl.newEndpointNode, updateEvent)(new)
			}
		},
		DeleteFunc: ctrl.handleFunc(ctrl.newEndpointNode, deleteEvent),
	})

	return ctrl
}

type eventType int

const (
	addEvent eventType = iota
	updateEvent
	deleteEvent
)

type nodeEvent struct {
	n     *endpointNode
	event eventType
}

func (sr *Controller) handleFunc(fn func(interface{}) *endpointNode, t eventType) func(interface{}) {
	return func(obj interface{}) {
		n := fn(obj)
		if n != nil {
			sr.enqueueNode(n, t)
		}
	}
}

func (sr *Controller) enqueueNode(n *endpointNode, t eventType) {
	sr.workqueue.Add(&nodeEvent{n: n, event: t})
}

func (sr *Controller) newEndpointNode(new interface{}) *endpointNode {
	endpoint, ok := new.(*corev1.Endpoints)
	if !ok {
		deletedState, ok := new.(cache.DeletedFinalStateUnknown)
		if !ok {
			return nil
		}

		endpoint, ok = deletedState.Obj.(*corev1.Endpoints)
		if !ok {
			return nil
		}
	}
	isRegisterd, targetClusters := isRegisteredEndpoint(endpoint)
	if isRegisterd {
		return &endpointNode{
			name:               endpoint.Namespace + "/" + endpoint.Name,
			endpoint:           endpoint,
			isRegisterd:        true,
			onlyDeletion:       false,
			targetClusters:     targetClusters,
			discoveryEndpoints: map[string]bool{},
		}
	}

	if isDiscoveryEndpoint(endpoint) {
		return &endpointNode{
			name:               endpoint.Namespace + "/" + endpoint.Name,
			endpoint:           endpoint.DeepCopy(),
			isRegisterd:        false,
			onlyDeletion:       false,
			targetClusters:     []string{},
			discoveryEndpoints: map[string]bool{},
		}
	}

	// annotation is removed
	registerdNode, existed := sr.registerEndpoints[endpoint.Namespace+"/"+endpoint.Name]
	if existed {
		registerdNode.onlyDeletion = true
		return registerdNode
	}

	return nil
}

// Run is the main run loop of kluster server
func (sr *Controller) Run() error {
	defer utilruntime.HandleCrash()
	defer sr.workqueue.ShutDown()

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for hcm informer caches to sync")
	if ok := cache.WaitForCacheSync(sr.stopCh, sr.endpointSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	go wait.Until(sr.runWorker, time.Second, sr.stopCh)
	go wait.Until(sr.syncup, time.Second, sr.stopCh)

	<-sr.stopCh
	klog.Info("Shutting controller")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (sr *Controller) runWorker() {
	for sr.processGraphChanges() {
	}
}

func (sr *Controller) processGraphChanges() bool {
	item, quit := sr.workqueue.Get()
	if quit {
		return false
	}
	defer sr.workqueue.Done(item)

	ne, ok := item.(*nodeEvent)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("expect a *event, got %v", item))
		return true
	}

	sr.mu.Lock()
	switch ne.event {
	case addEvent:
		sr.addNode(ne.n)
	case deleteEvent:
		sr.deleteNode(ne.n)
	case updateEvent:
		sr.deleteNode(ne.n)
		if !ne.n.onlyDeletion {
			sr.addNode(ne.n)
		}
	}
	sr.mu.Unlock()
	return true
}

func (sr *Controller) addNode(n *endpointNode) {
	if n.isRegisterd {
		sr.registerEndpoints[n.name] = n
	} else {
		sr.discoveryEndpoints[n.name] = n
	}
}

func (sr *Controller) deleteNode(n *endpointNode) {
	if n.isRegisterd {
		delete(sr.registerEndpoints, n.name)
	} else {
		delete(sr.discoveryEndpoints, n.name)
	}
}

func (sr *Controller) syncup() {
	sr.mu.RLock()
	toCreate, toUpdate, toDelete := sr.syncEndpoints()
	sr.mu.RUnlock()

	var wg sync.WaitGroup
	for _, ep := range toCreate {
		wg.Add(1)
		go func(ep *corev1.Endpoints) {
			defer wg.Done()
			_, err := sr.kubeclientset.CoreV1().Endpoints(ep.Namespace).Create(ep)
			if err != nil {
				klog.Errorf("failed to create ep %s, %v", ep.Name, err)
			}
		}(ep)
	}

	for _, ep := range toUpdate {
		wg.Add(1)
		go func(ep *corev1.Endpoints) {
			defer wg.Done()
			_, err := sr.kubeclientset.CoreV1().Endpoints(ep.Namespace).Update(ep)
			if err != nil {
				klog.Errorf("failed to update ep %s, %v", ep.Name, err)
			}
		}(ep)
	}

	for _, ep := range toDelete {
		wg.Add(1)
		go func(ep *corev1.Endpoints) {
			defer wg.Done()
			err := sr.kubeclientset.CoreV1().Endpoints(ep.Namespace).Delete(ep.Name, &metav1.DeleteOptions{})
			if err != nil {
				klog.Errorf("failed to delete ep %s, %v", ep.Name, err)
			}
		}(ep)
	}
	wg.Wait()
}

func (sr *Controller) syncEndpoints() (toCreate, toUpdate, toDelete []*corev1.Endpoints) {
	for _, registerNode := range sr.registerEndpoints {
		for _, targetClusterNamespace := range sr.findTargetClusterNamespaces(registerNode.targetClusters) {
			discoverEndpoint := registerNode.toDiscoverEndpoint(targetClusterNamespace)
			old, ok := sr.discoveryEndpoints[fmt.Sprintf("%s/%s", targetClusterNamespace, discoverEndpoint.Name)]
			if ok {
				if !reflect.DeepEqual(old.endpoint.Subsets, discoverEndpoint.Subsets) {
					toUpdate = append(toUpdate, discoverEndpoint)
				}
			} else {
				toCreate = append(toCreate, discoverEndpoint)
			}
		}
	}
	for _, discoveryNode := range sr.discoveryEndpoints {
		owner, ok := discoveryNode.endpoint.Annotations[ownersLabel]
		if !ok {
			continue
		}
		_, ok = sr.registerEndpoints[owner]
		if !ok {
			toDelete = append(toDelete, discoveryNode.endpoint)
		}
	}
	return toCreate, toUpdate, toDelete
}

func (sr *Controller) findTargetClusterNamespaces(targetClusters []string) []string {
	namespaces := []string{}
	clusters, err := sr.clusterLister.List(labels.Everything())
	if err != nil {
		return []string{}
	}
	readyClutsters := map[string]string{}
	for _, cluster := range clusters {
		if len(cluster.Status.Conditions) == 0 {
			continue
		}
		if cluster.Status.Conditions[len(cluster.Status.Conditions)-1].Type != clusterregistry.ClusterOK {
			continue
		}
		readyClutsters[cluster.Name] = cluster.Namespace
	}
	if len(targetClusters) == 0 {
		for _, clusterNamespace := range readyClutsters {
			namespaces = append(namespaces, clusterNamespace)
		}
		return namespaces
	}

	for _, clusterName := range targetClusters {
		clusterNamespace, ok := readyClutsters[clusterName]
		if !ok {
			continue
		}
		namespaces = append(namespaces, clusterNamespace)
	}
	return namespaces
}

// endpoint node
type endpointNode struct {
	name               string
	endpoint           *corev1.Endpoints
	isRegisterd        bool
	onlyDeletion       bool
	targetClusters     []string
	discoveryEndpoints map[string]bool
}

func (n *endpointNode) toDiscoverEndpoint(namespace string) *corev1.Endpoints {
	sourceCluster := n.endpoint.Labels[clusterLabel]
	endpointName := fmt.Sprintf("%s.%s", sourceCluster, n.endpoint.Name)

	labels := map[string]string{}
	labels[discoveryLabel] = "true"
	for key, val := range n.endpoint.Labels {
		if key != clusterLabel {
			labels[key] = val
		}
	}

	annotation := map[string]string{}
	annotation[ownersLabel] = fmt.Sprintf("%s/%s", n.endpoint.Namespace, n.endpoint.Name)
	for key, val := range n.endpoint.Annotations {
		annotation[key] = val
	}

	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        endpointName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotation,
		},
		Subsets: n.endpoint.Subsets,
	}
}

type serviceDiscovery struct {
	DNSPrefix      string   `json:"dns-prefix,omitempty"`
	TargetClusters []string `json:"target-clusters,omitempty"`
}

func isRegisteredEndpoint(endpoint *corev1.Endpoints) (bool, []string) {
	if _, existed := endpoint.Labels[discoveryLabel]; existed {
		return false, nil
	}

	annotation, existed := endpoint.Annotations[discoveryAnnotationKey]
	if !existed {
		return false, nil
	}

	serviceDiscovery := &serviceDiscovery{
		TargetClusters: []string{},
	}
	if err := json.Unmarshal([]byte(annotation), serviceDiscovery); err != nil {
		klog.Errorf("endpoint %s/%s has a bad service discovery annotation, %v", endpoint.Namespace, endpoint.Name, err)
		return false, nil
	}

	return true, serviceDiscovery.TargetClusters
}

func isDiscoveryEndpoint(endpoint *corev1.Endpoints) bool {
	if _, exist := endpoint.Labels[discoveryLabel]; !exist {
		return false
	}
	return true
}
