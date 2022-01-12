package clusterclaim

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	clusterv1beta1 "github.com/stolostron/multicloud-operators-foundation/pkg/apis/internal.open-cluster-management.io/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClaimK8sID                   = "id.k8s.io"
	ClaimOpenshiftID             = "id.openshift.io"
	ClaimOpenshiftVersion        = "version.openshift.io"
	ClaimOpenshiftInfrastructure = "infrastructure.openshift.io"

	// ClaimControlPlaneTopology expresses the expectations for operands that normally run on control nodes of Openshift.
	// have 2 modes: `HighlyAvailable` and `SingleReplica`.
	ClaimControlPlaneTopology = "controlplanetopology.openshift.io"

	ClaimOCMConsoleURL  = "consoleurl.cluster.open-cluster-management.io"
	ClaimOCMRegion      = "region.open-cluster-management.io"
	ClaimOCMKubeVersion = "kubeversion.open-cluster-management.io"
	ClaimOCMPlatform    = "platform.open-cluster-management.io"
	ClaimOCMProduct     = "product.open-cluster-management.io"
)

const labelCustomizedOnly = "open-cluster-management.io/spoke-only"
const labelHubManaged = "open-cluster-management.io/hub-managed"

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
	ProductROSA      = "ROSA"
	ProductARO       = "ARO"

	// ProductOther other (unable to auto detect)
	ProductOther = "Other"
)

// internalLabels includes the labels managed by ACM.
var internalLabels = sets.String{}

func init() {
	internalLabels.Insert(clusterv1beta1.LabelClusterID,
		clusterv1beta1.LabelCloudVendor,
		clusterv1beta1.LabelKubeVendor,
		clusterv1beta1.LabelManagedBy,
		clusterv1beta1.OCPVersion)
}

type ClusterClaimer struct {
	ClusterName    string
	Product        string
	Platform       string
	HubClient      client.Client
	KubeClient     kubernetes.Interface
	ConfigV1Client openshiftclientset.Interface
	Mapper         meta.RESTMapper
}

func (c *ClusterClaimer) List() ([]*clusterv1alpha1.ClusterClaim, error) {
	var claims []*clusterv1alpha1.ClusterClaim
	err := c.updatePlatformProduct()
	if err != nil {
		klog.Errorf("failed to update platform and product. err:%v", err)
		return claims, err
	}

	claims = append(claims, newClusterClaim(ClaimOCMPlatform, c.Platform))
	claims = append(claims, newClusterClaim(ClaimOCMProduct, c.Product))
	claims = append(claims, newClusterClaim(ClaimK8sID, c.ClusterName))

	version, clusterID, err := c.getOCPVersion()
	if err != nil {
		klog.Errorf("failed to get OCP version and clusterID, error: %v ", err)
		return claims, err
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
		return claims, err
	}
	if infraConfig != "" {
		claims = append(claims, newClusterClaim(ClaimOpenshiftInfrastructure, infraConfig))
	}

	_, _, consoleURL, err := c.getMasterAddresses()
	if err != nil {
		klog.Errorf("failed to get master Addresses, error: %v ", err)
		return claims, err
	}
	if consoleURL != "" {
		claims = append(claims, newClusterClaim(ClaimOCMConsoleURL, consoleURL))
	}

	kubeVersion, err := c.getKubeVersion()
	if err != nil {
		klog.Errorf("failed to get kubeVersion: %v", err)
		return claims, err
	}
	claims = append(claims, newClusterClaim(ClaimOCMKubeVersion, kubeVersion))

	region, err := c.getClusterRegion()
	if err != nil {
		klog.Errorf("failed to get region, error: %v ", err)
		return claims, err
	}
	if region != "" {
		claims = append(claims, newClusterClaim(ClaimOCMRegion, region))
	}

	controlPlaneTopology := c.getControlPlaneTopology()
	if controlPlaneTopology != "" {
		claims = append(claims, newClusterClaim(ClaimControlPlaneTopology, string(controlPlaneTopology)))
	}

	syncedClaims, err := c.syncLabelsToClaims()
	if err != nil {
		klog.Errorf("failed to sync labels to claims: %v", err)
		return claims, err
	}
	if len(syncedClaims) != 0 {
		claims = append(claims, syncedClaims...)
	}
	return claims, nil
}

