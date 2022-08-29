package clusterclaim

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"k8s.io/apimachinery/pkg/util/uuid"

	configv1 "github.com/openshift/api/config/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	openshiftoauthclientset "github.com/openshift/client-go/oauth/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClaimK8sID                      = "id.k8s.io"
	ClaimOpenshiftID                = "id.openshift.io"
	ClaimOpenshiftVersion           = "version.openshift.io"
	ClaimOpenshiftInfrastructure    = "infrastructure.openshift.io"
	ClaimOpenshiftOauthRedirectURIs = "oauthredirecturis.openshift.io"
	// ClaimControlPlaneTopology expresses the expectations for operands that normally run on control nodes of Openshift.
	// have 2 modes: `HighlyAvailable` and `SingleReplica`.
	ClaimControlPlaneTopology = "controlplanetopology.openshift.io"

	ClaimOCMConsoleURL  = "consoleurl.cluster.open-cluster-management.io"
	ClaimOCMRegion      = "region.open-cluster-management.io"
	ClaimOCMKubeVersion = "kubeversion.open-cluster-management.io"
	ClaimOCMPlatform    = "platform.open-cluster-management.io"
	ClaimOCMProduct     = "product.open-cluster-management.io"
)

const (
	labelCustomizedOnly = "open-cluster-management.io/spoke-only"
	labelHubManaged     = "open-cluster-management.io/hub-managed"

	// labelExcludeBackup is true for the local-cluster will be not backed up into velero
	labelExcludeBackup = "velero.io/exclude-from-backup"
)

// clusterClaimCreateOnlyList returns a list of cluster claims that only need to create, not update.
var clusterClaimCreateOnlyList = []string{
	ClaimK8sID,
}

// should be the type defined in infrastructure.config.openshift.io
const (
	PlatformAWS          = "AWS"
	PlatformGCP          = "GCP"
	PlatformAzure        = "Azure"
	PlatformIBM          = "IBM"
	PlatformIBMP         = "IBMPowerPlatform"
	PlatformIBMZ         = "IBMZPlatform"
	PlatformOpenStack    = "OpenStack"
	PlatformVSphere      = "VSphere"
	PlatformRHV          = "RHV"
	PlatformAlibabaCloud = "AlibabaCloud"
	PlatformBareMetal    = "BareMetal"
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
	ProductROKS      = "ROKS"

	// ProductOther other (unable to auto detect)
	ProductOther = "Other"
)

// ProductOCPList is OCP product list. should append the product to the list if the product is OCP.
var ProductOCPList = []string{
	ProductOpenShift,
	ProductOSD,
	ProductROSA,
	ProductARO,
	ProductROKS,
}

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
	ClusterName                     string
	Product                         string
	Platform                        string
	HubClient                       client.Client
	KubeClient                      kubernetes.Interface
	ConfigV1Client                  openshiftclientset.Interface
	OauthV1Client                   openshiftoauthclientset.Interface
	Mapper                          meta.RESTMapper
	managedclusterID                string
	EnableSyncLabelsToClusterClaims bool
}

// getManagedClusterID returns the managed cluster ID of the cluster.
// We get ID by the sequence of: openshiftID, uid of kube-system namespace, random UID.
// Note that: even we use randomID to generate the managed cluster ID, it would change because `id.k8s.io` is create only now.
func (c *ClusterClaimer) getManagedClusterID() (string, error) {
	if c.managedclusterID != "" {
		return c.managedclusterID, nil
	}

	isocp, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if the cluster is openshift.err:%v", err)
		return "", err
	}
	if isocp {
		_, ocpID, err := c.getOCPVersion()
		if err == nil {
			return ocpID, nil
		}
		klog.Errorf("Get ocpID failed, %v", err)
	}

	ns, err := c.KubeClient.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err == nil {
		return string(ns.UID), nil
	}
	klog.Errorf("Get kube-system namespace failed, %v", err)

	return string(uuid.NewUUID()), nil
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

	managedClusterID, err := c.getManagedClusterID()
	if err != nil {
		return nil, err
	}
	claims = append(claims, newClusterClaim(ClaimK8sID, managedClusterID))

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

	redirectURIs, err := c.getOCPOauthRedirectURIs()
	if err != nil {
		return claims, err
	}
	if redirectURIs != "" {
		claims = append(claims, newClusterClaim(ClaimOpenshiftOauthRedirectURIs, redirectURIs))
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

	if c.EnableSyncLabelsToClusterClaims {
		syncedClaims, err := c.syncLabelsToClaims()
		if err != nil {
			klog.Errorf("failed to sync labels to claims: %v", err)
			return claims, err
		}
		if len(syncedClaims) != 0 {
			claims = append(claims, syncedClaims...)
		}
	}

	return claims, nil
}

