package clusterclaim

import (
	"context"
	"encoding/json"
	"fmt"
	clusterv1alpha1 "github.com/open-cluster-management/api/cluster/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"strconv"
	"strings"
)

const (
	ClaimK8sID                   = "id.k8s.io"
	ClaimOpenshiftID             = "id.openshift.io"
	ClaimOpenshiftVersion        = "version.openshift.io"
	ClaimOpenshiftInfrastructure = "infrastructure.openshift.io"
	ClaimOCMConsoleURL           = "consoleurl.cluster.open-cluster-management.io"
	ClaimOCMRegion               = "region.open-cluster-management.io"
	ClaimOCMKubeVersion          = "kubeversion.open-cluster-management.io"
	ClaimOCMPlatform             = "platform.open-cluster-management.io"
	ClaimOCMProduct              = "product.open-cluster-management.io"
)

// should be the type defined in infrastructure.config.openshift.io
const (
	PlatformAWS       = "AWS"
	PlatformGCP       = "GCP"
	PlatformAzure     = "Azure"
	PlatformIBM       = "IBM"
	PlatformIBMP      = "IBMPowerPlatform"
	PlatformIBMZ      = "IBMZPlatform"
	PlatformOpenStack = "OpenStack"
	PlatformVSphere   = "VSphere"
	PlatformBareMetal = "BareMetal"
	// PlatformOther other (unable to auto detect)
	PlatformOther = "Other"
)

const (
	ProductAKS       = "AKS"
	ProductEKS       = "EKS"
	ProductGKE       = "GKE"
	ProductICP       = "ICP"
	ProductIKS       = "IKS"
	ProductOpenShift = "OpenShift"
	ProductOSD       = "OpenShiftDedicated"
	// ProductOther other (unable to auto detect)
	ProductOther = "Other"
)

type ClusterClaimer struct {
	ClusterName   string
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

func (c *ClusterClaimer) List() ([]*clusterv1alpha1.ClusterClaim, error) {
	var claims []*clusterv1alpha1.ClusterClaim

	version, clusterID, err := c.getOCPVersion()
	if err != nil {
		klog.Errorf("failed to get OCP version and clusterID, error: %v ", err)
	}
	if clusterID != "" {
		claims = append(claims, newClusterClaim(ClaimOpenshiftID, clusterID))
	}
	if version != "" {
		claims = append(claims, newClusterClaim(ClaimOpenshiftVersion, version))
	}

	infraConfig, err := c.getInfraConfig()
	if err != nil {
		klog.Errorf("failed to get infraConfig, error: %v ", err)
	}
	if infraConfig != "" {
		claims = append(claims, newClusterClaim(ClaimOpenshiftInfrastructure, infraConfig))
	}

	_, _, consoleURL, err := c.getMasterAddresses()
	if err != nil {
		klog.Errorf("failed to get master Addresses, error: %v ", err)
	}
	if consoleURL != "" {
		claims = append(claims, newClusterClaim(ClaimOCMConsoleURL, consoleURL))
	}

	claims = append(claims, newClusterClaim(ClaimK8sID, c.ClusterName))

	kubeVersion, platform, product, err := c.getKubeVersionPlatformProduct()
	if err != nil {
		klog.Errorf("failed to get kubeVersion/platform/product: %v", err)
	}
	claims = append(claims, newClusterClaim(ClaimOCMKubeVersion, kubeVersion))
	if len(platform) > 0 {
		claims = append(claims, newClusterClaim(ClaimOCMPlatform, platform))
	}
	if len(product) > 0 {
		claims = append(claims, newClusterClaim(ClaimOCMProduct, product))
	}

	region, err := c.getClusterRegion()
	if err != nil {
		klog.Errorf("failed to get region, error: %v ", err)
	}
	if region != "" {
		claims = append(claims, newClusterClaim(ClaimOCMRegion, region))
	}

	return claims, nil
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

func (c *ClusterClaimer) isOpenShift() bool {
	serverGroups, err := c.KubeClient.Discovery().ServerGroups()
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

func (c *ClusterClaimer) isOpenshiftDedicated() bool {
	hasProject := false
	hasManaged := false
	serverGroups, err := c.KubeClient.Discovery().ServerGroups()
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

func (c *ClusterClaimer) getOCPVersion() (version, clusterID string, err error) {
	if !c.isOpenShift() {
		return "", "", nil
	}

	obj, err := c.DynamicClient.Resource(ocpVersionGVR).Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			version = OCP3Version
			return version, "", nil
		}
		klog.Errorf("failed to get OCP cluster version: %v", err)
		return "", "", err
	}

	clusterID, _, err = unstructured.NestedString(obj.Object, "spec", "clusterID")
	if err != nil {
		klog.Errorf("failed to get OCP clusterID in spec: %v", err)
		return "", "", err
	}

	historyItems, _, err := unstructured.NestedSlice(obj.Object, "status", "history")
	if err != nil {
		klog.Errorf("failed to get OCP cluster version in history of status: %v", err)
		return "", "", err
	}

	for _, historyItem := range historyItems {
		state, _, err := unstructured.NestedString(historyItem.(map[string]interface{}), "state")
		if err != nil {
			klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			continue
		}
		if state == "Completed" {
			version, _, err = unstructured.NestedString(historyItem.(map[string]interface{}), "version")
			if err != nil {
				klog.Errorf("failed to get OCP cluster version in latest history of status: %v", err)
			}
			break
		}
	}

	return version, clusterID, nil
}

var ocpInfrGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "infrastructures",
}

type InfraConfig struct {
	InfraName string `json:"infraName,omitempty"`
}

func (c *ClusterClaimer) getInfraConfig() (string, error) {
	if !c.isOpenShift() {
		return "", nil
	}

	obj, err := c.DynamicClient.Resource(ocpInfrGVR).Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
		return "", err
	}

	infraConfig := InfraConfig{}
	infraConfig.InfraName, _, err = unstructured.NestedString(obj.Object, "status", "infrastructureName")
	if err != nil {
		klog.Errorf("failed to get OCP infrastructure Name in status of infrastructures.config.openshift.io/cluster: %v", err)
		return "", err
	}

	infraConfigRaw, err := json.Marshal(infraConfig)
	if err != nil {
		klog.Errorf("failed to marshal infraConfig: %v", err)
		return "", err
	}
	return string(infraConfigRaw), nil
}

