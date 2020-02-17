// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package klusterlet

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/version"
	"k8s.io/klog"

	clusterclient "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/cluster_clientset_generated/clientset"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/api"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/mcm/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	informers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/informers_generated/externalversions"
	listers "github.com/open-cluster-management/multicloud-operators-foundation/pkg/client/listers_generated/mcm/v1beta1"
	resourceutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	equalutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/equals"
	restutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/rest"

	routev1 "github.com/openshift/client-go/route/clientset/versioned"
)

const (
	controllerAgentName            = "klusterlet"
	clusterStatusUpdateFrequency   = 30 * time.Second
	endpointOperatorDeploymentName = "ibm-multicluster-endpoint-operator"
)

var clusterControllerKind = mcm.SchemeGroupVersion.WithKind("Cluster")

type workHandlerFunc func(*v1beta1.Work) error

// Config is the klusterlet configuration definition
type Config struct {
	ClusterName            string
	ClusterNamespace       string
	MasterAddresses        string
	ServerVersion          string
	KlusterletAddress      string
	KlusterletIngress      string
	KlusterletRoute        string
	KlusterletService      string
	MonitoringScrapeTarget string
	Kubeconfig             *rest.Config
	ClusterLabels          map[string]string
	ClusterAnnotations     map[string]string
	KlusterletPort         int32
	EnableImpersonation    bool
}

// Klusterlet is the main struct for klusterlet server
type Klusterlet struct {
	// klusterlet configuration
	config *Config

	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	routeV1Client routev1.Interface
	kubeControl   restutils.KubeControlInterface

	// kubeclientset is a standard kubernetes dynamic clientset
	kubeDynamicClientset dynamic.Interface

	// hcmclientset is a clientset for our own API group
	hcmclientset clientset.Interface

	// hubkubeclientset is the client to connect to kube api of hub cluster
	hubkubeclientset kubernetes.Interface

	// clusterclientset is a clientset for cluster registry api
	clusterclientset clusterclient.Interface

	nodeLister v1listers.NodeLister
	podLister  v1listers.PodLister
	pvLister   v1listers.PersistentVolumeLister
	workLister listers.WorkLister

	nodeSynced cache.InformerSynced
	podSynced  cache.InformerSynced
	workSynced cache.InformerSynced

	handlers map[v1beta1.WorkType]workHandlerFunc
	server   *klusterletServer

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	stopCh    <-chan struct{}
	runServer chan v1beta1.ClusterStatus
}

// NewKlusterlet create a new klusterlet
func NewKlusterlet(
	config *Config,
	kubeclientset kubernetes.Interface,
	routeV1Client routev1.Interface,
	hcmclientset clientset.Interface,
	kubeDynamicClientset dynamic.Interface,
	hubkubeclientset kubernetes.Interface,
	clusterclientset clusterclient.Interface,
	kubeControl restutils.KubeControlInterface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	informerFactory informers.SharedInformerFactory,
	stopCh <-chan struct{},
) *Klusterlet {
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()
	podInformer := kubeInformerFactory.Core().V1().Pods()
	pvInformer := kubeInformerFactory.Core().V1().PersistentVolumes()

	workInformer := informerFactory.Mcm().V1beta1().Works()

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	if hubkubeclientset != nil {
		eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: hubkubeclientset.CoreV1().Events("")})
	}
	recorder := eventBroadcaster.NewRecorder(api.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Klusterlet{
		config:               config,
		kubeclientset:        kubeclientset,
		routeV1Client:        routeV1Client,
		kubeControl:          kubeControl,
		kubeDynamicClientset: kubeDynamicClientset,
		hcmclientset:         hcmclientset,
		hubkubeclientset:     hubkubeclientset,
		clusterclientset:     clusterclientset,
		nodeLister:           nodeInformer.Lister(),
		podLister:            podInformer.Lister(),
		workLister:           workInformer.Lister(),
		pvLister:             pvInformer.Lister(),
		nodeSynced:           nodeInformer.Informer().HasSynced,
		podSynced:            podInformer.Informer().HasSynced,
		workSynced:           workInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Klusterlet"),
		recorder:             recorder,
		stopCh:               stopCh,
		runServer:            make(chan v1beta1.ClusterStatus),
	}

	controller.handlers = map[v1beta1.WorkType]workHandlerFunc{
		v1beta1.ResourceWorkType: controller.handleResourceWork,
		v1beta1.ActionWorkType:   controller.handleActionWork,
	}

	klog.Info("Setting up event handlers")

	// Set up an event handler for when Foo resources change
	workInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWork,
		UpdateFunc: func(old, new interface{}) {
			oldwork := old.(*v1beta1.Work)
			newwork := new.(*v1beta1.Work)
			if controller.needsUpdate(oldwork, newwork) {
				controller.cleanWorkStatus(&newwork.Status)
				controller.enqueueWork(new)
			}
		},
	})

	return controller
}

