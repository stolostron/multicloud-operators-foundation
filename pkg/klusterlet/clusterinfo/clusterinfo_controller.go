package controllers

import (
	"context"
	"fmt"
	clusterclientset "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	"net"
	"strings"
	"time"

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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterInfoReconciler reconciles a ManagedClusterInfo object
type ClusterInfoReconciler struct {
	client.Client
	Log                         logr.Logger
	Scheme                      *runtime.Scheme
	KubeClient                  kubernetes.Interface
	ManagedClusterDynamicClient dynamic.Interface
	ClusterClient               clusterclientset.Interface
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

func (r *ClusterInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("ManagedClusterInfo", req.NamespacedName)

	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err := r.Get(ctx, req.NamespacedName, clusterInfo)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update cluster info status here.
	newStatus := clusterv1beta1.ClusterInfoStatus{}
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

	// Get nodeList
	nodeList, err := r.getNodeList()
	if err != nil {
		log.Error(err, "Failed to get nodes status")
		errs = append(errs, fmt.Errorf("failed to get nodes status, error:%v ", err))
	} else {
		newStatus.NodeList = nodeList
	}

	claims, err := r.ClusterClient.ClusterV1alpha1().ClusterClaims().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err, "Failed to list claims.")
		errs = append(errs, fmt.Errorf("failed to list clusterClaims error:%v ", err))
	}
	for _, claim := range claims.Items {
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
		newSyncedCondition.Status = metav1.ConditionFalse
		newSyncedCondition.Reason = clusterv1beta1.ReasonManagedClusterInfoSyncedFailed
		applyErrors := errors.NewAggregate(errs)
		newSyncedCondition.Message = applyErrors.Error()
	}

	needUpdate := false
	oldStatus := clusterInfo.Status.DeepCopy()
	oldSyncedCondition := meta.FindStatusCondition(oldStatus.Conditions, clusterv1beta1.ManagedClusterInfoSynced)
	if oldSyncedCondition != nil {
		oldSyncedCondition.LastTransitionTime = metav1.Time{}
		if !equality.Semantic.DeepEqual(newSyncedCondition, *oldSyncedCondition) {
			needUpdate = true
		}
	} else {
		needUpdate = true
	}

	oldStatus.Conditions = []metav1.Condition{}
	if !equality.Semantic.DeepEqual(newStatus, *oldStatus) {
		needUpdate = true
	}

	if needUpdate {
		newStatus.Conditions = clusterInfo.Status.Conditions
		meta.SetStatusCondition(&newStatus.Conditions, newSyncedCondition)
		clusterInfo.Status = newStatus
		err = r.Client.Status().Update(ctx, clusterInfo)
		if err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	r.RefreshAgentServer(clusterInfo)

	return ctrl.Result{RequeueAfter: 60 * time.Minute}, nil
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
	case clusterclaim.PlatformBareMetal:
		cloudVendor = clusterv1beta1.CloudVendorBareMetal
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

const (
	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"

	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"

	// copied from k8s.io/api/core/v1/well_known_label.go
	LabelZoneFailureDomain  = "failure-domain.beta.kubernetes.io/zone"
	LabelZoneRegion         = "failure-domain.beta.kubernetes.io/region"
	LabelInstanceType       = "beta.kubernetes.io/instance-type"
	LabelInstanceTypeStable = "node.kubernetes.io/instance-type"
)

func (r *ClusterInfoReconciler) getNodeList() ([]clusterv1beta1.NodeStatus, error) {
	var nodeList []clusterv1beta1.NodeStatus
	nodes, err := r.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		nodeStatus := clusterv1beta1.NodeStatus{
			Name:       node.Name,
			Labels:     map[string]string{},
			Capacity:   clusterv1beta1.ResourceList{},
			Conditions: []clusterv1beta1.NodeCondition{},
		}

		// The roles are determined by looking for:
		// * a node-role.kubernetes.io/<role>="" label
		// * a kubernetes.io/role="<role>" label
		for k, v := range node.Labels {
			if strings.HasPrefix(k, LabelNodeRolePrefix) || k == NodeLabelRole ||
				k == LabelZoneFailureDomain || k == LabelZoneRegion ||
				k == LabelInstanceType || k == LabelInstanceTypeStable {
				nodeStatus.Labels[k] = v
			}
		}

		// append capacity of cpu and memory
		for k, v := range node.Status.Capacity {
			switch {
			case k == corev1.ResourceCPU:
				nodeStatus.Capacity[clusterv1beta1.ResourceCPU] = v
			case k == corev1.ResourceMemory:
				nodeStatus.Capacity[clusterv1beta1.ResourceMemory] = v
			}
		}

		// append condition of NodeReady
		readyCondition := clusterv1beta1.NodeCondition{
			Type: corev1.NodeReady,
		}
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				readyCondition.Status = condition.Status
				break
			}
		}
		nodeStatus.Conditions = append(nodeStatus.Conditions, readyCondition)

		nodeList = append(nodeList, nodeStatus)
	}
	return nodeList, nil
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

	ocpDistributionInfo.DesiredVersion, _, err = unstructured.NestedString(obj.Object, "status", "desired", "version")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
	}
	availableUpdates, _, err := unstructured.NestedSlice(obj.Object, "status", "availableUpdates")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
	}
	for _, update := range availableUpdates {
		availableVersion, _, err := unstructured.NestedString(update.(map[string]interface{}), "version")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			continue
		}
		ocpDistributionInfo.AvailableUpdates = append(ocpDistributionInfo.AvailableUpdates, availableVersion)
	}

	ocpDistributionInfo.UpgradeFailed = false
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
	}
	for _, condition := range conditions {
		conditiontype, _, err := unstructured.NestedString(condition.(map[string]interface{}), "type")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			continue
		}
		if conditiontype == "Failing" {
			conditionstatus, _, err := unstructured.NestedString(condition.(map[string]interface{}), "status")
			if err != nil {
				klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.ManagedClusterInfo{}).
		Complete(r)
}
