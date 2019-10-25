// Licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
// IBM Confidential
// OCO Source Materials
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.

// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2018. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package helm

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"

	"k8s.io/klog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/tlsutil"
)

//TODO should not use const var
const HubLocalRepoCertDir = "/tmp/hcm/repo"
const HubLocalRepoCertFileName = "hubClusterCa.ca"
const IsHubClusterLocalRepo = "mcm.ibm.com/hub-cluster-repo"

var (
	tillerLabels    = labels.Set{"app": "helm", "name": "tiller"}
	tillerNamespace = "kube-system"
)

// HelmControl implements helm interface
type HelmControl struct {
	helmclient   helm.Interface
	hcmclientset clientset.Interface
}

//HelmControlInterface is an interface for helm releases
type HelmControlInterface interface {
	//CreateHelmRelease create a helm release
	CreateHelmRelease(
		releaseName string,
		helmreponNamespace string,
		helmspec v1alpha1.HelmWorkSpec) (*rls.InstallReleaseResponse, error)
	//UpdateHelmRelease update a helm release
	UpdateHelmRelease(
		releaseName string,
		helmreponNamespace string,
		helmspec v1alpha1.HelmWorkSpec) (*rls.UpdateReleaseResponse, error)
	//GetHelmReleases get helm releases
	GetHelmReleases(
		nameFilter string,
		codes []release.Status_Code,
		namespace string, limit int) (*rls.ListReleasesResponse, error)
	//DeleteHelmRelease delete a helm release
	DeleteHelmRelease(relName string) (*rls.UninstallReleaseResponse, error)
}

// NewHelmControl creates a new helm control
func NewHelmControl(helmclient helm.Interface, hcmclientset clientset.Interface) HelmControlInterface {
	if helmclient == nil {
		return nil
	}
	return &HelmControl{
		helmclient:   helmclient,
		hcmclientset: hcmclientset,
	}
}

func searchTillerPodIP(client kubernetes.Interface) (string, error) {
	options := metav1.ListOptions{LabelSelector: tillerLabels.AsSelector().String()}
	pods, err := client.CoreV1().Pods(tillerNamespace).List(options)
	if err != nil {
		return "", err
	}

	if len(pods.Items) < 1 {
		return "", fmt.Errorf("Failed to find tiller service")
	}

	for _, item := range pods.Items {
		cSS := item.Status.ContainerStatuses
		healthy := true
		for _, cS := range cSS {
			state := cS.State
			if state.Running == nil {
				healthy = false
			}
		}

		if healthy {
			return item.Status.HostIP, nil
		}
	}

	return "", fmt.Errorf("Failed to find working tiller")
}

func searchTillerEndpoint(client kubernetes.Interface) (string, error) {

	var ip string
	var port string
	options := metav1.ListOptions{LabelSelector: tillerLabels.AsSelector().String()}
	services, err := client.CoreV1().Services(tillerNamespace).List(options)
	if err != nil {
		return "", err
	}

	if len(services.Items) < 1 {
		return "", fmt.Errorf("Failed to find tiller service")
	}

	for _, item := range services.Items {
		ports := item.Spec.Ports
		switch item.Spec.Type {
		case corev1.ServiceTypeClusterIP:
			ip = item.Spec.ClusterIP
			if ip == "None" {
				resolvedIps, _ := net.LookupHost("tiller-deploy")
				for _, resolvedIP := range resolvedIps {
					ip = resolvedIP
					break
				}
			}
			for _, p := range ports {
				if p.Port != 0 {
					port = strconv.Itoa(int(p.Port))
					break
				}
			}
		case corev1.ServiceTypeNodePort:
			ip, err = searchTillerPodIP(client)
			if err != nil {
				break
			}
			for _, p := range ports {
				if p.NodePort != 0 {
					port = strconv.Itoa(int(p.NodePort))
					break
				}
			}
		case corev1.ServiceTypeLoadBalancer:
			ip = item.Spec.LoadBalancerIP
			for _, p := range ports {
				if p.NodePort != 0 {
					port = strconv.Itoa(int(p.NodePort))
					break
				}
			}
		case corev1.ServiceTypeExternalName:
			ip = item.Spec.ExternalName
			for _, p := range ports {
				if p.NodePort != 0 {
					port = strconv.Itoa(int(p.NodePort))
					break
				}
			}
		}
	}

	if ip == "" {
		return "", fmt.Errorf("Failed to find working tiller")
	}

	return ip + ":" + port, nil
}

func Initialize(endpoint, key, cert, ca string, client kubernetes.Interface) (helm.Interface, error) {
	var err error
	fromcfg := false

	if endpoint == "" {
		endpoint, err = searchTillerEndpoint(client)
		if err != nil {
			return nil, err
		}
		fromcfg = false
	}

	hcli, err := ConnectToTiller(endpoint, key, cert, ca, fromcfg)
	if err != nil {
		return nil, err
	}

	return hcli, nil
}