// Run is the main run loop of kluster server
func (k *Klusterlet) Run(workers int) {
	defer utilruntime.HandleCrash()
	defer k.workqueue.ShutDown()

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for kubernetes informer caches to sync")
	if ok := cache.WaitForCacheSync(k.stopCh, k.nodeSynced, k.podSynced); !ok {
		klog.Errorf("failed to wait for kubernetes caches to sync")
		return
	}

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for hcm informer caches to sync")
	if ok := cache.WaitForCacheSync(k.stopCh, k.workSynced); !ok {
		klog.Errorf("failed to wait for hcm caches to sync")
		return
	}

	// Start syncing cluster status immediately, this may set up things the runtime needs to run.
	go wait.Until(k.syncClusterStatus, clusterStatusUpdateFrequency, wait.NeverStop)

	klog.Info("Starting workers")
	// Launch workers to process Work resources
	for i := 0; i < workers; i++ {
		go wait.Until(k.runWorker, time.Second, k.stopCh)
	}

	klog.Info("Started workers")
	<-k.stopCh
	klog.Info("Shutting down workers")
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (k *Klusterlet) runWorker() {
	for k.processNextWorkItem() {
	}
}

func (k *Klusterlet) processNextWorkItem() bool {
	obj, shutdown := k.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer k.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			k.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := k.processWork(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		k.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// processWork process work and update work result
func (k *Klusterlet) processWork(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	work, err := k.workLister.Works(namespace).Get(name)
	if err != nil {
		// The Work resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("work '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// Check if this cluster should process the work
	if work.Spec.Cluster.Name != k.config.ClusterName {
		klog.Infof("Work %s is not processed on cluster %s", key, k.config.ClusterName)
		return nil
	}

	if work.Status.Type == v1beta1.WorkCompleted {
		klog.V(3).Infof("Work %s is completed or failed, so skip it", key)
		return nil
	}

	if work.Status.Type == v1beta1.WorkFailed && work.Spec.Scope.Mode != v1beta1.PeriodicResourceUpdate {
		klog.V(3).Infof("Work %s is failed, so skip it", key)
		return nil
	}

	klog.Infof("Starting process work '%s'", key)
	if handler, ok := k.handlers[work.Spec.Type]; ok {
		err = handler(work)
		if err != nil {
			return err
		}

		if work.Spec.Scope.Mode == v1beta1.PeriodicResourceUpdate {
			k.workqueue.AddAfter(key, time.Duration(work.Spec.Scope.UpdateIntervalSeconds)*time.Second)
		}
	}

	return nil
}

func (k *Klusterlet) syncClusterStatus() {
	cluster, err := k.clusterclientset.ClusterregistryV1alpha1().Clusters(k.config.ClusterNamespace).Get(k.config.ClusterName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Failed to get cluster: %v", err)
		return
	}

	masterAddresses, masterPorts, clusterURL := k.getMasterAddresses()
	apiServerEndpoints := k.convertServerAddressToClusterEndpoint(masterAddresses, masterPorts)
	if err == nil && !reflect.DeepEqual(cluster.Spec.KubernetesAPIEndpoints, apiServerEndpoints) {
		cluster.Spec.KubernetesAPIEndpoints = apiServerEndpoints
		cluster, err = k.clusterclientset.ClusterregistryV1alpha1().Clusters(k.config.ClusterNamespace).Update(cluster)
		if err != nil {
			klog.Errorf("Failed to update cluster with new endpoints: %v", err)
			return
		}
	}

	if apierrors.IsNotFound(err) {
		cluster = &clusterv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:        k.config.ClusterName,
				Labels:      k.config.ClusterLabels,
				Annotations: k.config.ClusterAnnotations,
			},
		}

		cluster.Spec.KubernetesAPIEndpoints = apiServerEndpoints

		cluster, err = k.clusterclientset.ClusterregistryV1alpha1().Clusters(k.config.ClusterNamespace).Create(cluster)
		if err != nil {
			klog.Errorf("Failed to create cluster: %v", err)
			return
		}
	}

	if cluster.ObjectMeta.Labels == nil {
		cluster.ObjectMeta.Labels = make(map[string]string)
	}

	conditions := cluster.Status.Conditions
	if len(conditions) == 0 {
		condition := clusterv1alpha1.ClusterCondition{
			Type:              clusterv1alpha1.ClusterOK,
			LastHeartbeatTime: metav1.Now(),
		}
		conditions = append(conditions, condition)
	}

	for k, v := range k.config.ClusterLabels {
		if _, ok := cluster.Labels[k]; ok {
			if cluster.Labels[k] == "auto-detect" {
				cluster.Labels[k] = v
			}
			continue
		}
		cluster.Labels[k] = v
	}

	err = k.updateClusterStatus(cluster.Labels, masterAddresses, clusterURL)
	if err != nil {
		conditions[0].Type = ""
		conditions[0].LastTransitionTime = metav1.Now()
		conditions[0].Reason = err.Error()
	} else {
		conditions[0].LastHeartbeatTime = metav1.Now()
		conditions[0].Type = clusterv1alpha1.ClusterOK
		conditions[0].Reason = ""
	}
	cluster.Status.Conditions = conditions

	cluster, err = k.clusterclientset.ClusterregistryV1alpha1().Clusters(k.config.ClusterNamespace).UpdateStatus(cluster)
	if err != nil {
		klog.Errorf("Failed to update cluster status: %v", err)
		k.recorder.Event(cluster, corev1.EventTypeWarning, "Failed to update cluster status", err.Error())
	} else {
		k.recorder.Event(cluster, corev1.EventTypeNormal, "", "cluster status updated")
	}
}

func (k *Klusterlet) cleanWorkStatus(status *v1beta1.WorkStatus) {
	status.Type = ""
	status.Reason = ""
	status.Result = runtime.RawExtension{}
}

// getMasterAddresses get addresses of kubernetes master, it checks icp api config at first,
// then check the default kubernetes service.
func (k *Klusterlet) getMasterAddresses() ([]corev1.EndpointAddress, []corev1.EndpointPort, string) {
	//for 3.2.0
	masterAddresses, masterPorts, clusterURL, notEmpty, err := k.getMasterAddressesFromIBMcloudClusterInfo()
	if err == nil && notEmpty {
		return masterAddresses, masterPorts, clusterURL
	}

	//for 3.1.2
	masterAddresses, masterPorts, clusterURL, notEmpty, err = k.getMasterAddressesFromPlatformAPI()
	if err == nil && notEmpty {
		return masterAddresses, masterPorts, clusterURL
	}

	//for Openshift
	masterAddresses, masterPorts, clusterURL, notEmpty, err = k.getMasterAddressesFromConsoleConfig()
	if err == nil && notEmpty {
		return masterAddresses, masterPorts, clusterURL
	}

	if k.config.MasterAddresses == "" {
		kubeEndpoints, serviceErr := k.kubeclientset.CoreV1().Endpoints("default").Get("kubernetes", metav1.GetOptions{})
		if serviceErr == nil && len(kubeEndpoints.Subsets) > 0 {
			masterAddresses = kubeEndpoints.Subsets[0].Addresses
			masterPorts = kubeEndpoints.Subsets[0].Ports
		}
	} else {
		masterAddresses = append(masterAddresses, corev1.EndpointAddress{IP: k.config.MasterAddresses})
	}

	return masterAddresses, masterPorts, clusterURL
}

//for OpenShift, read endpoint address from console-config in openshift-console
func (k *Klusterlet) getMasterAddressesFromConsoleConfig() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, bool, error) {
	masterAddresses := []corev1.EndpointAddress{}
	masterPorts := []corev1.EndpointPort{}
	clusterURL := ""

	cfg, err := k.kubeclientset.CoreV1().ConfigMaps("openshift-console").Get("console-config", metav1.GetOptions{})
	if err == nil && cfg.Data != nil {
		consoleConfigString, ok := cfg.Data["console-config.yaml"]
		if ok {
			consoleConfigList := strings.Split(consoleConfigString, "\n")
			eu := ""
			cu := ""
			for _, configInfo := range consoleConfigList {
				parse := strings.Split(strings.Trim(configInfo, " "), ": ")
				if parse[0] == "masterPublicURL" {
					eu = strings.Trim(parse[1], " ")
				}
				if parse[0] == "consoleBaseAddress" {
					cu = strings.Trim(parse[1], " ")
				}
			}
			if eu != "" {
				euArray := strings.Split(strings.Trim(eu, "htps:/"), ":")
				if len(euArray) == 2 {
					masterAddresses = append(masterAddresses, corev1.EndpointAddress{IP: euArray[0]})
					klog.Info("masterAddresses" + euArray[0])
					port, _ := strconv.ParseInt(euArray[1], 10, 32)
					klog.Info("masterAddresses" + euArray[1])
					masterPorts = append(masterPorts, corev1.EndpointPort{Port: int32(port)})
				}
			}
			if cu != "" {
				clusterURL = cu
			}
		}
	}
	return masterAddresses, masterPorts, clusterURL, cfg != nil && cfg.Data != nil, err
}

//for ICP 3.1.2 and before, read endpoint address from platform-api in kube-system
func (k *Klusterlet) getMasterAddressesFromPlatformAPI() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, bool, error) {
	masterAddresses := []corev1.EndpointAddress{}
	masterPorts := []corev1.EndpointPort{}
	clusterURL := ""

	cfg, err := k.kubeclientset.CoreV1().ConfigMaps("kube-system").Get("platform-api", metav1.GetOptions{})
	if err == nil && cfg.Data != nil {
		eu, ok := cfg.Data["KUBERNETES_API_EXTERNAL_URL"]
		if ok {
			eu = strings.TrimPrefix(eu, "https://")
			euArray := strings.Split(eu, ":")
			if len(euArray) == 2 {
				masterAddresses = append(masterAddresses, corev1.EndpointAddress{IP: euArray[0]})
				port, _ := strconv.ParseInt(euArray[1], 10, 32)
				masterPorts = append(masterPorts, corev1.EndpointPort{Port: int32(port)})
			}
		}
		cu, ok := cfg.Data["CLUSTER_EXTERNAL_URL"]
		if ok {
			clusterURL = cu
		}
	}
	return masterAddresses, masterPorts, clusterURL, cfg != nil && cfg.Data != nil, err
}

