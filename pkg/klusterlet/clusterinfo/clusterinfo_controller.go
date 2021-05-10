package controllers

import (
	"context"
	"fmt"
	"net"

	clusterv1alpha1informer "github.com/open-cluster-management/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1lister "github.com/open-cluster-management/api/client/cluster/listers/cluster/v1alpha1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"

	"github.com/go-logr/logr"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ClusterInfoReconciler reconciles a ManagedClusterInfo object
type ClusterInfoReconciler struct {
	client.Client
	Log                         logr.Logger
	Scheme                      *runtime.Scheme
	KubeClient                  kubernetes.Interface
	NodeInformer                coreinformers.NodeInformer
	NodeLister                  corev1lister.NodeLister
	ClaimInformer               clusterv1alpha1informer.ClusterClaimInformer
	ClaimLister                 clusterv1alpha1lister.ClusterClaimLister
	ManagedClusterDynamicClient dynamic.Interface
	RouteV1Client               routev1.Interface
	ClusterName                 string
	MasterAddresses             string
	AgentAddress                string
	AgentIngress                string
	AgentRoute                  string
	AgentService                string
	AgentPort                   int32
	Agent                       *agent.Agent
}

func (r *ClusterInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ManagedClusterInfo", req.NamespacedName)

	request := types.NamespacedName{Namespace: r.ClusterName, Name: r.ClusterName}
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err := r.Get(ctx, request, clusterInfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update cluster info status here.
	newStatus := clusterInfo.DeepCopy().Status
	var errs []error

	// Config Agent endpoint
	agentEndpoint, agentPort, err := r.readAgentConfig()
	if err != nil {
		log.Error(err, "Failed to get agent server config")
		errs = append(errs, fmt.Errorf("failed to get agent server config, error:%v ", err))
	} else {
		newStatus.LoggingEndpoint = *agentEndpoint
		newStatus.LoggingPort = *agentPort
	}

	// Get distribution info
	newStatus.DistributionInfo, err = r.getDistributionInfo()
	if err != nil {
		log.Error(err, "Failed to get distribution info")
		errs = append(errs, fmt.Errorf("failed to get distribution info, error:%v ", err))
	}

	claims, err := r.ClaimLister.List(labels.Everything())
	if err != nil {
		log.Error(err, "Failed to list claims.")
		errs = append(errs, fmt.Errorf("failed to list clusterClaims error:%v ", err))
	}
	for _, claim := range claims {
		value := claim.Spec.Value
		switch claim.Name {
		case clusterclaim.ClaimOCMConsoleURL:
			newStatus.ConsoleURL = value
		case clusterclaim.ClaimOCMKubeVersion:
			newStatus.Version = value
		case clusterclaim.ClaimOpenshiftID:
			newStatus.ClusterID = value
		case clusterclaim.ClaimOCMProduct:
			newStatus.KubeVendor = r.getKubeVendor(value)
		case clusterclaim.ClaimOCMPlatform:
			newStatus.CloudVendor = r.getCloudVendor(value)
		case clusterclaim.ClaimOpenshiftVersion:
			if newStatus.DistributionInfo.Type == clusterv1beta1.DistributionTypeOCP {
				newStatus.DistributionInfo.OCP.Version = value
			}
		}
	}

	newSyncedCondition := metav1.Condition{
		Type:    clusterv1beta1.ManagedClusterInfoSynced,
		Status:  metav1.ConditionTrue,
		Reason:  clusterv1beta1.ReasonManagedClusterInfoSynced,
		Message: "Managed cluster info is synced",
	}
	if len(errs) > 0 {
		newSyncedCondition = metav1.Condition{
			Type:    clusterv1beta1.ManagedClusterInfoSynced,
			Status:  metav1.ConditionFalse,
			Reason:  clusterv1beta1.ReasonManagedClusterInfoSyncedFailed,
			Message: errors.NewAggregate(errs).Error(),
		}
	}

	meta.SetStatusCondition(&newStatus.Conditions, newSyncedCondition)

	if equality.Semantic.DeepEqual(newStatus, clusterInfo.Status) {
		return ctrl.Result{}, nil
	}

	clusterInfo.Status = newStatus
	err = r.Client.Status().Update(ctx, clusterInfo)
	if err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	r.RefreshAgentServer(clusterInfo)

	return ctrl.Result{}, nil
}

func (r *ClusterInfoReconciler) readAgentConfig() (*corev1.EndpointAddress, *corev1.EndpointPort, error) {
	endpoint := &corev1.EndpointAddress{}
	port := &corev1.EndpointPort{
		Name:     "https",
		Protocol: corev1.ProtocolTCP,
		Port:     r.AgentPort,
	}
	// set endpoint by user flag at first, if it is not set, use IP in ingress status
	ip := net.ParseIP(r.AgentAddress)
	if ip != nil {
		endpoint.IP = r.AgentAddress
	} else {
		endpoint.Hostname = r.AgentAddress
	}

	if r.AgentIngress != "" {
		if err := r.setEndpointAddressFromIngress(endpoint); err != nil {
			return nil, nil, err
		}
	}

	// Only use Agent service when neither ingress nor address is set
	if r.AgentService != "" && endpoint.IP == "" && endpoint.Hostname == "" {
		if err := r.setEndpointAddressFromService(endpoint, port); err != nil {
			return nil, nil, err
		}
	}

	if r.AgentRoute != "" && endpoint.IP == "" && endpoint.Hostname == "" {
		if err := r.setEndpointAddressFromRoute(endpoint); err != nil {
			return nil, nil, err
		}
	}

	return endpoint, port, nil
}

func (r *ClusterInfoReconciler) setEndpointAddressFromIngress(endpoint *corev1.EndpointAddress) error {
	log := r.Log.WithName("SetEndpoint")
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(r.AgentIngress)
	if err != nil {
		log.Error(err, "Failed do parse ingress resource:")
		return err
	}
	klIngress, err := r.KubeClient.ExtensionsV1beta1().Ingresses(klNamespace).Get(context.TODO(), klName, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "Failed do get ingress resource: %v")
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

func (r *ClusterInfoReconciler) setEndpointAddressFromService(endpoint *corev1.EndpointAddress, port *corev1.EndpointPort) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(r.AgentService)
	if err != nil {
		return err
	}

	klSvc, err := r.KubeClient.CoreV1().Services(klNamespace).Get(context.TODO(), klName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if klSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return fmt.Errorf("agent service should be in type of loadbalancer")
	}

	if len(klSvc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf("agent load balancer service does not have valid ip address")
	}

	if len(klSvc.Spec.Ports) == 0 {
		return fmt.Errorf("agent load balancer service does not have valid port")
	}

	// Only get the first IP and port
	endpoint.IP = klSvc.Status.LoadBalancer.Ingress[0].IP
	endpoint.Hostname = klSvc.Status.LoadBalancer.Ingress[0].Hostname
	port.Port = klSvc.Spec.Ports[0].Port
	return nil
}

func (r *ClusterInfoReconciler) setEndpointAddressFromRoute(endpoint *corev1.EndpointAddress) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(r.AgentRoute)
	if err != nil {
		log.Error(err, "Route name input error")
		return err
	}

	route, err := r.RouteV1Client.RouteV1().Routes(klNamespace).Get(context.TODO(), klName, metav1.GetOptions{})

	if err != nil {
		log.Error(err, "Failed to get the route")
		return err
	}

	endpoint.Hostname = route.Spec.Host
	return nil
}

func (r *ClusterInfoReconciler) getCloudVendor(platform string) (cloudVendor clusterv1beta1.CloudVendorType) {
	switch platform {
	case clusterclaim.PlatformAzure:
		cloudVendor = clusterv1beta1.CloudVendorAzure
	case clusterclaim.PlatformAWS:
		cloudVendor = clusterv1beta1.CloudVendorAWS
	case clusterclaim.PlatformIBM:
		cloudVendor = clusterv1beta1.CloudVendorIBM
	case clusterclaim.PlatformIBMZ:
		cloudVendor = clusterv1beta1.CloudVendorIBMZ
	case clusterclaim.PlatformIBMP:
		cloudVendor = clusterv1beta1.CloudVendorIBMP
	case clusterclaim.PlatformGCP:
		cloudVendor = clusterv1beta1.CloudVendorGoogle
	case clusterclaim.PlatformVSphere:
		cloudVendor = clusterv1beta1.CloudVendorVSphere
	case clusterclaim.PlatformOpenStack:
		cloudVendor = clusterv1beta1.CloudVendorOpenStack
	default:
		cloudVendor = clusterv1beta1.CloudVendorOther
	}
	return
}
func (r *ClusterInfoReconciler) getKubeVendor(product string) (kubeVendor clusterv1beta1.KubeVendorType) {
	switch product {
	case clusterclaim.ProductAKS:
		kubeVendor = clusterv1beta1.KubeVendorAKS
	case clusterclaim.ProductGKE:
		kubeVendor = clusterv1beta1.KubeVendorGKE
	case clusterclaim.ProductEKS:
		kubeVendor = clusterv1beta1.KubeVendorEKS
	case clusterclaim.ProductIKS:
		kubeVendor = clusterv1beta1.KubeVendorIKS
	case clusterclaim.ProductICP:
		kubeVendor = clusterv1beta1.KubeVendorICP
	case clusterclaim.ProductOpenShift:
		kubeVendor = clusterv1beta1.KubeVendorOpenShift
	case clusterclaim.ProductOSD:
		kubeVendor = clusterv1beta1.KubeVendorOSD
	default:
		kubeVendor = clusterv1beta1.KubeVendorOther
	}
	return
}

func (r *ClusterInfoReconciler) isOpenshift() bool {
	serverGroups, err := r.KubeClient.Discovery().ServerGroups()
	if err != nil {
		klog.Errorf("failed to get server group %v", err)
		return false
	}
	for _, apiGroup := range serverGroups.Groups {
		if apiGroup.Name == "project.openshift.io" {
			return true
		}
	}
	return false
}

var ocpVersionGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "clusterversions",
}

