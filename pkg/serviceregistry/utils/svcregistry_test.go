// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package utils

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	v1 "k8s.io/api/core/v1"
)

func TestFindClusterProxyIPFromClusterInfoConfigmap(t *testing.T) {
	configmap := &v1.ConfigMap{
		Data: map[string]string{
			"cluster_address":         "9.46.73.10",
			"proxy_address":           "9.46.73.106",
			"proxy_ingress_http_port": "80",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "ibmcloud-cluster-info",
		},
	}
	client := kubefake.NewSimpleClientset()
	client.Core().ConfigMaps("kube-public").Create(configmap)
	proxyIP, err := FindClusterProxyIPFromConfigmap(client)
	if err != nil {
		t.Fatalf("Failed to find prox-ip from configmap")
	}
	if proxyIP != "9.46.73.106" {
		t.Fatalf("Expected to get 1.2.3.4, but get %s", proxyIP)
	}
}

func TestFindClusterProxyIPFromIngressConfigmap(t *testing.T) {
	configmap := &v1.ConfigMap{
		Data: map[string]string{
			"ui-config.json": "{\"uiConfiguration\":{\"proxy_ip\":\"1.2.3.4\"}}",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "icp-management-ingress-config",
		},
	}
	client := kubefake.NewSimpleClientset()
	client.Core().ConfigMaps("kube-system").Create(configmap)
	proxyIP, err := FindClusterProxyIPFromConfigmap(client)
	if err != nil {
		t.Fatalf("Failed to find prox-ip from configmap")
	}
	if proxyIP != "1.2.3.4" {
		t.Fatalf("Expected to get 1.2.3.4, but get %s", proxyIP)
	}
}

func TestSplitRegisteredEndpointsName(t *testing.T) {
	invalidName := "cluster1.kube-service.default.httpbin"
	resType, resNamespace, resName, err := SplitRegisteredEndpointsName(invalidName)
	if err == nil {
		t.Fatalf("Expected to fail, but does not")
	}
	if resType != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	if resNamespace != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	if resName != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	epName := "kube-service.default.httpbin"
	resType, resNamespace, resName, err = SplitRegisteredEndpointsName(epName)
	if err != nil {
		t.Fatalf("Failed to parse endpoints name %+v", err)
	}
	if resType != "kube-service" {
		t.Fatalf("Expected to get kube-service, but get %s", resType)
	}
	if resNamespace != "default" {
		t.Fatalf("Expected to get default, but get %s", resNamespace)
	}
	if resName != "httpbin" {
		t.Fatalf("Expected to get httpbin, but get %s", resName)
	}
}

func TestSplitDiscoveredEndpointsName(t *testing.T) {
	invalidName := "kube-service.default.httpbin"
	clusterName, resType, resNamespace, resName, err := SplitDiscoveredEndpointsName(invalidName)
	if err == nil {
		t.Fatalf("Expected to fail, but does not")
	}
	if clusterName != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	if resType != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	if resNamespace != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	if resName != "" {
		t.Fatalf("Expected to get an empty value, but failed")
	}
	epName := "cluster1.kube-service.default.httpbin"
	clusterName, resType, resNamespace, resName, err = SplitDiscoveredEndpointsName(epName)
	if err != nil {
		t.Fatalf("Failed to parse endpoints name %+v", err)
	}
	if clusterName != "cluster1" {
		t.Fatalf("Expected to get cluster1, but get %s", clusterName)
	}
	if resType != "kube-service" {
		t.Fatalf("Expected to get kube-service, but get %s", resType)
	}
	if resNamespace != "default" {
		t.Fatalf("Expected to get default, but get %s", resNamespace)
	}
	if resName != "httpbin" {
		t.Fatalf("Expected to get httpbin, but get %s", resName)
	}
}

func TestNeedToUpdateEndpoints(t *testing.T) {
	old := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "kube-service.test.ep",
			Namespace:       "cluster1",
			ResourceVersion: "10000",
			Labels: map[string]string{
				"mcm.ibm.com/service-type": "kube-service",
				"mcm.ibm.com/cluster":      "cluster1",
			},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{{IP: "6.7.8.9"}},
				Ports:     []v1.EndpointPort{{Name: "http", Port: 30197, Protocol: "TCP"}},
			},
		},
	}
	newEp := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-service.test.ep",
			Namespace: "cluster1",
			Labels: map[string]string{
				"mcm.ibm.com/service-type": "kube-service",
				"mcm.ibm.com/cluster":      "cluster1",
			},
			Annotations: map[string]string{"mcm.ibm.com/service-discovery": "{}"},
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{{IP: "6.7.8.9"}},
				Ports:     []v1.EndpointPort{{Name: "http", Port: 30197, Protocol: "TCP"}},
			},
		},
	}
	if !NeedToUpdateEndpoints(old, newEp) {
		t.Fatalf("Expected to equal, but does not")
	}
}

func TestGetDNSPrefix(t *testing.T) {
	empty, err := GetDNSPrefix("{}")
	if err != nil {
		t.Fatalf("Expected no error, but %v", err)
	}
	if empty != "" {
		t.Fatalf("Expected empty, but does not")
	}
	empty, err = GetDNSPrefix("{\"target-clusters\": [\"cluster0\"]}")
	if err != nil {
		t.Fatalf("Expected no error, but %v", err)
	}
	if empty != "" {
		t.Fatalf("Expected empty, but does not")
	}
	dnsPrefix, _ := GetDNSPrefix("{\"dns-prefix\": \"http.svc\"}")
	if dnsPrefix != "http.svc" {
		t.Fatalf("Expected http.svc, but %s", dnsPrefix)
	}
}

func TestIsIstioEnabledNamespace(t *testing.T) {
	client := kubefake.NewSimpleClientset()
	client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test0",
			Labels: map[string]string{"istio-injection": "enabled"},
		},
	})
	client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test1",
		},
	})
	client.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "test0",
			Labels: map[string]string{"istio-injection": "disabled"},
		},
	})
	if !IsIstioEnabledNamespace(client, "test0") {
		t.Fatalf("Expected true, but false")
	}
	if IsIstioEnabledNamespace(client, "test1") {
		t.Fatalf("Expected false, but true")
	}
	if IsIstioEnabledNamespace(client, "test2") {
		t.Fatalf("Expected false, but true")
	}
}
