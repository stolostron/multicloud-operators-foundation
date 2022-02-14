package controllers

import (
	"context"
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/go-logr/logr"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	clusterv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	clusterv1alpha1informer "open-cluster-management.io/api/client/cluster/informers/externalversions/cluster/v1alpha1"
	clusterv1alpha1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1alpha1"
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
	Log             logr.Logger
	Scheme          *runtime.Scheme
	KubeClient      kubernetes.Interface
	NodeInformer    coreinformers.NodeInformer
	NodeLister      corev1lister.NodeLister
	ClaimInformer   clusterv1alpha1informer.ClusterClaimInformer
	ClaimLister     clusterv1alpha1lister.ClusterClaimLister
	RouteV1Client   routev1.Interface
	ConfigV1Client  openshiftclientset.Interface
	ClusterName     string
	MasterAddresses string
	AgentAddress    string
	AgentIngress    string
	AgentRoute      string
	AgentService    string
	AgentPort       int32
	Agent           *agent.Agent
	isIbmCloud      bool
}

func (r *ClusterInfoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ManagedClusterInfo", req.NamespacedName)

	request := types.NamespacedName{Namespace: r.ClusterName, Name: r.ClusterName}
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err := r.Get(ctx, request, clusterInfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if utils.ClusterIsOffLine(clusterInfo.Status.Conditions) {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// refresh logging server CA if CA is changed
	r.RefreshAgentServer(clusterInfo)

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

	if newStatus.CloudVendor == clusterv1beta1.CloudVendorIBM {
		r.isIbmCloud = true
	}

	// Get distribution info
	newStatus.DistributionInfo, err = r.getDistributionInfo(ctx)
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

	// check if distributionInfo is Updated.
	// sort the slices in distributionInfo to make it comparable using DeepEqual if not update.
	infoUpdated := distributionInfoUpdated(&clusterInfo.Status.DistributionInfo, &newStatus.DistributionInfo)

	// need to sync ocp ClusterVersion info every 5 min since do not watch it.
	if !infoUpdated && equality.Semantic.DeepEqual(newStatus, clusterInfo.Status) {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	meta.SetStatusCondition(&newStatus.Conditions, newSyncedCondition)

	if equality.Semantic.DeepEqual(newStatus, clusterInfo.Status) {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	clusterInfo.Status = newStatus
	err = r.Client.Status().Update(ctx, clusterInfo)
	if err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
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
		klog.Error(err, "Route name input error")
		return err
	}

	route, err := r.RouteV1Client.RouteV1().Routes(klNamespace).Get(context.TODO(), klName, metav1.GetOptions{})

	if err != nil {
		klog.Error(err, "Failed to get the route")
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
	case clusterclaim.ProductOpenShift, clusterclaim.ProductROSA, clusterclaim.ProductARO:
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

func (r *ClusterInfoReconciler) getOCPDistributionInfo(ctx context.Context) (clusterv1beta1.OCPDistributionInfo, error) {
	ocpDistributionInfo := clusterv1beta1.OCPDistributionInfo{}
	clusterVersion, err := r.ConfigV1Client.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return ocpDistributionInfo, client.IgnoreNotFound(err)
	}
	ocpDistributionInfo.Channel = clusterVersion.Spec.Channel

	// ocpDistributionInfo.DesiredVersion is deprecated in release 2.3 and will be remove in the feture.
	// Use ocpDistributionInfo.Desired instead.
	ocpDistributionInfo.DesiredVersion = clusterVersion.Status.Desired.Version
	ocpDistributionInfo.Desired.Version = clusterVersion.Status.Desired.Version
	ocpDistributionInfo.Desired.Image = clusterVersion.Status.Desired.Image
	ocpDistributionInfo.Desired.URL = string(clusterVersion.Status.Desired.URL)
	ocpDistributionInfo.Desired.Channels = clusterVersion.Status.Desired.Channels

	availableUpdates := clusterVersion.Status.AvailableUpdates
	for _, update := range availableUpdates {
		versionUpdate := clusterv1beta1.OCPVersionRelease{}
		versionUpdate.Version = update.Version
		versionUpdate.Image = update.Image
		versionUpdate.URL = string(update.URL)
		versionUpdate.Channels = update.Channels
		if versionUpdate.Version != "" {
			// ocpDistributionInfo.AvailableUpdates is deprecated in release 2.3 and will be removed in the future.
			// Use VersionAvailableUpdates instead.
			ocpDistributionInfo.AvailableUpdates = append(ocpDistributionInfo.AvailableUpdates, versionUpdate.Version)
			ocpDistributionInfo.VersionAvailableUpdates = append(ocpDistributionInfo.VersionAvailableUpdates, versionUpdate)
		}

	}
	historyItems := clusterVersion.Status.History
	for _, historyItem := range historyItems {
		history := clusterv1beta1.OCPVersionUpdateHistory{}
		history.State = string(historyItem.State)
		history.Image = historyItem.Image
		history.Version = historyItem.Version
		history.Verified = historyItem.Verified
		ocpDistributionInfo.VersionHistory = append(ocpDistributionInfo.VersionHistory, history)
	}
	ocpDistributionInfo.ManagedClusterClientConfig = r.getClientConfig(ctx)
	ocpDistributionInfo.UpgradeFailed = false
	conditions := clusterVersion.Status.Conditions

	for _, condition := range conditions {
		if condition.Type == "Failing" {
			if condition.Status == "True" && ocpDistributionInfo.DesiredVersion != ocpDistributionInfo.Version {
				ocpDistributionInfo.UpgradeFailed = true
			}
			break
		}
	}

	return ocpDistributionInfo, nil
}

func (r *ClusterInfoReconciler) getClientConfig(ctx context.Context) clusterv1beta1.ClientConfig {
	// get ocp apiserver url
	kubeAPIServer, err := utils.GetKubeAPIServerAddress(ctx, r.ConfigV1Client)
	if err != nil {
		klog.Errorf("Failed to get kube Apiserver. err:%v", err)
		return clusterv1beta1.ClientConfig{}
	}

	// get ocp ca
	clusterca := r.getClusterCA(ctx, kubeAPIServer)
	if len(clusterca) <= 0 {
		klog.Errorf("Failed to get clusterca.")
		return clusterv1beta1.ClientConfig{}
	}

	klog.V(5).Infof("kubeapiserver: %v, clusterca:%v", kubeAPIServer, clusterca)

	return clusterv1beta1.ClientConfig{
		URL:      kubeAPIServer,
		CABundle: clusterca,
	}
}

func (r *ClusterInfoReconciler) getClusterCA(ctx context.Context, kubeAPIServer string) []byte {
	// Get ca from apiserver
	certData, err := utils.GetCAFromApiserver(ctx, r.ConfigV1Client, r.KubeClient, kubeAPIServer)
	if err == nil && len(certData) > 0 {
		return certData
	}
	klog.V(3).Infof("Failed to get ca from apiserver, error:%v", err)

	// Get ca from configmap in kube-public namespace
	certData, err = utils.GetCAFromConfigMap(ctx, r.KubeClient)
	if err == nil && len(certData) > 0 {
		return certData
	}

	klog.V(3).Infof("Failed to get ca from kubepublic namespace configmap, error:%v", err)

	// Fallback to service account token ca.crt
	certData, err = utils.GetCAFromServiceAccount(ctx, r.KubeClient)
	if err == nil && len(certData) > 0 {
		// check if it's roks
		// if it's ocp && it's on ibm cloud, we treat it as roks
		if r.isIbmCloud {
			// simply don't give any certs as the apiserver is using certs signed by known CAs
			return nil
		}
		return certData
	}

	klog.V(3).Infof("Failed to get ca from service account, error:%v", err)
	return nil
}

func (r *ClusterInfoReconciler) getDistributionInfo(ctx context.Context) (clusterv1beta1.DistributionInfo, error) {
	var err error
	var distributionInfo = clusterv1beta1.DistributionInfo{
		Type: clusterv1beta1.DistributionTypeUnknown,
	}

	switch {
	case r.isOpenshift():
		distributionInfo.Type = clusterv1beta1.DistributionTypeOCP
		if distributionInfo.OCP, err = r.getOCPDistributionInfo(ctx); err != nil {
			return distributionInfo, err
		}
	}
	return distributionInfo, nil
}

func distributionInfoUpdated(old, new *clusterv1beta1.DistributionInfo) bool {
	switch new.Type {
	case clusterv1beta1.DistributionTypeOCP:
		return ocpDistributionInfoUpdated(&old.OCP, &new.OCP)
	}
	return false
}

func ocpDistributionInfoUpdated(old, new *clusterv1beta1.OCPDistributionInfo) bool {
	sort.SliceStable(new.AvailableUpdates, func(i, j int) bool { return new.AvailableUpdates[i] < new.AvailableUpdates[j] })
	sort.SliceStable(new.VersionAvailableUpdates, func(i, j int) bool {
		return new.VersionAvailableUpdates[i].Version < new.VersionAvailableUpdates[j].Version
	})
	sort.SliceStable(new.VersionHistory, func(i, j int) bool { return new.VersionHistory[i].Version < new.VersionHistory[j].Version })
	return !equality.Semantic.DeepEqual(old, new)
}

func (r *ClusterInfoReconciler) RefreshAgentServer(clusterInfo *clusterv1beta1.ManagedClusterInfo) {
	select {
	case r.Agent.RunServer <- *clusterInfo:
		klog.Info("Signal agent server to start")
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