func (r *ClusterInfoReconciler) getOCPDistributionInfo() (clusterv1beta1.OCPDistributionInfo, error) {
	ocpDistributionInfo := clusterv1beta1.OCPDistributionInfo{}
	obj, err := r.ManagedClusterDynamicClient.Resource(ocpVersionGVR).Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		return ocpDistributionInfo, client.IgnoreNotFound(err)
	}

	ocpDistributionInfo.Channel, _, err = unstructured.NestedString(obj.Object, "spec", "channel")
	if err != nil {
		klog.Errorf("failed to get OCP channel in clusterVersion: %v", err)
	}
	ocpDistributionInfo.DesiredVersion, _, err = unstructured.NestedString(obj.Object, "status", "desired", "version")
	if err != nil {
		klog.Errorf("failed to get OCP desired version in clusterVersion: %v", err)
	}
	availableUpdates, _, err := unstructured.NestedSlice(obj.Object, "status", "availableUpdates")
	if err != nil {
		klog.Errorf("failed to get OCP availableUpdates in clusterVersion: %v", err)
	}
	for _, update := range availableUpdates {
		versionUpdate := clusterv1beta1.OCPVersionRelease{}
		versionUpdate.Version, _, err = unstructured.NestedString(update.(map[string]interface{}), "version")
		if err != nil {
			klog.Errorf("failed to get version in availableUpdates of clusterVersion: %v", err)
		}
		versionUpdate.Image, _, err = unstructured.NestedString(update.(map[string]interface{}), "image")
		if err != nil {
			klog.Errorf("failed to get image in availableUpdates of clusterVersion: %v", err)
		}
		versionUpdate.URL, _, err = unstructured.NestedString(update.(map[string]interface{}), "url")
		if err != nil {
			klog.Errorf("failed to get url in availableUpdates of clusterVersion: %v", err)
		}
		versionUpdate.Channels, _, err = unstructured.NestedStringSlice(update.(map[string]interface{}), "channels")
		if err != nil {
			klog.Errorf("failed to get url in availableUpdates of clusterVersion: %v", err)
		}
		if versionUpdate.Version != "" {
			// ocpDistributionInfo.AvailableUpdates is deprecated in release 2.3 and will be removed in the future.
			// Use VersionAvailableUpdates instead.
			ocpDistributionInfo.AvailableUpdates = append(ocpDistributionInfo.AvailableUpdates, versionUpdate.Version)
			ocpDistributionInfo.VersionAvailableUpdates = append(ocpDistributionInfo.VersionAvailableUpdates, versionUpdate)
		}

	}

	historyItems, _, err := unstructured.NestedSlice(obj.Object, "status", "history")
	if err != nil {
		klog.Errorf("failed to get history in clusterVersion: %v", err)
	}
	for _, historyItem := range historyItems {
		history := clusterv1beta1.OCPVersionUpdateHistory{}
		history.State, _, err = unstructured.NestedString(historyItem.(map[string]interface{}), "state")
		if err != nil {
			klog.Errorf("failed to get the state of history in clusterVersion: %v", err)
		}
		history.Image, _, err = unstructured.NestedString(historyItem.(map[string]interface{}), "image")
		if err != nil {
			klog.Errorf("failed to get the image of history in clusterVersion: %v", err)
		}
		history.Version, _, err = unstructured.NestedString(historyItem.(map[string]interface{}), "version")
		if err != nil {
			klog.Errorf("failed to get the version of history in clusterVersion: %v", err)
		}
		history.Verified, _, err = unstructured.NestedBool(historyItem.(map[string]interface{}), "verified")
		if err != nil {
			klog.Errorf("failed to get the verified of history in clusterVersion: %v", err)
		}
		ocpDistributionInfo.VersionHistory = append(ocpDistributionInfo.VersionHistory, history)
	}

	ocpDistributionInfo.UpgradeFailed = false
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		klog.Errorf("failed to get conditions in clusterVersion: %v", err)
	}
	for _, condition := range conditions {
		conditiontype, _, err := unstructured.NestedString(condition.(map[string]interface{}), "type")
		if err != nil {
			klog.Errorf("failed to get the condition type in clusterVersion: %v", err)
			continue
		}
		if conditiontype == "Failing" {
			conditionstatus, _, err := unstructured.NestedString(condition.(map[string]interface{}), "status")
			if err != nil {
				klog.Errorf("failed to get the status of Failing condition in clusterVersion : %v", err)
				continue
			}
			if conditionstatus == "True" && ocpDistributionInfo.DesiredVersion != ocpDistributionInfo.Version {
				ocpDistributionInfo.UpgradeFailed = true
			}
			break
		}
	}

	return ocpDistributionInfo, nil
}

