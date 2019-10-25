// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package controller

import (
	"fmt"
	"strings"
	"time"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog"
)

const svcregistryDNSDBKey string = "svcregistry.db"

// EndpointsController object
type EndpointsController struct {
	memberKubeClient   kubernetes.Interface
	hubEndpointsLister listers.EndpointsLister
	clusterInfo        plugin.ClusterInfo
	clusterNamespace   string
	dnsConfigmapName   string
	dnsSuffix          string
	plugins            []plugin.Plugin
	syncPeriod         int
	stopCh             <-chan struct{}
}

// NewEndpointsController creates endpoints controller
func NewEndpointsController(memberKubeClient kubernetes.Interface,
	hubInformerFactory informers.SharedInformerFactory,
	clusterInfo plugin.ClusterInfo,
	clusterNamespace, dnsConfigmapName, dnsSuffix string,
	enabledPlugins []plugin.Plugin,
	syncPeriod int,
	stopCh <-chan struct{}) *EndpointsController {
	return &EndpointsController{
		memberKubeClient:   memberKubeClient,
		hubEndpointsLister: hubInformerFactory.Core().V1().Endpoints().Lister(),
		clusterInfo:        clusterInfo,
		clusterNamespace:   clusterNamespace,
		dnsConfigmapName:   dnsConfigmapName,
		dnsSuffix:          dnsSuffix,
		plugins:            enabledPlugins,
		syncPeriod:         syncPeriod,
		stopCh:             stopCh,
	}
}

// Run starts this controller
func (c *EndpointsController) Run() {
	defer runtime.HandleCrash()

	klog.Infof("endpoints controller is ready\n")

	period := time.Duration(c.syncPeriod) * time.Second
	go wait.Until(c.syncAutoDiscoveryEndpoints, period, c.stopCh)

	<-c.stopCh
	klog.Infof("endpoints plugin controller is stop\n")
}

func (c *EndpointsController) syncAutoDiscoveryEndpoints() {
	dnsRequiredLocations := []*plugin.ServiceLocation{}
	for _, p := range c.plugins {
		if !p.DiscoveryRequired() {
			continue
		}
		selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				utils.ServiceTypeLabel:   p.GetType(),
				utils.AutoDiscoveryLabel: "true",
			},
		})
		endpoints, err := c.hubEndpointsLister.Endpoints(c.clusterNamespace).List(selector)
		if err != nil {
			return
		}
		dnsRequired, locations := p.SyncDiscoveredResouces(endpoints)
		if dnsRequired {
			dnsRequiredLocations = append(dnsRequiredLocations, locations...)
		}
	}

	err := c.refreshDNSRecords(dnsRequiredLocations)
	if err != nil {
		runtime.HandleError(err)
	}
}

func (c *EndpointsController) refreshDNSRecords(locations []*plugin.ServiceLocation) error {
	nameParts := strings.Split(c.dnsConfigmapName, "/")
	if len(nameParts) != 2 {
		return fmt.Errorf("the dns configmap name %s should be with the format of namespace/name", c.dnsConfigmapName)
	}
	configmap, err := c.memberKubeClient.Core().ConfigMaps(nameParts[0]).Get(nameParts[1], metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get the dns configmap %s, %v", c.dnsConfigmapName, err)
	}

	lastRecords, ok := configmap.Data[svcregistryDNSDBKey]
	if !ok {
		return fmt.Errorf("failed to get the %s from dns configmap %s", svcregistryDNSDBKey, c.dnsConfigmapName)
	}
	updateTime := time.Now().Unix()
	lastSOARecord, newSOARecord, err := utils.UpdateSOARecord(lastRecords, updateTime)
	if err != nil {
		return fmt.Errorf("failed to update SOA record, %v", err)
	}

	newRecords := utils.NewServiceDNSRecords(c.clusterInfo, c.dnsSuffix, locations)
	if !utils.NeedToUpdateDNSRecords(lastRecords, fmt.Sprintf("%s\n%s", lastSOARecord, newRecords)) {
		return nil
	}

	configmap.Data[svcregistryDNSDBKey] = fmt.Sprintf("%s\n%s", newSOARecord, newRecords)
	_, err = c.memberKubeClient.Core().ConfigMaps(nameParts[0]).Update(configmap)
	if err != nil {
		return fmt.Errorf("failed to updated dns configmap %s, %v", c.dnsConfigmapName, err)
	}
	klog.Infof("update discovered resouce dns records in configmap %s at %d", c.dnsConfigmapName, updateTime)
	return nil
}
