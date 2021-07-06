package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/klusterlet/agent"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	if utils.ClusterIsOffLine(clusterInfo.Status.Conditions) {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// Update cluster info status here.
	newStatus := clusterv1beta1.ClusterInfoStatus{}
	var errs []error
	// Config console url
	_, _, clusterURL := r.getMasterAddresses()
	newStatus.ConsoleURL = clusterURL

	// Config Agent endpoint
	agentEndpoint, agentPort, err := r.readAgentConfig()
	if err != nil {
		log.Error(err, "Failed to get agent server config")
		errs = append(errs, fmt.Errorf("failed to get agent server config, error:%v ", err))
	} else {
		newStatus.LoggingEndpoint = *agentEndpoint
		newStatus.LoggingPort = *agentPort
	}

	// Get version
	newStatus.Version = r.getVersion()

	// Get distribution info and ClusterID
	newStatus.DistributionInfo, newStatus.ClusterID, err = r.getDistributionInfoAndClusterID()
	if err != nil {
		log.Error(err, "Failed to get distribution info and clusterID")
		errs = append(errs, fmt.Errorf("failed to get distribution info and clusterID, error:%v ", err))
	}

	// Get Vendor
	kubeVendor, cloudVendor := r.getVendor(newStatus.Version, newStatus.DistributionInfo.Type == clusterv1beta1.DistributionTypeOCP)
	newStatus.KubeVendor = kubeVendor
	newStatus.CloudVendor = cloudVendor

	// Get nodeList
	nodeList, err := r.getNodeList()
	if err != nil {
		log.Error(err, "Failed to get nodes status")
		errs = append(errs, fmt.Errorf("failed to get nodes status, error:%v ", err))
	} else {
		newStatus.NodeList = nodeList
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

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ClusterInfoReconciler) getMasterAddresses() ([]corev1.EndpointAddress, []corev1.EndpointPort, string) {
	// for OpenShift
	masterAddresses, masterPorts, clusterURL, notEmpty, err := r.getMasterAddressesFromConsoleConfig()
	if err == nil && notEmpty {
		return masterAddresses, masterPorts, clusterURL
	}

	if r.MasterAddresses == "" {
		kubeEndpoints, serviceErr := r.KubeClient.CoreV1().Endpoints("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
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

	cfg, err := r.KubeClient.CoreV1().ConfigMaps("openshift-console").Get(context.TODO(), "console-config", metav1.GetOptions{})
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

func (r *ClusterInfoReconciler) getVersion() string {
	serverVersionString := ""
	serverVersion, err := r.KubeClient.Discovery().ServerVersion()
	if err == nil {
		serverVersionString = serverVersion.String()
	}

	return serverVersionString
}

func (r *ClusterInfoReconciler) getVendor(gitVersion string, isOpenShift bool) (
	kubeVendor clusterv1beta1.KubeVendorType, cloudVendor clusterv1beta1.CloudVendorType) {
	gitVersion = strings.ToUpper(gitVersion)
	switch {
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorIKS)):
		kubeVendor = clusterv1beta1.KubeVendorIKS
		cloudVendor = clusterv1beta1.CloudVendorIBM
		return
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorEKS)):
		kubeVendor = clusterv1beta1.KubeVendorEKS
		cloudVendor = clusterv1beta1.CloudVendorAWS
		return
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorGKE)):
		kubeVendor = clusterv1beta1.KubeVendorGKE
		cloudVendor = clusterv1beta1.CloudVendorGoogle
		return
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorICP)):
		kubeVendor = clusterv1beta1.KubeVendorICP
	case r.isOpenshiftDedicated():
		kubeVendor = clusterv1beta1.KubeVendorOSD
	case isOpenShift:
		kubeVendor = clusterv1beta1.KubeVendorOpenShift
	default:
		kubeVendor = clusterv1beta1.KubeVendorOther
	}

	cloudVendor = r.getCloudVendor()
	if cloudVendor == clusterv1beta1.CloudVendorAzure && kubeVendor == clusterv1beta1.KubeVendorOther {
		kubeVendor = clusterv1beta1.KubeVendorAKS
	}

	return
}

func (r *ClusterInfoReconciler) getCloudVendor() clusterv1beta1.CloudVendorType {
	nodes, err := r.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to get nodes list %v", err)
		return clusterv1beta1.CloudVendorOther
	}

	if len(nodes.Items) == 0 {
		klog.Errorf("failed to get nodes list, the count of nodes is 0")
		return clusterv1beta1.CloudVendorOther
	}

	providerID := nodes.Items[0].Spec.ProviderID
	switch {
	case strings.Contains(providerID, "ibm"):
		return clusterv1beta1.CloudVendorIBM
	case strings.Contains(providerID, "azure"):
		return clusterv1beta1.CloudVendorAzure
	case strings.Contains(providerID, "aws"):
		return clusterv1beta1.CloudVendorAWS
	case strings.Contains(providerID, "gce"):
		return clusterv1beta1.CloudVendorGoogle
	case strings.Contains(providerID, "vsphere"):
		return clusterv1beta1.CloudVendorVSphere
	case strings.Contains(providerID, "openstack"):
		return clusterv1beta1.CloudVendorOpenStack
	}

	return clusterv1beta1.CloudVendorOther
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