func newClusterClaim(name, value string) *clusterv1alpha1.ClusterClaim {
	return &clusterv1alpha1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{labelHubManaged: ""},
		},
		Spec: clusterv1alpha1.ClusterClaimSpec{
			Value: value,
		},
	}
}

func (c *ClusterClaimer) isOpenShift() (bool, error) {
	_, err := c.Mapper.RESTMapping(schema.GroupKind{Group: "project.openshift.io", Kind: "Project"}, "v1")
	if err != nil {
		if meta.IsNoMatchError(err) {
			return false, nil
		}
		klog.Errorf("failed to mapping project:%v", err)
		return false, err
	}

	return true, nil
}

func (c *ClusterClaimer) isOpenshiftDedicated() (bool, error) {
	// this API group is created for the Openshift Dedicated platform to manage various permissions.
	// defined in https://github.com/openshift/rbac-permissions-operator/blob/master/pkg/apis/managed/v1alpha1/subjectpermission_types.go
	_, err := c.Mapper.RESTMapping(schema.GroupKind{Group: "managed.openshift.io", Kind: "SubjectPermission"}, "v1alpha1")
	if err != nil {
		if meta.IsNoMatchError(err) {
			return false, nil
		}
		klog.Errorf("failed to mapping SubjectPermission:%v", err)
		return false, err
	}
	return true, nil
}

func (c *ClusterClaimer) isROSA() (bool, error) {
	_, err := c.KubeClient.CoreV1().ConfigMaps("openshift-config").Get(context.TODO(), "rosa-brand-logo", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		klog.Errorf("failed to get configmap openshift-config/rosa-brand-logo. err: %v", err)
		return false, err
	}

	return true, nil
}

func (c *ClusterClaimer) isARO() (bool, error) {
	_, err := c.Mapper.RESTMapping(schema.GroupKind{Group: "aro.openshift.io", Kind: "Cluster"}, "v1alpha1")
	if err != nil {
		if meta.IsNoMatchError(err) {
			return false, nil
		}
		klog.Errorf("failed to mapping project:%v", err)
		return false, err
	}
	return true, nil
}

const (
	OCP3Version = "3"
)

func (c *ClusterClaimer) getOCPVersion() (version, clusterID string, err error) {
	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if the cluster is openshift.err:%v", err)
		return "", "", err
	}
	if !isOpenShift {
		return "", "", nil
	}

	clusterVersion, err := c.ConfigV1Client.ConfigV1().ClusterVersions().Get(context.TODO(), "version", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			version = OCP3Version
			return version, "", nil
		}
		klog.Errorf("failed to get OCP cluster version: %v", err)
		return "", "", err
	}

	clusterID = string(clusterVersion.Spec.ClusterID)
	historyItems := clusterVersion.Status.History
	for _, historyItem := range historyItems {
		if historyItem.State == "Completed" {
			version = historyItem.Version
			break
		}
	}

	return version, clusterID, nil
}

type InfraConfig struct {
	InfraName string `json:"infraName,omitempty"`
}

