// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/serviceregistry/utils"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	istioPluginType        = "istio"
	istioDNSSuffix         = "global"
	servicePortsAnnotation = "mcm.ibm.com/istio-service-ports"
	autoCreationLabel      = "mcm.ibm.com/istio-auto-creation"
	ingressgatewayPort     = 15443
)

// IstioPlugin istio plugin
type IstioPlugin struct {
	clusterName        string
	clusterNamespace   string
	registryNamespace  string
	ingressName        string
	clusterProxyIP     string
	memberKubeClient   kubernetes.Interface
	serviceEntryClient dynamic.NamespaceableResourceInterface
	informer           cache.SharedIndexInformer
	serviceLister      listers.ServiceLister
}

type istioService struct {
	hostname  string
	addresses map[string]int
	ports     []int
}

// NewIstioPlugin returns the instance of istio plugin
func NewIstioPlugin(memberKubeClient kubernetes.Interface,
	memberDynamicClient dynamic.Interface,
	memberInformerFactory informers.SharedInformerFactory,
	options *options.SvcRegistryOptions) *IstioPlugin {
	return &IstioPlugin{
		clusterName:       options.ClusterName,
		clusterNamespace:  options.ClusterNamespace,
		registryNamespace: options.IstioPluginOptions.ServiceEntryRegistryNamespace,
		ingressName:       options.IstioPluginOptions.IstioIngressGateway,
		clusterProxyIP:    options.ClusterProxyIP,
		memberKubeClient:  memberKubeClient,
		serviceEntryClient: memberDynamicClient.Resource(schema.GroupVersionResource{
			Group:    "networking.istio.io",
			Version:  "v1alpha3",
			Resource: "serviceentries",
		}),
		informer:      memberInformerFactory.Core().V1().Services().Informer(),
		serviceLister: memberInformerFactory.Core().V1().Services().Lister(),
	}
}

// GetType always returns istio
func (p *IstioPlugin) GetType() string {
	return istioPluginType
}

// RegisterAnnotatedResouceHandler registers annotated istio gateway as an endpoints
func (p *IstioPlugin) RegisterAnnotatedResouceHandler(registryHandler cache.ResourceEventHandlerFuncs) {
	p.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoints := p.toEndpoints(obj)
			if endpoints != nil {
				registryHandler.OnAdd(endpoints)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldEndpoints := p.toEndpoints(old)
			newEndpoints := p.toEndpoints(new)
			if oldEndpoints != nil && newEndpoints != nil {
				registryHandler.OnUpdate(oldEndpoints, newEndpoints)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deletionEndpoints := p.toEndpoints(obj)
			if deletionEndpoints != nil {
				registryHandler.OnDelete(deletionEndpoints)
			}
		},
	})
}

// SyncRegisteredEndpoints synchronizes annotated istio gateways with plugin registered endpoints
func (p *IstioPlugin) SyncRegisteredEndpoints(registereds []*v1.Endpoints) (toCreate, toDelete, toUpdate []*v1.Endpoints) {
	toCreate = []*v1.Endpoints{}
	toUpdate = []*v1.Endpoints{}
	toDelete = []*v1.Endpoints{}

	for _, registered := range registereds {
		_, namespace, name, err := utils.SplitRegisteredEndpointsName(registered.Name)
		if err != nil {
			continue
		}
		service, err := p.serviceLister.Services(namespace).Get(name)
		if err != nil {
			// the endpoints corresponding service does not exist
			toDelete = append(toDelete, registered)
			continue
		}
		newEp := p.toEndpoints(service)
		if newEp == nil {
			// service may be changed or the annotation may be removed
			toDelete = append(toDelete, registered)
			continue
		}
		if !utils.NeedToUpdateEndpoints(registered, newEp) {
			// service changed
			toUpdate = append(toUpdate, newEp)
		}
	}

	services, err := p.serviceLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("failed to get istio services list %+v", err)
		return toCreate, toDelete, toUpdate
	}

	for _, service := range services {
		newEp := p.toEndpoints(service)
		if newEp == nil {
			continue
		}
		if p.isRegisterd(registereds, newEp) {
			continue
		}
		toCreate = append(toCreate, newEp)
	}
	return toCreate, toDelete, toUpdate
}