//for ICP 3.2.0 and after, read endpoint address from ibmcloud-cluster-info in kube-public
func (k *Klusterlet) getMasterAddressesFromIBMcloudClusterInfo() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, bool, error) {
	masterAddresses := []corev1.EndpointAddress{}
	masterPorts := []corev1.EndpointPort{}
	clusterURL := ""

	cfg, err := k.kubeclientset.CoreV1().ConfigMaps("kube-public").Get("ibmcloud-cluster-info", metav1.GetOptions{})
	if err == nil && cfg.Data != nil {
		k8sAPIHost, ok := cfg.Data["cluster_kube_apiserver_host"]
		if ok {
			masterAddresses = append(masterAddresses, corev1.EndpointAddress{IP: k8sAPIHost})
		}

		k8sAPIPort, ok := cfg.Data["cluster_kube_apiserver_port"]
		if ok {
			port, err := strconv.ParseInt(k8sAPIPort, 10, 32)
			if err != nil {
				klog.Error(err)
			} else {
				masterPorts = append(masterPorts, corev1.EndpointPort{Port: int32(port)})
			}
		}

		clusterAddress, ok1 := cfg.Data["cluster_address"]
		clusterPort, ok2 := cfg.Data["cluster_router_https_port"]
		if ok1 && ok2 {
			clusterURL = "https://" + clusterAddress + ":" + clusterPort
		}
	}
	return masterAddresses, masterPorts, clusterURL, cfg != nil && cfg.Data != nil, err
}