// ConnectToTiller ...
func ConnectToTiller(endpoint, key, cert, ca string, fromcfg bool) (*helm.Client, error) {
	var tillerOptions []helm.Option

	klog.V(5).Infoln("Connecting to Tiller:", endpoint, " key:", key, " cert:", cert, " ca:", ca, " fromcfg:", fromcfg)

	if endpoint == "" {
		klog.Infoln("Tiller connection disabled", endpoint, " key:", key, " cert:", cert, " ca:", ca)
		return nil, nil
	}

	protoloc := strings.Index(endpoint, "//")
	if protoloc != -1 {
		endpoint = endpoint[protoloc+2:]
	}

	tillerOptions = []helm.Option{
		helm.Host(endpoint),
	}

	if key != "" && cert != "" {
		tlsopts := tlsutil.Options{KeyFile: key, CertFile: cert}
		_, err := os.Stat(ca)

		if err == nil {
			tlsopts.CaCertFile = ca
		} else {
			tlsopts.InsecureSkipVerify = true
		}

		tlscfg, err := tlsutil.ClientConfig(tlsopts)
		if err != nil {
			klog.Errorf("Unable to load tiller cert/key: %v", err)
			return nil, err
		}

		tillerOptions = append(tillerOptions, helm.WithTLS(tlscfg))
	}

	// create the client
	client := helm.NewClient(tillerOptions...)
	if _, err := client.GetVersion(); err != nil {
		klog.Errorf("Failed to connect to Tiller: %v", err)
		return nil, err
	}

	klog.Infof("Connected to Tiller at %s", endpoint)

	return client, nil
}

//Download HelmRepo
func (hc *HelmControl) downloadRepo(repoNamespace string, helmspec v1alpha1.HelmWorkSpec) (string, error) {
	var chartPath string
	var err error
	var inSecure bool
	if helmspec.ChartURL != "" {
		if helmspec.InSecureSkipVerify {
			inSecure = true
		}
		chartPath, err = DownloadChart(
			helmspec.ChartURL,
			helmspec.Version,
			"", "", "",
			false,
			inSecure,
		)

		if err != nil {
			return "", err
		}
	}

	return chartPath, nil
}

//CreateHelmRelease create a helm release
func (hc *HelmControl) CreateHelmRelease(releaseName string, helmreponNamespace string, helmspec v1alpha1.HelmWorkSpec) (*rls.InstallReleaseResponse, error) {
	chartPath, err := hc.downloadRepo(helmreponNamespace, helmspec)
	if err != nil {
		return nil, err
	}

	chartRequested, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	rls, err := hc.helmclient.InstallReleaseFromChart(
		chartRequested,
		helmspec.Namespace,
		helm.ValueOverrides(helmspec.Values),
		helm.ReleaseName(releaseName),
		helm.InstallTimeout(300))
	if err != nil {
		return nil, err
	}
	return rls, nil
}

// UpdateHelmRelease updates a helmrelease
func (hc *HelmControl) UpdateHelmRelease(releaseName string, helmreponNamespace string, helmspec v1alpha1.HelmWorkSpec) (*rls.UpdateReleaseResponse, error) {
	chartPath, err := hc.downloadRepo(helmreponNamespace, helmspec)
	if err != nil {
		return nil, err
	}

	resp, err := hc.helmclient.UpdateRelease(
		releaseName,
		chartPath,
		helm.UpdateValueOverrides(helmspec.Values),
		helm.UpgradeForce(true),
		helm.UpgradeTimeout(300),
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetHelmReleases ...
func (hc *HelmControl) GetHelmReleases(nameFilter string, codes []release.Status_Code, namespace string, limit int) (*rls.ListReleasesResponse, error) {
	var listRelOptions []helm.ReleaseListOption
	if nameFilter != "" {
		listRelOptions = append(listRelOptions, helm.ReleaseListFilter(nameFilter))
	}

	if codes != nil && len(codes) > 0 {
		listRelOptions = append(listRelOptions, helm.ReleaseListStatuses(codes))
	}

	if limit > 0 {
		listRelOptions = append(listRelOptions, helm.ReleaseListLimit(limit))
	}

	if namespace != "" {
		listRelOptions = append(listRelOptions, helm.ReleaseListNamespace(namespace))
	}

	rels, err := hc.helmclient.ListReleases(listRelOptions...)

	return rels, err
}

// DeleteHelmRelease delete helm release
func (hc *HelmControl) DeleteHelmRelease(relName string) (*rls.UninstallReleaseResponse, error) {
	var urr *rls.UninstallReleaseResponse
	var err error
	if relName != "" {
		urr, err = hc.helmclient.DeleteRelease(
			relName,
			helm.DeletePurge(true),
			helm.DeleteTimeout(300),
		)
		if err != nil {
			klog.Errorf("Uninstall of helm release %s failed: %v", relName, err)
			return urr, err
		}
	}

	return urr, err
}