// DiscoveryRequired always returns true
func (p *IstioPlugin) DiscoveryRequired() bool {
	return true
}

// SyncDiscoveredResouces returns the current hub discovered service locations
func (p *IstioPlugin) SyncDiscoveredResouces(discoveredEndpoints []*v1.Endpoints) (bool, []*plugin.ServiceLocation) {
	// record created service entries
	created := map[string]bool{}
	currentServiceEntries, _ := p.serviceEntryClient.Namespace("").List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", autoCreationLabel),
	})
	discoveredServiceEntries := p.toDiscoveryServiceEntries(discoveredEndpoints, currentServiceEntries.Items)

	for _, current := range currentServiceEntries.Items {
		discovered, exists := discoveredServiceEntries[current.GetName()]
		if !exists {
			//if current service entry does not exist in discovered list, delete it
			p.deleteServiceEntry(current.GetNamespace(), current.GetName())
			continue
		}
		created[current.GetName()] = true
		//if current service entry corresponding discovered service entry is changed, update it
		if p.needToUpdate(&current, discovered) {
			discovered.SetResourceVersion(current.GetResourceVersion())
			p.updateServiceEntry(discovered)
		}
	}

	//discovered service entry does not exist in current list, create it
	for name, discovered := range discoveredServiceEntries {
		_, exists := created[name]
		if !exists {
			p.createServiceEntry(discovered)
		}
	}
	return false, nil
}

func (p *IstioPlugin) toEndpoints(obj interface{}) *v1.Endpoints {
	service, ok := obj.(*v1.Service)
	if !ok {
		return nil
	}

	_, ok = service.Annotations[utils.ServiceDiscoveryAnnotation]
	if !ok {
		return nil
	}

	if !utils.IsIstioEnabledNamespace(p.memberKubeClient, service.Namespace) {
		return nil
	}

	// labels
	labels := make(map[string]string)
	labels[utils.ServiceTypeLabel] = istioPluginType
	labels[utils.ClusterLabel] = p.clusterName
	for k, v := range service.Labels {
		if strings.HasPrefix(k, utils.DeployablePrefix) {
			labels[fmt.Sprintf("%s.%s", utils.ServiceDiscoveryPrefix, k)] = v
		} else {
			labels[k] = v
		}
	}

	// annotations
	annotations := make(map[string]string)
	for k, v := range service.Annotations {
		if strings.HasPrefix(k, utils.DeployablePrefix) {
			annotations[fmt.Sprintf("%s.%s", utils.ServiceDiscoveryPrefix, k)] = v
		} else {
			annotations[k] = v
		}
	}

	// service ports
	servicePorts := []string{}
	for _, svcport := range service.Spec.Ports {
		servicePorts = append(servicePorts, strconv.Itoa(int(svcport.Port)))
	}
	annotations[servicePortsAnnotation] = strings.Join(servicePorts, ",")

	// load balancer address
	lbaddress, lbport, err := p.findLoadBalancerAddress(p.serviceLister, p.ingressName)
	if err != nil {
		klog.Errorf("istio ingress gateway is not ready, %v", err)
		return nil
	}

	if lbaddress.IP == "" && lbaddress.Hostname == "" {
		klog.Errorf("failed to find load balaner address for %s", p.ingressName)
		return nil
	}

	if lbaddress.IP == "" {
		lbaddress.IP = "255.255.255.255"
		annotations[utils.LoadBalancerAnnotation] = lbaddress.Hostname
	}

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s.%s.%s", istioPluginType, service.Namespace, service.Name),
			Namespace:   p.clusterNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Subsets: []v1.EndpointSubset{{
			Addresses: []v1.EndpointAddress{{IP: lbaddress.IP}},
			Ports:     []v1.EndpointPort{{Port: lbport}},
		}},
	}
}