func (c *ClusterClaimer) getClusterRegion() (string, error) {
	var region = ""

	switch {
	case c.isOpenShift():
		obj, err := c.DynamicClient.Resource(ocpInfrGVR).Get(context.TODO(), "cluster", metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
			return "", err
		}

		platformType, _, err := unstructured.NestedString(obj.Object, "status", "platform")
		if err != nil {
			klog.Errorf("failed to get OCP platform type in status of infrastructures.config.openshift.io/cluster: %v", err)
			return "", err
		}

		// only ocp on aws and gcp has region definition
		// refer to https://github.com/openshift/api/blob/master/config/v1/types_infrastructure.go
		switch platformType {
		case PlatformAWS:
			region, _, err = unstructured.NestedString(obj.Object, "status", "platformStatus", "aws", "region")
			if err != nil {
				klog.Errorf("failed to get OCP region in status of infrastructures.config.openshift.io/cluster: %v", err)
				return "", err
			}
		case PlatformGCP:
			region, _, err = unstructured.NestedString(obj.Object, "status", "platformStatus", "gcp", "region")
			if err != nil {
				klog.Errorf("failed to get OCP region in status of infrastructures.config.openshift.io/cluster: %v", err)
				return "", err
			}
		}
	}

	return region, nil
}

// for OpenShift, read endpoint address from console-config in openshift-console
func (c *ClusterClaimer) getMasterAddressesFromConsoleConfig() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, error) {
	var masterAddresses []corev1.EndpointAddress
	var masterPorts []corev1.EndpointPort
	clusterURL := ""

	cfg, err := c.KubeClient.CoreV1().ConfigMaps("openshift-console").Get(context.TODO(), "console-config", metav1.GetOptions{})
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
				euArray := strings.Split(strings.Trim(eu, "https:/"), ":")
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
	return masterAddresses, masterPorts, clusterURL, err
}

func (c *ClusterClaimer) getMasterAddresses() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, error) {
	var masterAddresses []corev1.EndpointAddress
	var masterPorts []corev1.EndpointPort
	var err error

	if c.isOpenShift() {
		return c.getMasterAddressesFromConsoleConfig()
	}

	kubeEndpoints, err := c.KubeClient.CoreV1().Endpoints("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err == nil && len(kubeEndpoints.Subsets) > 0 {
		masterAddresses = kubeEndpoints.Subsets[0].Addresses
		masterPorts = kubeEndpoints.Subsets[0].Ports
	}

	return masterAddresses, masterPorts, "", err
}

func (c *ClusterClaimer) getKubeVersion() (string, error) {
	serverVersion, err := c.KubeClient.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}

	return serverVersion.String(), nil
}

func (c *ClusterClaimer) getNodeInfo() (architecture, providerID string, err error) {
	nodes, err := c.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	if len(nodes.Items) == 0 {
		return "", "", fmt.Errorf("failed to get nodes list, the count of nodes is 0")
	}

	architecture = nodes.Items[0].Status.NodeInfo.Architecture
	providerID = nodes.Items[0].Spec.ProviderID

	return
}

func (c *ClusterClaimer) getKubeVersionPlatformProduct() (kubeVersion, platform, product string, err error) {
	product = ProductOther
	platform = PlatformOther

	if kubeVersion, err = c.getKubeVersion(); err != nil {
		klog.Errorf("failed to get kubeVersion: %v", err)
		return
	}
	gitVersion := strings.ToUpper(kubeVersion)
	// deal with product and platform
	switch {
	case strings.Contains(gitVersion, ProductIKS):
		platform = PlatformIBM
		product = ProductIKS
		return
	case strings.Contains(gitVersion, ProductICP):
		platform = PlatformIBM
		product = ProductICP
		return
	case strings.Contains(gitVersion, ProductEKS):
		platform = PlatformAWS
		product = ProductEKS
		return
	case strings.Contains(gitVersion, ProductGKE):
		platform = PlatformGCP
		product = ProductGKE
		return
	case c.isOpenshiftDedicated():
		product = ProductOSD
	case c.isOpenShift():
		product = ProductOpenShift
	}

	var architecture, providerID string
	if architecture, providerID, err = c.getNodeInfo(); err != nil {
		klog.Errorf("failed to get node info: %v", err)
		return
	}

	switch {
	case architecture == "s390x":
		platform = PlatformIBMZ
	case architecture == "ppc64le":
		platform = PlatformIBMP
	case strings.Contains(providerID, "ibm"):
		platform = PlatformIBM
	case strings.Contains(providerID, "azure"):
		platform = PlatformAzure
		if product == ProductOther {
			product = ProductAKS
		}
	case strings.Contains(providerID, "aws"):
		platform = PlatformAWS
	case strings.Contains(providerID, "gce"):
		platform = PlatformGCP
	case strings.Contains(providerID, "vsphere"):
		platform = PlatformVSphere
	case strings.Contains(providerID, "openstack"):
		platform = PlatformOpenStack
	case strings.Contains(providerID, "baremetal"):
		platform = PlatformBareMetal
	}

	return kubeVersion, platform, product, nil
}