func (r *ClusterInfoReconciler) isOpenshiftDedicated() bool {
	hasProject := false
	hasManaged := false
	serverGroups, err := r.KubeClient.Discovery().ServerGroups()
	if err != nil {
		klog.Errorf("failed to get server group %v", err)
		return false
	}
	for _, apiGroup := range serverGroups.Groups {
		if apiGroup.Name == "project.openshift.io" {
			hasProject = true
		}
		// this API group is created for the Openshift Dedicated platform to manage various permissions.
		// defined in https://github.com/openshift/rbac-permissions-operator/blob/master/pkg/apis/managed/v1alpha1/subjectpermission_types.go
		if apiGroup.Name == "managed.openshift.io" {
			hasManaged = true
		}
	}
	return hasProject && hasManaged
}

var ocpVersionGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "clusterversions",
}

const (
	OCP3Version = "3"
)

func (r *ClusterInfoReconciler) getOCPDistributionInfo() (clusterv1beta1.OCPDistributionInfo, string, error) {
	ocpDistributionInfo := clusterv1beta1.OCPDistributionInfo{}
	obj, err := r.ManagedClusterDynamicClient.Resource(ocpVersionGVR).Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ocpDistributionInfo.DesiredVersion = OCP3Version
			ocpDistributionInfo.Version = OCP3Version
			return ocpDistributionInfo, "", nil
		}
		klog.Errorf("failed to get OCP cluster version: %v", err)
		return ocpDistributionInfo, "", client.IgnoreNotFound(err)
	}

	clusterID, _, err := unstructured.NestedString(obj.Object, "spec", "clusterID")
	if err != nil {
		klog.Errorf("failed to get OCP clusterID in spec: %v", err)
		return ocpDistributionInfo, "", err
	}

	historyItems, _, err := unstructured.NestedSlice(obj.Object, "status", "history")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in history of status: %v", err)
	}
	for _, historyItem := range historyItems {
		state, _, err := unstructured.NestedString(historyItem.(map[string]interface{}), "state")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			continue
		}
		if state == "Completed" {
			ocpDistributionInfo.Version, _, err = unstructured.NestedString(historyItem.(map[string]interface{}), "version")
			if err != nil {
				klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			}
			break
		}
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

	return ocpDistributionInfo, clusterID, nil
}

func (r *ClusterInfoReconciler) getDistributionInfoAndClusterID() (clusterv1beta1.DistributionInfo, string, error) {
	var err error
	var clusterID string
	var distributionInfo = clusterv1beta1.DistributionInfo{
		Type: clusterv1beta1.DistributionTypeUnknown,
	}

	switch {
	case r.isOpenshift():
		distributionInfo.Type = clusterv1beta1.DistributionTypeOCP
		if distributionInfo.OCP, clusterID, err = r.getOCPDistributionInfo(); err != nil {
			return distributionInfo, clusterID, err
		}
	}
	return distributionInfo, clusterID, nil
}

var ocpInfrGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "infrastructures",
}

// should be the type defined in infrastructure.config.openshift.io
const (
	AWSPlatformType = "AWS"
	GCPPlatformType = "GCP"
	IBMPlatformType = "IBM"
)

func (r *ClusterInfoReconciler) getClusterRegion() string {
	var region = ""

	switch {
	case r.isOpenshift():
		obj, err := r.ManagedClusterDynamicClient.Resource(ocpInfrGVR).Get(context.TODO(), "cluster", metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
			return ""
		}

		platformType, _, err := unstructured.NestedString(obj.Object, "status", "platform")
		if err != nil {
			klog.Errorf("failed to get OCP platform type in status of infrastructures.config.openshift.io/cluster: %v", err)
			return ""
		}

		// only ocp on aws and gcp has region definition
		// refer to https://github.com/openshift/api/blob/master/config/v1/types_infrastructure.go
		switch platformType {
		case AWSPlatformType:
			region, _, err = unstructured.NestedString(obj.Object, "status", "platformStatus", "aws", "region")
			if err != nil {
				klog.Errorf("failed to get OCP region in status of infrastructures.config.openshift.io/cluster: %v", err)
				return ""
			}
		case GCPPlatformType:
			region, _, err = unstructured.NestedString(obj.Object, "status", "platformStatus", "gcp", "region")
			if err != nil {
				klog.Errorf("failed to get OCP region in status of infrastructures.config.openshift.io/cluster: %v", err)
				return ""
			}
		}
	}

	return region
}

type InfraConfig struct {
	InfraName string `json:"infraName,omitempty"`
}

