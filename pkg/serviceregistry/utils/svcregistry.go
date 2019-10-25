// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// build-in labels and annotations
const (
	AutoDiscoveryLabel         = "mcm.ibm.com/auto-discovery"
	ServiceTypeLabel           = "mcm.ibm.com/service-type"
	ClusterLabel               = "mcm.ibm.com/cluster"
	ServiceDiscoveryAnnotation = "mcm.ibm.com/service-discovery"
	LoadBalancerAnnotation     = "mcm.ibm.com/load-balancer"
	ClusterIPAnnotation        = "mcm.ibm.com/cluster-ip"
	DeployablePrefix           = "app.ibm.com"
	ServiceDiscoveryPrefix     = "service-discovery"
)

// FindClusterProxyIPFromConfigmap returns cluster proxy IP from kube-system/icp-management-ingress-config
func FindClusterProxyIPFromConfigmap(clientset kubernetes.Interface) (string, error) {
	configmap, err := clientset.Core().ConfigMaps("kube-public").Get("ibmcloud-cluster-info", metav1.GetOptions{})
	if err == nil {
		proxyIP, ok := configmap.Data["proxy_address"]
		if !ok {
			return "", fmt.Errorf("cannot find proxy_ip from kube-public/ibmcloud-cluster-info")
		}
		return proxyIP, nil
	}
	configmap, err = clientset.Core().ConfigMaps("kube-system").Get("icp-management-ingress-config", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	uiCfg := configmap.Data["ui-config.json"]
	var uiCfgRawMap map[string]*json.RawMessage
	if err := json.Unmarshal([]byte(uiCfg), &uiCfgRawMap); err != nil {
		return "", err
	}
	var uiConfiguration map[string]string
	if err := json.Unmarshal(*uiCfgRawMap["uiConfiguration"], &uiConfiguration); err != nil {
		return "", err
	}
	proxyIP, ok := uiConfiguration["proxy_ip"]
	if !ok || proxyIP == "" {
		return "", fmt.Errorf("cannot find proxy_ip from kube-system/icp-management-ingress-config")
	}
	return proxyIP, nil
}

// SplitRegisteredEndpointsName splits a registered endpoints name and returns the endpoints corresponding resource's type,
// namespace and name
func SplitRegisteredEndpointsName(endpointsName string) (resType, resNamespace, resName string, err error) {
	parts := strings.Split(endpointsName, ".")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("The registered endpoints name format should be type.namespace.name")
	}
	return parts[0], parts[1], parts[2], nil
}

// SplitDiscoveredEndpointsName splits a discovered endpoints name and returns the endpoints corresponding resource's cluster,
// type, namespace and name
func SplitDiscoveredEndpointsName(endpointsName string) (clusterName, resType, resNamespace, resName string, err error) {
	parts := strings.Split(endpointsName, ".")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("The discovered endpoints name format should be cluster.type.namespace.name")
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}

// NeedToUpdateEndpoints compares two endpoints with their labels, annotations and subsets to determine update is needed or not
func NeedToUpdateEndpoints(old, new *v1.Endpoints) bool {
	if !apiequality.Semantic.DeepEqual(old.Subsets, new.Subsets) ||
		!apiequality.Semantic.DeepEqual(old.Labels, new.Labels) ||
		!apiequality.Semantic.DeepEqual(old.Annotations, new.Annotations) {
		return false
	}
	return true
}

// GetDNSPrefix gets DNS prefix from service discovery annotation
func GetDNSPrefix(serviceDiscoveryAnnotation string) (dnsPrefix string, err error) {
	var rawMap map[string]*json.RawMessage
	if err = json.Unmarshal([]byte(serviceDiscoveryAnnotation), &rawMap); err != nil {
		return "", fmt.Errorf("failed to parse service discovery annotation, %v", err)
	}

	if len(rawMap) == 0 {
		return "", nil
	}

	_, ok := rawMap["dns-prefix"]
	if !ok {
		return "", nil
	}

	if err = json.Unmarshal(*rawMap["dns-prefix"], &dnsPrefix); err != nil {
		return "", fmt.Errorf("failed to parse service discovery annotation, %v", err)
	}

	return dnsPrefix, nil
}

// IsIstioEnabledNamespace ...
func IsIstioEnabledNamespace(clientset kubernetes.Interface, namespaceName string) bool {
	namespace, err := clientset.Core().Namespaces().Get(namespaceName, metav1.GetOptions{})
	if err != nil {
		return false
	}
	injection, ok := namespace.Labels["istio-injection"]
	if !ok {
		return false
	}
	return injection == "enabled"
}