func (k *Klusterlet) convertServerAddressToClusterEndpoint(
	masterAddresses []corev1.EndpointAddress,
	masterPorts []corev1.EndpointPort,
) clusterv1alpha1.KubernetesAPIEndpoints {
	apiEndpoints := clusterv1alpha1.KubernetesAPIEndpoints{
		ServerEndpoints: []clusterv1alpha1.ServerAddressByClientCIDR{},
	}
	for _, addr := range masterAddresses {
		var serverAddress string
		if addr.Hostname != "" {
			serverAddress = addr.Hostname
		} else if addr.IP != "" {
			serverAddress = addr.IP
		}

		if len(masterPorts) == 0 {
			endpoint := clusterv1alpha1.ServerAddressByClientCIDR{
				ClientCIDR:    "0.0.0.0/0",
				ServerAddress: serverAddress,
			}
			apiEndpoints.ServerEndpoints = append(apiEndpoints.ServerEndpoints, endpoint)
		}

		for _, port := range masterPorts {
			endpoint := clusterv1alpha1.ServerAddressByClientCIDR{
				ClientCIDR:    "0.0.0.0/0",
				ServerAddress: serverAddress + ":" + strconv.Itoa(int(port.Port)),
			}
			apiEndpoints.ServerEndpoints = append(apiEndpoints.ServerEndpoints, endpoint)
		}
	}

	return apiEndpoints
}