func (r *ClusterInfoReconciler) getDistributionInfo() (clusterv1beta1.DistributionInfo, error) {
	var err error
	var distributionInfo = clusterv1beta1.DistributionInfo{
		Type: clusterv1beta1.DistributionTypeUnknown,
	}

	switch {
	case r.isOpenshift():
		distributionInfo.Type = clusterv1beta1.DistributionTypeOCP
		if distributionInfo.OCP, err = r.getOCPDistributionInfo(); err != nil {
			return distributionInfo, err
		}
	}
	return distributionInfo, nil
}

func (r *ClusterInfoReconciler) RefreshAgentServer(clusterInfo *clusterv1beta1.ManagedClusterInfo) {
	select {
	case r.Agent.RunServer <- *clusterInfo:
		log.Info("Signal agent server to start")
	default:
	}

	r.Agent.RefreshServerIfNeeded(clusterInfo)
}

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	claimSource := clusterclaim.NewClusterClaimSource(r.ClaimInformer)
	nodeSource := &nodeSource{nodeInformer: r.NodeInformer.Informer()}
	return ctrl.NewControllerManagedBy(mgr).
		Watches(claimSource, &clusterclaim.ClusterClaimEventHandler{}).
		Watches(nodeSource, &nodeEventHandler{}).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}

// nodeSource is the event source of nodes on managed cluster.
type nodeSource struct {
	nodeInformer cache.SharedIndexInformer
}

var _ source.SyncingSource = &nodeSource{}

func (s *nodeSource) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	predicates ...predicate.Predicate) error {
	// all predicates are ignored
	s.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handler.Create(event.CreateEvent{}, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handler.Update(event.UpdateEvent{}, queue)
		},
		DeleteFunc: func(obj interface{}) {
			handler.Delete(event.DeleteEvent{}, queue)
		},
	})

	return nil
}

func (s *nodeSource) WaitForSync(ctx context.Context) error {
	if ok := cache.WaitForCacheSync(ctx.Done(), s.nodeInformer.HasSynced); !ok {
		return fmt.Errorf("Never achieved initial sync")
	}
	return nil
}

// nodeEventHandler maps any event to an empty request
type nodeEventHandler struct{}

var _ handler.EventHandler = &nodeEventHandler{}

func (e *nodeEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}

func (e *nodeEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	q.Add(reconcile.Request{})
}