func (r *ClusterInfoReconciler) getInfraConfig() string {
	if !r.isOpenshift() {
		return ""
	}

	obj, err := r.ManagedClusterDynamicClient.Resource(ocpInfrGVR).Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
		return ""
	}

	infraConfig := InfraConfig{}
	infraConfig.InfraName, _, err = unstructured.NestedString(obj.Object, "status", "infrastructureName")
	if err != nil {
		klog.Errorf("failed to get OCP infrastructure Name in status of infrastructures.config.openshift.io/cluster: %v", err)
		return ""
	}

	infraConfigRaw, err := json.Marshal(infraConfig)
	if err != nil {
		klog.Errorf("failed to marshal infraConfig: %v", err)
		return ""
	}
	return string(infraConfigRaw)
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

func (r *ClusterInfoReconciler) GetClusterClaims() ([]*clusterv1alpha1.ClusterClaim, error) {
	claims := []*clusterv1alpha1.ClusterClaim{}
	distributionInfo, clusterID, err := r.getDistributionInfoAndClusterID()
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution info and clusterID, error: %w ", err)
	}

	// handle OpenShift specific claims
	if len(clusterID) > 0 {
		claims = append(claims, newClusterClaim("id.openshift.io", clusterID))
	}
	if len(distributionInfo.OCP.Version) > 0 {
		claims = append(claims, newClusterClaim("version.openshift.io", distributionInfo.OCP.Version))
	}

	infraConfig := r.getInfraConfig()
	if len(infraConfig) > 0 {
		claims = append(claims, newClusterClaim("infrastructure.openshift.io", infraConfig))
	}

	_, _, consoleURL := r.getMasterAddresses()
	if len(consoleURL) > 0 {
		claims = append(claims, newClusterClaim("consoleurl.cluster.open-cluster-management.io", consoleURL))
	}

	// handle reserved claims
	claims = append(claims, newClusterClaim("id.k8s.io", r.ClusterName))
	kubeVersion, platform, product := r.getVersionPlatformProduct(distributionInfo)
	claims = append(claims, newClusterClaim("kubeversion.open-cluster-management.io", kubeVersion))
	if len(platform) > 0 {
		claims = append(claims, newClusterClaim("platform.open-cluster-management.io", platform))
	}
	if len(product) > 0 {
		claims = append(claims, newClusterClaim("product.open-cluster-management.io", product))
	}

	region := r.getClusterRegion()
	if len(region) > 0 {
		claims = append(claims, newClusterClaim("region.open-cluster-management.io", region))
	}

	return claims, nil
}

func (r *ClusterInfoReconciler) getVersionPlatformProduct(distributionInfo clusterv1beta1.DistributionInfo) (kubeVersion, platform, product string) {
	kubeVersion = r.getVersion()
	isOpenShift := distributionInfo.Type == clusterv1beta1.DistributionTypeOCP
	cloudVendor := r.getCloudVendor()
	gitVersion := strings.ToUpper(kubeVersion)

	// deal with product and platform
	switch {
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorIKS)):
		// IBM Cloud Platform + IKS
		platform = IBMPlatformType
		product = string(clusterv1beta1.KubeVendorIKS)
		return
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorEKS)):
		// AWS + EKS
		platform = AWSPlatformType
		product = string(clusterv1beta1.KubeVendorEKS)
		return
	case strings.Contains(gitVersion, string(clusterv1beta1.KubeVendorGKE)):
		// Google Cloud Platform + GKE
		platform = GCPPlatformType
		product = string(clusterv1beta1.KubeVendorGKE)
		return
	case r.isOpenshiftDedicated():
		product = string(clusterv1beta1.KubeVendorOSD)
	case isOpenShift:
		product = string(clusterv1beta1.KubeVendorOpenShift)
	}

	switch cloudVendor {
	case clusterv1beta1.CloudVendorIBM:
		// IBM Cloud Platform
		platform = IBMPlatformType
	case clusterv1beta1.CloudVendorAWS:
		// AWS
		platform = AWSPlatformType
	case clusterv1beta1.CloudVendorGoogle:
		// Google Cloud Platform
		platform = GCPPlatformType
	case clusterv1beta1.CloudVendorAzure:
		// Azure
		platform = string(clusterv1beta1.CloudVendorAzure)
		if !isOpenShift {
			// Azure + AKS
			product = string(clusterv1beta1.KubeVendorAKS)
		}
	case clusterv1beta1.CloudVendorOpenStack:
		// OpenStack
		platform = string(clusterv1beta1.CloudVendorOpenStack)
	case clusterv1beta1.CloudVendorVSphere:
		// VSphere
		platform = string(clusterv1beta1.CloudVendorVSphere)
	}

	return kubeVersion, platform, product
}

func newClusterClaim(name, value string) *clusterv1alpha1.ClusterClaim {
	return &clusterv1alpha1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: clusterv1alpha1.ClusterClaimSpec{
			Value: value,
		},
	}
}