func (p *IstioPlugin) isRegisterd(registereds []*v1.Endpoints, toRegister *v1.Endpoints) bool {
	for _, ep := range registereds {
		if ep.Name == toRegister.Name {
			return true
		}
	}
	return false
}

func (p *IstioPlugin) toDiscoveryServiceEntries(discovereds []*v1.Endpoints,
	currentServiceEntries []unstructured.Unstructured) map[string]*unstructured.Unstructured {
	services := map[string]istioService{}
	for _, discovered := range discovereds {
		_, _, serviceNamespace, serviceName, err := utils.SplitDiscoveredEndpointsName(discovered.Name)
		if err != nil {
			continue
		}

		svcPortsAnnotation, ok := discovered.Annotations[servicePortsAnnotation]
		if !ok {
			continue
		}
		svcPorts := strings.Split(svcPortsAnnotation, ",")
		servicePorts := []int{}
		for _, svcport := range svcPorts {
			port, err := strconv.Atoi(svcport)
			if err != nil {
				break
			}
			servicePorts = append(servicePorts, port)
		}
		if len(servicePorts) != len(svcPorts) {
			continue
		}

		serviceEntryName := fmt.Sprintf("%s-%s-%s", utils.ServiceDiscoveryPrefix, serviceNamespace, serviceName)
		service, ok := services[serviceEntryName]
		if !ok {
			addresses := map[string]int{}
			addr, port := getIngressGatewayAddress(discovered)
			addresses[addr] = port
			services[serviceEntryName] = istioService{
				hostname:  fmt.Sprintf("%s.%s.%s", serviceName, serviceNamespace, istioDNSSuffix),
				addresses: addresses,
				ports:     servicePorts,
			}
			continue
		}
		addr, port := getIngressGatewayAddress(discovered)
		service.addresses[addr] = port
	}
	serviceEntries := map[string]*unstructured.Unstructured{}
	for name, service := range services {
		ip, ok := generateUniqueIP(name, currentServiceEntries)
		if !ok {
			continue
		}
		serviceEntries[name] = p.toServiceEntry(name, ip, service)
	}
	return serviceEntries
}

func (p *IstioPlugin) toServiceEntry(serviceEntryName string, ip string, service istioService) *unstructured.Unstructured {
	content := make(map[string]interface{})
	content["apiVersion"] = "networking.istio.io/v1alpha3"
	content["kind"] = "ServiceEntry"
	metadata := make(map[string]interface{})
	metadata["name"] = serviceEntryName
	metadata["namespace"] = p.registryNamespace
	metadata["labels"] = map[string]string{autoCreationLabel: "true"}
	content["metadata"] = metadata
	spec := make(map[string]interface{})
	spec["hosts"] = []string{service.hostname}
	spec["location"] = 1   // ServiceEntry_MESH_INTERNAL
	spec["resolution"] = 2 // ServiceEntry_DNS
	ports := []map[string]interface{}{}
	for _, svcport := range service.ports {
		portName := fmt.Sprintf("svc%d", svcport)
		port := make(map[string]interface{})
		port["name"] = portName
		port["number"] = svcport
		port["protocol"] = "tls"
		ports = append(ports, port)
	}
	spec["ports"] = ports
	spec["addresses"] = []string{ip}
	endpoints := []interface{}{}
	for address, port := range service.addresses {
		endpoint := make(map[string]interface{})
		endpoint["address"] = address
		epports := map[string]int{}
		for _, svcport := range service.ports {
			epports[fmt.Sprintf("svc%d", svcport)] = port
		}
		endpoint["ports"] = epports
		endpoints = append(endpoints, endpoint)
	}
	spec["endpoints"] = endpoints
	content["spec"] = spec
	serviceEntry := &unstructured.Unstructured{}
	// there must be no errors
	contentData, _ := json.Marshal(content)
	_ = serviceEntry.UnmarshalJSON(contentData)
	return serviceEntry
}

func (p *IstioPlugin) needToUpdate(oldObj, newObj *unstructured.Unstructured) bool {
	oldServiceEntry := oldObj.Object["spec"]
	newServiceEntry := newObj.Object["spec"]
	return !apiequality.Semantic.DeepEqual(oldServiceEntry, newServiceEntry)
}