func (c *ClusterClaimer) getInfraConfig() (string, error) {
	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if the cluster is openshift.err:%v", err)
		return "", err
	}
	if !isOpenShift {
		return "", nil
	}

	infrastructure, err := c.ConfigV1Client.ConfigV1().Infrastructures().Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}

		klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
		return "", err
	}

	infraConfig := InfraConfig{
		InfraName: infrastructure.Status.InfrastructureName,
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

	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if the cluster is openshift.err:%v", err)
		return "", err
	}

	switch {
	case isOpenShift:
		infrastructure, err := c.ConfigV1Client.ConfigV1().Infrastructures().Get(context.TODO(), "cluster", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return "", nil
			}
			klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
			return "", err
		}
		platformType := infrastructure.Status.PlatformStatus.Type

		// only ocp on aws and gcp has region definition
		// refer to https://github.com/openshift/api/blob/master/config/v1/types_infrastructure.go
		switch platformType {
		case PlatformAWS:
			region = infrastructure.Status.PlatformStatus.AWS.Region
		case PlatformGCP:
			region = infrastructure.Status.PlatformStatus.GCP.Region
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
	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if the cluster is openshift.err:%v", err)
		return masterAddresses, masterPorts, "", err
	}
	if isOpenShift {
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

func (c *ClusterClaimer) updatePlatformProduct() (err error) {
	// platform and product are constant, update only when they are empty.
	if c.Platform != "" && c.Product != "" {
		return nil
	}

	c.Product = ProductOther
	c.Platform = PlatformOther

	kubeVersion, err := c.getKubeVersion()
	if err != nil {
		klog.Errorf("failed to get kubeVersion: %v", err)
		return err
	}
	gitVersion := strings.ToUpper(kubeVersion)

	// deal with product and platform
	switch {
	case strings.Contains(gitVersion, ProductIKS):
		c.Platform = PlatformIBM
		c.Product = ProductIKS
		return nil
	case strings.Contains(gitVersion, ProductICP):
		c.Platform = PlatformIBM
		c.Product = ProductICP
		return nil
	case strings.Contains(gitVersion, ProductEKS):
		c.Platform = PlatformAWS
		c.Product = ProductEKS
		return nil
	case strings.Contains(gitVersion, ProductGKE):
		c.Platform = PlatformGCP
		c.Product = ProductGKE
		return nil
	}

	isROSA, err := c.isROSA()
	if err != nil {
		klog.Errorf("failed to check if cluster is ROSA. %v", err)
		return err
	}
	if isROSA {
		c.Platform = PlatformAWS
		c.Product = ProductROSA
		return nil
	}

	isARO, err := c.isARO()
	if err != nil {
		klog.Errorf("failed to check if cluster is ARO. %v", err)
		return err
	}
	if isARO {
		c.Platform = PlatformAzure
		c.Product = ProductARO
		return nil
	}

	// OSD is also openshift, should check openshift firstly.
	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if cluster is openshift. %v", err)
		return err
	}
	if isOpenShift {
		c.Product = ProductOpenShift
	}

	isOpenshiftDedicated, err := c.isOpenshiftDedicated()
	if err != nil {
		klog.Errorf("failed to check if cluster is OSD. %v", err)
		return err
	}
	if isOpenshiftDedicated {
		c.Product = ProductOSD
	}

	var architecture, providerID string
	if architecture, providerID, err = c.getNodeInfo(); err != nil {
		klog.Errorf("failed to get node info: %v", err)
		return err
	}

	switch {
	case architecture == "s390x":
		c.Platform = PlatformIBMZ
	case architecture == "ppc64le":
		c.Platform = PlatformIBMP
	case strings.HasPrefix(providerID, "ibm"):
		c.Platform = PlatformIBM
	case strings.HasPrefix(providerID, "azure"):
		c.Platform = PlatformAzure
		if c.Product == ProductOther {
			c.Product = ProductAKS
		}
	case strings.HasPrefix(providerID, "aws"):
		c.Platform = PlatformAWS
	case strings.HasPrefix(providerID, "gce"):
		c.Platform = PlatformGCP
	case strings.HasPrefix(providerID, "vsphere"):
		c.Platform = PlatformVSphere
	case strings.HasPrefix(providerID, "openstack"):
		c.Platform = PlatformOpenStack
	}

	return nil
}

func (c *ClusterClaimer) getControlPlaneTopology() configv1.TopologyMode {
	infra, err := c.ConfigV1Client.ConfigV1().Infrastructures().Get(context.TODO(), "cluster", metav1.GetOptions{})
	if err != nil {
		return ""
	}

	return infra.Status.ControlPlaneTopology
}

func (c *ClusterClaimer) syncLabelsToClaims() ([]*clusterv1alpha1.ClusterClaim, error) {
	var claims []*clusterv1alpha1.ClusterClaim
	request := types.NamespacedName{Namespace: c.ClusterName, Name: c.ClusterName}
	clusterInfo := &clusterv1beta1.ManagedClusterInfo{}
	err := c.HubClient.Get(context.TODO(), request, clusterInfo)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return claims, nil
		}
		return claims, err
	}

	// do not create claim if the label is managed by ACM.
	// if the label format is aaa/bbb, the name of claim will be bbb.aaa.
	for label, value := range clusterInfo.Labels {
		if internalLabels.Has(label) || strings.Contains(label, "open-cluster-management.io") {
			continue
		}

		newLabel := label
		subs := strings.Split(label, "/")
		if len(subs) == 2 {
			newLabel = fmt.Sprintf("%s.%s", subs[1], subs[0])
		} else if len(subs) > 2 {
			newLabel = strings.ReplaceAll(label, "/", ".")
		}

		claim := newClusterClaim(newLabel, value)
		if claim.Labels != nil {
			claim.Labels[labelCustomizedOnly] = ""
		}
		claims = append(claims, claim)
	}
	return claims, nil
}