func (k *Klusterlet) updateClusterStatus(
	clusterLabels map[string]string,
	masterAddresses []corev1.EndpointAddress,
	clusterURL string,
) error {
	cluster, err := k.clusterclientset.ClusterregistryV1alpha1().Clusters(k.config.ClusterNamespace).Get(k.config.ClusterName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Failed to get cluster: %v", err)
		return err
	}

	clusterStatus, err := k.hcmclientset.McmV1beta1().ClusterStatuses(k.config.ClusterNamespace).Get(k.config.ClusterName, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Failed to get cluster status: %v", err)
		return err
	}
	if apierrors.IsNotFound(err) {
		clusterStatus = &v1beta1.ClusterStatus{
			ObjectMeta: metav1.ObjectMeta{
				Name:            k.config.ClusterName,
				Namespace:       k.config.ClusterNamespace,
				Labels:          k.config.ClusterLabels,
				Annotations:     k.config.ClusterAnnotations,
				OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cluster, clusterControllerKind)},
			},
			Spec: v1beta1.ClusterStatusSpec{MonitoringScrapeTarget: k.config.MonitoringScrapeTarget},
		}
		clusterStatus, err = k.hcmclientset.McmV1beta1().ClusterStatuses(k.config.ClusterNamespace).Create(clusterStatus)
		if err != nil {
			klog.Errorf("Failed to create cluster: %v", err)
			return err
		}
	}

	clusterStatus.Labels = clusterLabels

	nodes, err := k.nodeLister.List(labels.Everything())
	if err != nil {
		return err
	}

	pods, err := k.podLister.List(labels.Everything())
	if err != nil {
		return err
	}

	pvs, err := k.pvLister.List(labels.Everything())
	if err != nil {
		return err
	}

	cpuCapacity, memoryCapacity := resourceutils.GetCPUAndMemoryCapacity(nodes)
	storageCapacity, storageAllocation := resourceutils.GetStorageCapacityAndAllocation(pvs)
	cpuAllocation, memoryAllocation := resourceutils.GetCPUAndMemoryAllocation(pods)

	var shouldUpdateStatus bool
	capacity := corev1.ResourceList{
		corev1.ResourceCPU:     cpuCapacity,
		corev1.ResourceMemory:  resourceutils.FormatQuatityToMi(memoryCapacity),
		corev1.ResourceStorage: resourceutils.FormatQuatityToGi(storageCapacity),
		v1beta1.ResourceNodes:  *resource.NewQuantity(int64(len(nodes)), resource.DecimalSI),
	}
	if !equalutils.EqualResourceList(clusterStatus.Spec.Capacity, capacity) {
		clusterStatus.Spec.Capacity = capacity
		shouldUpdateStatus = true
	}

	usage := corev1.ResourceList{
		v1beta1.ResourcePods:   *resource.NewQuantity(int64(len(pods)), resource.DecimalSI),
		corev1.ResourceCPU:     cpuAllocation,
		corev1.ResourceMemory:  resourceutils.FormatQuatityToMi(memoryAllocation),
		corev1.ResourceStorage: resourceutils.FormatQuatityToGi(storageAllocation),
	}
	if !equalutils.EqualResourceList(clusterStatus.Spec.Usage, usage) {
		clusterStatus.Spec.Usage = usage
		shouldUpdateStatus = true
	}

	// Config endpoint
	if !equalutils.EqualEndpointAddresses(masterAddresses, clusterStatus.Spec.MasterAddresses) {
		clusterStatus.Spec.MasterAddresses = masterAddresses
		shouldUpdateStatus = true
	}

	// Config console url
	if clusterStatus.Spec.ConsoleURL != clusterURL {
		clusterStatus.Spec.ConsoleURL = clusterURL
		shouldUpdateStatus = true
	}

	clusterStatus.Spec.KlusterletVersion = version.Get().GitVersion
	clusterStatus.Spec.Version = k.config.ServerVersion
	clusterStatus.Spec.EndpointOperatorVersion = k.getEndponitOperatorVersion()

	// Config klusterlet endpoint
	klusterletEndpoint, klusterletPort, err := k.readKlusterletConfig()
	if err == nil {
		if !reflect.DeepEqual(klusterletEndpoint, &clusterStatus.Spec.KlusterletEndpoint) {
			clusterStatus.Spec.KlusterletEndpoint = *klusterletEndpoint
			shouldUpdateStatus = true
		}
		if !reflect.DeepEqual(klusterletPort, &clusterStatus.Spec.KlusterletPort) {
			clusterStatus.Spec.KlusterletPort = *klusterletPort
			shouldUpdateStatus = true
		}
	} else {
		klog.Errorf("Failed to get klusterlet server config: %v", err)
	}

	if clusterStatus.Spec.MonitoringScrapeTarget != k.config.MonitoringScrapeTarget {
		clusterStatus.Spec.MonitoringScrapeTarget = k.config.MonitoringScrapeTarget
		shouldUpdateStatus = true
	}

	if shouldUpdateStatus {
		clusterStatus, err = k.hcmclientset.McmV1beta1().ClusterStatuses(k.config.ClusterNamespace).Update(clusterStatus)
		if err != nil {
			k.recorder.Event(clusterStatus, corev1.EventTypeWarning, "Failed to update cluster status", err.Error())
			return err
		}
		k.recorder.Event(clusterStatus, corev1.EventTypeNormal, "", "cluster status updated")
	}

	// Finally signal klusterlet server to start
	select {
	case k.runServer <- *clusterStatus:
		klog.Info("Signal klusterlet server to start")
	default:
	}

	k.refreshServerIfNeeded(clusterStatus)

	return nil
}