func (p *IstioPlugin) createServiceEntry(serviceEntry *unstructured.Unstructured) {
	_, err := p.serviceEntryClient.Namespace(serviceEntry.GetNamespace()).Create(serviceEntry, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("failed to create istio ServiceEntry (%s/%s), %v", serviceEntry.GetNamespace(), serviceEntry.GetName(), err)
		return
	}
	klog.Infof("istio ServiceEntry (%s/%s) is created", serviceEntry.GetNamespace(), serviceEntry.GetName())
}

func (p *IstioPlugin) updateServiceEntry(serviceEntry *unstructured.Unstructured) {
	_, err := p.serviceEntryClient.Namespace(serviceEntry.GetNamespace()).Update(serviceEntry, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("failed to update istio ServiceEntry (%s/%s), %v", serviceEntry.GetNamespace(), serviceEntry.GetName(), err)
		return
	}
	klog.Infof("istio ServiceEntry (%s/%s) is updated", serviceEntry.GetNamespace(), serviceEntry.GetName())
}

func (p *IstioPlugin) deleteServiceEntry(namespace, name string) {
	err := p.serviceEntryClient.Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("failed to delete istio ServiceEntry (%s/%s), %v", namespace, name, err)
		return
	}
	klog.Infof("istio ServiceEntry (%s/%s) is deleted", namespace, name)
}

func (p *IstioPlugin) findLoadBalancerAddress(lister listers.ServiceLister,
	istioIngressgatewayName string) (*plugin.ServiceAddress, int32, error) {
	names := strings.Split(istioIngressgatewayName, "/")
	if len(names) != 2 {
		return nil, 0, fmt.Errorf("the name of istio ingress gateway should be in the format of namespace/name")
	}

	ingressGateway, err := lister.Services(names[0]).Get(names[1])
	if err != nil {
		return nil, 0, err
	}

	lb := ingressGateway.Status.LoadBalancer
	if lb.Ingress == nil || len(lb.Ingress) == 0 {
		var nodePort int32
		for _, port := range ingressGateway.Spec.Ports {
			if port.Port == ingressgatewayPort {
				nodePort = port.NodePort
				break
			}
		}
		return &plugin.ServiceAddress{IP: p.clusterProxyIP}, nodePort, nil
	}

	return &plugin.ServiceAddress{IP: lb.Ingress[0].IP, Hostname: lb.Ingress[0].Hostname}, ingressgatewayPort, nil
}

func getIngressGatewayAddress(discoveredEndpoints *v1.Endpoints) (string, int) {
	ingGWAddr := discoveredEndpoints.Subsets[0].Addresses[0].IP
	ingGWPort := discoveredEndpoints.Subsets[0].Ports[0].Port
	if ingGWAddr == "255.255.255.255" {
		ingGWAddr = discoveredEndpoints.Annotations[utils.LoadBalancerAnnotation]
	}
	return ingGWAddr, int(ingGWPort)
}

func generateUniqueIP(currentServiceEntryName string, serviceEntries []unstructured.Unstructured) (string, bool) {
	currentIPs := []int64{}
	for _, serviceEntry := range serviceEntries {
		spec, ok, _ := unstructured.NestedMap(serviceEntry.Object, "spec")
		if !ok {
			return "", false
		}
		addresses, ok, _ := unstructured.NestedStringSlice(spec, "addresses")
		if !ok {
			return "", false
		}
		if currentServiceEntryName == serviceEntry.GetName() {
			return addresses[0], true
		}
		currentIPs = append(currentIPs, big.NewInt(0).SetBytes(net.ParseIP(addresses[0]).To4()).Int64())
	}
	if len(currentIPs) == 0 {
		return "127.255.0.1", true
	}
	sort.Slice(currentIPs, func(i, j int) bool { return currentIPs[i] > currentIPs[j] })
	newIP := currentIPs[0] + 1
	return fmt.Sprintf("%d.%d.%d.%d", byte(newIP>>24), byte(newIP>>16), byte(newIP>>8), byte(newIP)), true
}