func newClusterClaim(name, value string) *clusterv1alpha1.ClusterClaim {
	return &clusterv1alpha1.ClusterClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{labelHubManaged: "", labelExcludeBackup: "true"},
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

func (c *ClusterClaimer) getOCPOauthRedirectURIs() (string, error) {
	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if the cluster is openshift.err:%v", err)
		return "", err
	}
	if !isOpenShift {
		return "", nil
	}

	oauthclient, err := c.OauthV1Client.OauthV1().OAuthClients().Get(
		context.TODO(), "openshift-challenging-client", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}

		klog.Errorf("failed to get OCP cluster oauth redirect URIs: %v", err)
		return "", err
	}

	return strings.Join(oauthclient.RedirectURIs, ","), nil
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

	if region != "" {
		return region, nil
	}

	nodes, err := c.KubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", client.IgnoreNotFound(err)
	}

	regionList := sets.NewString()
	for _, node := range nodes.Items {
		labels := node.GetLabels()
		if regionValue, ok := labels[corev1.LabelTopologyRegion]; ok {
			regionList.Insert(regionValue)
			continue
		}
		if regionValue, ok := labels[corev1.LabelFailureDomainBetaRegion]; ok {
			regionList.Insert(regionValue)
		}
	}

	return strings.Join(regionList.List(), ","), nil
}

// for OpenShift, read endpoint address from console-config in openshift-console
// note: SNO has no console
func (c *ClusterClaimer) getMasterAddressesFromConsoleConfig() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, error) {
	var masterAddresses []corev1.EndpointAddress
	var masterPorts []corev1.EndpointPort
	clusterURL := ""

	cfg, err := c.KubeClient.CoreV1().ConfigMaps("openshift-console").Get(context.TODO(), "console-config", metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return masterAddresses, masterPorts, clusterURL, nil
	}
	if err != nil {
		return masterAddresses, masterPorts, clusterURL, err
	}
	if cfg.Data == nil {
		return masterAddresses, masterPorts, clusterURL, nil
	}

	consoleConfigString, ok := cfg.Data["console-config.yaml"]
	if !ok {
		return masterAddresses, masterPorts, clusterURL, nil
	}

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

	return masterAddresses, masterPorts, clusterURL, nil
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
	// platform and product are constant, update only when they are empty or Other.
	// there is a case cannot get project api using RESTMapping during the OCP is failed to upgrade. in this case the OCP
	// is detected Other, we need to retry to detect the product type.
	if c.Platform != "" && c.Platform != PlatformOther && c.Product != "" && c.Product != ProductOther {
		return nil
	}

	platform, product, err := c.getPlatformProduct()
	if err != nil {
		return err
	}
	c.Product = product
	c.Platform = platform
	return
}