func (k *Klusterlet) updateFailedStatus(work *v1beta1.Work, err error) error {
	if err == nil {
		return nil
	}

	work.Status.LastUpdateTime = metav1.Now()
	work.Status.Type = v1beta1.WorkFailed
	work.Status.Reason = err.Error()
	_, updateErr := k.hcmclientset.McmV1beta1().Works(k.config.ClusterNamespace).UpdateStatus(work)
	return updateErr
}

func (k *Klusterlet) readKlusterletConfig() (*corev1.EndpointAddress, *corev1.EndpointPort, error) {
	endpoint := &corev1.EndpointAddress{}
	port := &corev1.EndpointPort{
		Name:     "https",
		Protocol: corev1.ProtocolTCP,
		Port:     k.config.KlusterletPort,
	}
	// set endpoint by user flag at first, if it is not set, use IP in ingress status
	ip := net.ParseIP(k.config.KlusterletAddress)
	if ip != nil {
		endpoint.IP = k.config.KlusterletAddress
	} else {
		endpoint.Hostname = k.config.KlusterletAddress
	}

	if k.config.KlusterletIngress != "" {
		if err := k.setEndpointAddressFromIngress(endpoint); err != nil {
			return nil, nil, err
		}
	}

	// Only use klusterlet service when neither ingress nor address is set
	if k.config.KlusterletService != "" && endpoint.IP == "" && endpoint.Hostname == "" {
		if err := k.setEndpointAddressFromService(endpoint, port); err != nil {
			return nil, nil, err
		}
	}

	if k.config.KlusterletRoute != "" && endpoint.IP == "" && endpoint.Hostname == "" {
		if err := k.setEndpointAddressFromRoute(endpoint); err != nil {
			return nil, nil, err
		}
	}

	return endpoint, port, nil
}

