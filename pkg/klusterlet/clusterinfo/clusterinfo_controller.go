package controllers

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
)

// ClusterInfoReconciler reconciles a ManagedClusterInfo object
type ClusterInfoReconciler struct {
	client.Client
	Log                         logr.Logger
	Scheme                      *runtime.Scheme
	KubeClient                  kubernetes.Interface
	ManagedClusterDynamicClient dynamic.Interface
	RouteV1Client               routev1.Interface
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
	newStatus := clusterv1beta1.ClusterInfoStatus{
		Conditions: clusterInfo.Status.Conditions,
	}

	// Config console url
	_, _, clusterURL := r.getMasterAddresses()
	newStatus.ConsoleURL = clusterURL

	// Config Agent endpoint
	agentEndpoint, agentPort, err := r.readAgentConfig()
	if err != nil {
		log.Error(err, "Failed to get agent server config")
		return ctrl.Result{}, err
	}
	newStatus.LoggingEndpoint = *agentEndpoint
	newStatus.LoggingPort = *agentPort

	// Get version
	newStatus.Version = r.getVersion()

	// Get distribution info
	newStatus.DistributionInfo, err = r.getDistributionInfo()
	if err != nil {
		log.Error(err, "Failed to get distribution info")
		return ctrl.Result{}, err
	}

	// Get nodeList
	nodeList, err := r.getNodeList()
	if err != nil {
		log.Error(err, "Failed to get nodes status")
		return ctrl.Result{}, err
	}
	newStatus.NodeList = nodeList

	if !equality.Semantic.DeepEqual(newStatus, clusterInfo.Status) {
		clusterInfo.Status = newStatus
		err = r.Client.Status().Update(ctx, clusterInfo)
		if err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	r.RefreshAgentServer(clusterInfo)

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ClusterInfoReconciler) getMasterAddresses() ([]corev1.EndpointAddress, []corev1.EndpointPort, string) {
	// for OpenShift
	masterAddresses, masterPorts, clusterURL, notEmpty, err := r.getMasterAddressesFromConsoleConfig()
	if err == nil && notEmpty {
		return masterAddresses, masterPorts, clusterURL
	}

	if r.MasterAddresses == "" {
		kubeEndpoints, serviceErr := r.KubeClient.CoreV1().Endpoints("default").Get("kubernetes", metav1.GetOptions{})
		if serviceErr == nil && len(kubeEndpoints.Subsets) > 0 {
			masterAddresses = kubeEndpoints.Subsets[0].Addresses
			masterPorts = kubeEndpoints.Subsets[0].Ports
		}
	} else {
		masterAddresses = append(masterAddresses, corev1.EndpointAddress{IP: r.MasterAddresses})
	}

	return masterAddresses, masterPorts, clusterURL
}

// for OpenShift, read endpoint address from console-config in openshift-console
func (r *ClusterInfoReconciler) getMasterAddressesFromConsoleConfig() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, bool, error) {
	masterAddresses := []corev1.EndpointAddress{}
	masterPorts := []corev1.EndpointPort{}
	clusterURL := ""

	cfg, err := r.KubeClient.CoreV1().ConfigMaps("openshift-console").Get("console-config", metav1.GetOptions{})
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
					port, _ := strconv.ParseInt(euArray[1], 10, 32)
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
	klIngress, err := r.KubeClient.ExtensionsV1beta1().Ingresses(klNamespace).Get(klName, metav1.GetOptions{})
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

	klSvc, err := r.KubeClient.CoreV1().Services(klNamespace).Get(klName, metav1.GetOptions{})
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

	route, err := r.RouteV1Client.RouteV1().Routes(klNamespace).Get(klName, metav1.GetOptions{})

	if err != nil {
		log.Error(err, "Failed to get the route")
		return err
	}

	endpoint.Hostname = route.Spec.Host
	return nil
}