func (c *ClusterClaimer) getPlatformProduct() (string, string, error) {
	kubeVersion, err := c.getKubeVersion()
	if err != nil {
		klog.Errorf("failed to get kubeVersion: %v", err)
		return "", "", err
	}
	gitVersion := strings.ToUpper(kubeVersion)

	// deal with product and platform
	switch {
	case strings.Contains(gitVersion, ProductIKS):
		return PlatformIBM, ProductIKS, nil
	case strings.Contains(gitVersion, ProductICP):
		return PlatformIBM, ProductICP, nil
	case strings.Contains(gitVersion, ProductEKS):
		return PlatformAWS, ProductEKS, nil
	case strings.Contains(gitVersion, ProductGKE):
		return PlatformGCP, ProductGKE, nil
	}

	isROSA, err := c.isROSA()
	if err != nil {
		klog.Errorf("failed to check if cluster is ROSA. %v", err)
		return "", "", err
	}
	if isROSA {
		return PlatformAWS, ProductROSA, nil
	}

	isARO, err := c.isARO()
	if err != nil {
		klog.Errorf("failed to check if cluster is ARO. %v", err)
		return "", "", err
	}
	if isARO {
		return PlatformAzure, ProductARO, nil
	}

	product := ProductOther
	platform := PlatformOther
	// OSD is also openshift, should check openshift firstly.
	isOpenShift, err := c.isOpenShift()
	if err != nil {
		klog.Errorf("failed to check if cluster is openshift. %v", err)
		return "", "", err
	}
	if isOpenShift {
		product = ProductOpenShift
	}

	isOpenshiftDedicated, err := c.isOpenshiftDedicated()
	if err != nil {
		klog.Errorf("failed to check if cluster is OSD. %v", err)
		return "", "", err
	}
	if isOpenshiftDedicated {
		product = ProductOSD
	}

	if isOpenShift {
		infrastructure, err := c.ConfigV1Client.ConfigV1().Infrastructures().Get(context.TODO(), "cluster", metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			klog.Errorf("failed to get OCP infrastructures.config.openshift.io/cluster: %v", err)
			return "", "", err
		}
		if err == nil {
			platformType := infrastructure.Status.PlatformStatus.Type
			switch platformType {
			case configv1.AWSPlatformType:
				return PlatformAWS, product, nil
			case configv1.AzurePlatformType:
				return PlatformAzure, product, nil
			case configv1.AlibabaCloudPlatformType:
				return PlatformAlibabaCloud, product, nil
			case configv1.OvirtPlatformType:
				return PlatformRHV, product, nil
			case configv1.GCPPlatformType:
				return PlatformGCP, product, nil
			case configv1.VSpherePlatformType:
				return PlatformVSphere, product, nil
			case configv1.OpenStackPlatformType:
				return PlatformOpenStack, product, nil
			case configv1.BareMetalPlatformType:
				return PlatformBareMetal, product, nil
			case configv1.IBMCloudPlatformType:
				return PlatformIBM, ProductROKS, nil
			}
		}
	}

	var architecture, providerID string
	if architecture, providerID, err = c.getNodeInfo(); err != nil {
		klog.Errorf("failed to get node info: %v", err)
		return "", "", err
	}

	switch {
	case architecture == "s390x":
		platform = PlatformIBMZ
	case architecture == "ppc64le":
		platform = PlatformIBMP
	case strings.HasPrefix(providerID, "ibm"):
		platform = PlatformIBM
	case strings.HasPrefix(providerID, "azure"):
		platform = PlatformAzure
		if product == ProductOther {
			return PlatformAzure, ProductAKS, nil
		}
	case strings.HasPrefix(providerID, "aws"):
		platform = PlatformAWS
	case strings.HasPrefix(providerID, "gce"):
		platform = PlatformGCP
	case strings.HasPrefix(providerID, "vsphere"):
		platform = PlatformVSphere
	case strings.HasPrefix(providerID, "openstack"):
		platform = PlatformOpenStack
	}

	return platform, product, nil
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
	// Besides, "_" and "/" in the label name will be replaced with "-" and "." respectively.
	for label, value := range clusterInfo.Labels {
		if internalLabels.Has(label) || strings.Contains(label, "open-cluster-management.io") {
			continue
		}

		// convert the string to lower case
		name := strings.ToLower(label)

		// and then replace invalid characters
		subs := strings.Split(name, "/")
		if len(subs) == 2 {
			name = fmt.Sprintf("%s.%s", subs[1], subs[0])
		} else if len(subs) > 2 {
			name = strings.ReplaceAll(name, "/", ".")
		}
		name = strings.ReplaceAll(name, "_", "-")

		// ignore the label if the transformed name is still not a valid resource name
		if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
			klog.V(4).Infof("skip syncing label %q of ManagedClusterInfo to ClusterCliam because it's an invalid resource name", label)
			continue
		}

		// ignore the label if its value is empty. (the value of ClusterCliam can not be empty)
		if len(value) == 0 {
			klog.V(4).Infof("skip syncing label %q of ManagedClusterInfo to ClusterCliam because its value is empty.", label)
			continue
		}

		claim := newClusterClaim(name, value)
		if claim.Labels != nil {
			claim.Labels[labelCustomizedOnly] = ""
		}
		claims = append(claims, claim)
	}
	return claims, nil
}