func (k *Klusterlet) setEndpointAddressFromIngress(endpoint *corev1.EndpointAddress) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(k.config.KlusterletIngress)
	if err != nil {
		klog.Warningf("Failed do parse ingress resource: %v", err)
		return err
	}
	klIngress, err := k.kubeclientset.ExtensionsV1beta1().Ingresses(klNamespace).Get(klName, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Failed do get ingress resource: %v", err)
		return err
	}

	if endpoint.IP == "" && len(klIngress.Status.LoadBalancer.Ingress) > 0 {
		endpoint.IP = klIngress.Status.LoadBalancer.Ingress[0].IP
	}

	if endpoint.Hostname == "" && len(klIngress.Spec.Rules) > 0 {
		endpoint.Hostname = klIngress.Spec.Rules[0].Host
	}
	return nil
}

func (k *Klusterlet) setEndpointAddressFromService(endpoint *corev1.EndpointAddress, port *corev1.EndpointPort) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(k.config.KlusterletService)
	if err != nil {
		return err
	}

	klSvc, err := k.kubeclientset.CoreV1().Services(klNamespace).Get(klName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if klSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return fmt.Errorf("klusterlet service should be in type of loadbalancer")
	}

	if len(klSvc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf("klusterlet load balancer service does not have valid ip address")
	}

	if len(klSvc.Spec.Ports) == 0 {
		return fmt.Errorf("klusterlet load balancer service does not have valid port")
	}

	// Only get the first IP and port
	endpoint.IP = klSvc.Status.LoadBalancer.Ingress[0].IP
	endpoint.Hostname = klSvc.Status.LoadBalancer.Ingress[0].Hostname
	port.Port = klSvc.Spec.Ports[0].Port
	return nil
}

func (k *Klusterlet) setEndpointAddressFromRoute(endpoint *corev1.EndpointAddress) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(k.config.KlusterletRoute)
	if err != nil {
		klog.Warningf("Route name input error: %v", err)
		return err
	}

	route, err := k.routeV1Client.RouteV1().Routes(klNamespace).Get(klName, metav1.GetOptions{})

	if err != nil {
		klog.Warningf("Failed to get the route: %v", err)
		return err
	}

	endpoint.Hostname = route.Spec.Host
	return nil
}

// enqueueWork takes a Work resource and converts it into a name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Work.
func (k *Klusterlet) enqueueWork(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	k.workqueue.AddRateLimited(key)
}

func (k *Klusterlet) needsUpdate(oldwork, newwork *v1beta1.Work) bool {
	return !equalutils.EqualWorkSpec(&oldwork.Spec, &newwork.Spec)
}

// getEndponitOperatorVersion gets the Endpoint Operator Version
func (k *Klusterlet) getEndponitOperatorVersion() string {
	klNamespace, _, err := cache.SplitMetaNamespaceKey(k.config.KlusterletRoute)
	if err != nil {
		klog.Warningf("Failed do parse ingress resource: %v", err)
		return ""
	}

	deployment, err := k.kubeclientset.AppsV1beta1().Deployments(klNamespace).Get(endpointOperatorDeploymentName, v1.GetOptions{})
	if err != nil {
		klog.Info("Failed to get the Endpoint Operator Version: ", err)
		return ""
	}

	return strings.Split(deployment.Spec.Template.Spec.Containers[0].Image, ":")[1]
}