func (r *ClusterInfoReconciler) getVersion() string {
	serverVersionString := ""
	serverVersion, err := r.KubeClient.Discovery().ServerVersion()
	if err == nil {
		serverVersionString = serverVersion.String()
	}

	return serverVersionString
}

const (
	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"

	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"

	// copied from k8s.io/api/core/v1/well_known_labels.go
	LabelZoneFailureDomain  = "failure-domain.beta.kubernetes.io/zone"
	LabelZoneRegion         = "failure-domain.beta.kubernetes.io/region"
	LabelInstanceType       = "beta.kubernetes.io/instance-type"
	LabelInstanceTypeStable = "node.kubernetes.io/instance-type"
)

func (r *ClusterInfoReconciler) getNodeList() ([]clusterv1beta1.NodeStatus, error) {
	var nodeList []clusterv1beta1.NodeStatus
	nodes, err := r.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
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

var ocpVersionGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "clusterversions",
}

func (r *ClusterInfoReconciler) getDistributionInfo() (clusterv1beta1.DistributionInfo, error) {
	var isOpenShift = false
	distributionInfo := clusterv1beta1.DistributionInfo{}
	serverGroups, err := r.KubeClient.Discovery().ServerGroups()
	if err != nil {
		return distributionInfo, err
	}
	for _, apiGroup := range serverGroups.Groups {
		if apiGroup.Name == "project.openshift.io" {
			isOpenShift = true
			break
		}
	}
	if !isOpenShift {
		return distributionInfo, nil
	}

	distributionInfo.Type = clusterv1beta1.DistributionTypeOCP
	obj, err := r.ManagedClusterDynamicClient.Resource(ocpVersionGVR).Get("version", metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get OCP cluster version: %v", err)
		return distributionInfo, client.IgnoreNotFound(err)
	}
	historyItems, _, err := unstructured.NestedSlice(obj.Object, "status", "history")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in history of status: %v", err)
		return distributionInfo, client.IgnoreNotFound(err)
	}
	var version string
	for _, historyItem := range historyItems {
		state, _, err := unstructured.NestedString(historyItem.(map[string]interface{}), "state")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			return distributionInfo, client.IgnoreNotFound(err)
		}
		if state == "Completed" {
			version, _, err = unstructured.NestedString(historyItem.(map[string]interface{}), "version")
			if err != nil {
				klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
				return distributionInfo, client.IgnoreNotFound(err)
			}
			break
		}
	}

	desiredVersion, _, err := unstructured.NestedString(obj.Object, "status", "desired", "version")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
		return distributionInfo, client.IgnoreNotFound(err)
	}
	availableUpdates, _, err := unstructured.NestedSlice(obj.Object, "status", "availableUpdates")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
		return distributionInfo, client.IgnoreNotFound(err)
	}
	var availableVersions []string
	for _, update := range availableUpdates {
		availableVersion, _, err := unstructured.NestedString(update.(map[string]interface{}), "version")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			return distributionInfo, client.IgnoreNotFound(err)
		}
		availableVersions = append(availableVersions, availableVersion)
	}
	conditions, _, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
		return distributionInfo, client.IgnoreNotFound(err)
	}
	upgradeFailed := false

	for _, condition := range conditions {
		conditiontype, _, err := unstructured.NestedString(condition.(map[string]interface{}), "type")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			return distributionInfo, client.IgnoreNotFound(err)
		}
		if conditiontype == "Failing" {
			conditionstatus, _, err := unstructured.NestedString(condition.(map[string]interface{}), "status")
			if err != nil {
				klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
				return distributionInfo, client.IgnoreNotFound(err)
			}
			if conditionstatus == "True" {
				upgradeFailed = true
			}
			break
		}
	}
	distributionInfo.OCP = clusterv1beta1.OCPDistributionInfo{
		Version: version,
		//status.desired.version
		DesiredVersion: desiredVersion,
		//true when the first status.conditions element with type === Failing has status === True
		UpgradeFailed: upgradeFailed,
		//the value of the version property from each element in status.availableUpdates
		AvailableUpdates: availableVersions,
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
