// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	"fmt"
	"strings"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/serviceregistry/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const kubeServicePluginType = "kube-service"

// KubeServicePlugin kube-service plugin
type KubeServicePlugin struct {
	clusterName      string
	clusterZone      string
	clusterRegion    string
	clusterNamespace string
	clusterProxyIP   string
	memberKubeClient kubernetes.Interface
	informer         cache.SharedIndexInformer
	serviceLister    listers.ServiceLister
}

// NewKubeServicePlugin returns the instance of kube-service plugin
func NewKubeServicePlugin(memberKubeClient kubernetes.Interface,
	memberInformerFactory informers.SharedInformerFactory,
	options *options.SvcRegistryOptions) *KubeServicePlugin {
	var err error
	proxyIP := options.ClusterProxyIP
	if proxyIP == "" {
		if proxyIP, err = utils.FindClusterProxyIPFromConfigmap(memberKubeClient); err != nil {
			klog.Warningf("failed to find cluster proxy ip, %v", err)
		}
	}
	return &KubeServicePlugin{
		clusterName:      options.ClusterName,
		clusterZone:      options.ClusterZone,
		clusterRegion:    options.ClusterRegion,
		clusterNamespace: options.ClusterNamespace,
		clusterProxyIP:   proxyIP,
		memberKubeClient: memberKubeClient,
		informer:         memberInformerFactory.Core().V1().Services().Informer(),
		serviceLister:    memberInformerFactory.Core().V1().Services().Lister(),
	}
}

// GetType always returns kube-service
func (s *KubeServicePlugin) GetType() string {
	return kubeServicePluginType
}

// RegisterAnnotatedResouceHandler registers annotated k8s service as an endpoints
func (s *KubeServicePlugin) RegisterAnnotatedResouceHandler(registryHandler cache.ResourceEventHandlerFuncs) {
	s.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoints := s.toEndpoints(obj)
			if endpoints != nil {
				registryHandler.OnAdd(endpoints)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldEndpoints := s.toEndpoints(old)
			newEndpoints := s.toEndpoints(new)
			if oldEndpoints != nil && newEndpoints != nil {
				registryHandler.OnUpdate(oldEndpoints, newEndpoints)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deletionEndpoints := s.toDeletionEndpoints(obj)
			if deletionEndpoints != nil {
				registryHandler.OnDelete(deletionEndpoints)
			}
		},
	})
}

// SyncRegisteredEndpoints synchronizes annotated k8s service with plugin registered endpoints
func (s *KubeServicePlugin) SyncRegisteredEndpoints(
	registeredEdpoints []*v1.Endpoints) (toCreate, toDelete, toUpdate []*v1.Endpoints) {
	toCreate = []*v1.Endpoints{}
	toUpdate = []*v1.Endpoints{}
	toDelete = []*v1.Endpoints{}

	for _, ep := range registeredEdpoints {
		_, namespace, name, err := utils.SplitRegisteredEndpointsName(ep.Name)
		if err != nil {
			continue
		}
		service, err := s.serviceLister.Services(namespace).Get(name)
		if err != nil {
			// the endpoints corresponding service does not exist
			toDelete = append(toDelete, ep)
			continue
		}
		endpoints := s.toEndpoints(service)
		if endpoints == nil {
			// service type may be changed or the annotation may be removed
			toDelete = append(toDelete, ep)
			continue
		}
		if !utils.NeedToUpdateEndpoints(ep, endpoints) {
			// service changed
			toUpdate = append(toUpdate, ep)
		}
	}

	services, err := s.serviceLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("failed to get kube service list %+v", err)
		return toCreate, toDelete, toUpdate
	}

	for _, service := range services {
		_, annotated := s.isAnnotatedService(service)
		if annotated && !s.hasBeenRegisterd(registeredEdpoints, service) {
			// service does not register
			toCreate = append(toCreate, s.toEndpoints(service))
		}
	}
	return toCreate, toDelete, toUpdate
}

// DiscoveryRequired always returns true
func (s *KubeServicePlugin) DiscoveryRequired() bool {
	return true
}

// SyncDiscoveredResouces returns the current hub discovered service locations
func (s *KubeServicePlugin) SyncDiscoveredResouces(discoveredEndpoints []*v1.Endpoints) (bool, []*plugin.ServiceLocation) {
	locations := []*plugin.ServiceLocation{}
	for _, endpoints := range discoveredEndpoints {
		location, err := s.toServiceLocation(endpoints)
		if err != nil {
			klog.Errorf("failed to sync discovered endpoints (%s/%s), %v", endpoints.Namespace, endpoints.Name, err)
			continue
		}
		locations = append(locations, location)
		localLocation := s.findLocalService(endpoints)
		if localLocation != nil {
			locations = append(locations, location)
		}
	}
	return true, locations
}

func (s *KubeServicePlugin) toEndpoints(obj interface{}) *v1.Endpoints {
	service, ok := s.isAnnotatedService(obj)
	if !ok {
		return nil
	}

	annotations := make(map[string]string)
	annotations[utils.ClusterIPAnnotation] = service.Spec.ClusterIP
	for k, v := range service.Annotations {
		if strings.HasPrefix(k, utils.DeployablePrefix) {
			annotations[fmt.Sprintf("%s.%s", utils.ServiceDiscoveryPrefix, k)] = v
		} else {
			annotations[k] = v
		}
	}

	address := v1.EndpointAddress{IP: s.clusterProxyIP}
	if service.Spec.Type == v1.ServiceTypeLoadBalancer && s.loadBalancerIsReady(service) {
		if service.Status.LoadBalancer.Ingress[0].IP != "" {
			address.IP = service.Status.LoadBalancer.Ingress[0].IP
		} else {
			address.IP = "255.255.255.255"
			// put the load balancer hostname to annotaion, because the endpoints address hostname must consist of lower case
			// alphanumeric characters or '-', and must start and end with an alphanumeric character
			annotations[utils.LoadBalancerAnnotation] = service.Status.LoadBalancer.Ingress[0].Hostname
		}
	}

	if address.IP == "" {
		klog.Errorf("failed to get service (%s/%s) address ip", service.Namespace, service.Name)
		return nil
	}

	labels := make(map[string]string)
	labels[utils.ServiceTypeLabel] = kubeServicePluginType
	labels[utils.ClusterLabel] = s.clusterName
	for k, v := range service.Labels {
		if strings.HasPrefix(k, utils.DeployablePrefix) {
			labels[fmt.Sprintf("%s.%s", utils.ServiceDiscoveryPrefix, k)] = v
		} else {
			labels[k] = v
		}
	}

	ports := []v1.EndpointPort{}
	for _, port := range service.Spec.Ports {
		ports = append(ports, v1.EndpointPort{Name: port.Name, Port: port.NodePort, Protocol: port.Protocol})
	}

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s.%s.%s", kubeServicePluginType, service.Namespace, service.Name),
			Namespace:   s.clusterNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Subsets: []v1.EndpointSubset{{
			Addresses: []v1.EndpointAddress{address},
			Ports:     ports,
		}},
	}
}

func (s *KubeServicePlugin) toDeletionEndpoints(obj interface{}) *v1.Endpoints {
	service, ok := s.isAnnotatedService(obj)
	if !ok {
		return nil
	}
	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s.%s", kubeServicePluginType, service.Namespace, service.Name),
			Namespace: s.clusterNamespace,
		},
	}
}

func (s *KubeServicePlugin) isAnnotatedService(obj interface{}) (*v1.Service, bool) {
	service, ok := obj.(*v1.Service)
	if !ok {
		return nil, false
	}

	// ignore unannotated services
	_, ok = service.Annotations[utils.ServiceDiscoveryAnnotation]
	if !ok {
		return nil, false
	}

	if utils.IsIstioEnabledNamespace(s.memberKubeClient, service.Namespace) {
		return nil, false
	}

	// ignore other type services
	if service.Spec.Type != v1.ServiceTypeNodePort && service.Spec.Type != v1.ServiceTypeLoadBalancer {
		klog.Warningf("ingore the service (%s/%s) with type %s", service.Namespace, service.Name, service.Spec.Type)
		return nil, false
	}
	return service, true
}

func (s *KubeServicePlugin) loadBalancerIsReady(service *v1.Service) bool {
	if len(service.Status.LoadBalancer.Ingress) != 0 {
		return true
	}
	klog.V(5).Infof("the load balancer service (%s/%s) is not ready", service.Namespace, service.Name)
	return false
}

func (s *KubeServicePlugin) hasBeenRegisterd(registeredEdpoints []*v1.Endpoints, service *v1.Service) bool {
	for _, ep := range registeredEdpoints {
		if ep.Name == fmt.Sprintf("%s.%s.%s", kubeServicePluginType, service.Namespace, service.Name) {
			return true
		}
	}
	return false
}

func (s *KubeServicePlugin) toServiceLocation(discoveredEndpoints *v1.Endpoints) (*plugin.ServiceLocation, error) {
	clusterName, _, serviceNamespace, serviceName, err := utils.SplitDiscoveredEndpointsName(discoveredEndpoints.Name)
	if err != nil {
		return nil, err
	}

	ip := discoveredEndpoints.Subsets[0].Addresses[0].IP
	if clusterName == s.clusterName {
		ip = discoveredEndpoints.Annotations[utils.ClusterIPAnnotation]
	}
	address := plugin.ServiceAddress{
		IP: ip,
	}
	if address.IP == "255.255.255.255" {
		address.IP = ""
		address.Hostname = discoveredEndpoints.Annotations[utils.LoadBalancerAnnotation]
	}

	host, err := utils.GetDNSPrefix(discoveredEndpoints.Annotations[utils.ServiceDiscoveryAnnotation])
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = fmt.Sprintf("%s.%s", serviceName, serviceNamespace)
	}

	return &plugin.ServiceLocation{
		Address: address,
		Hosts:   []string{host},
		ClusterInfo: plugin.ClusterInfo{
			Name:   clusterName,
			Zone:   "",
			Region: "",
		},
	}, nil
}

func (s *KubeServicePlugin) findLocalService(discoveredEndpoints *v1.Endpoints) *plugin.ServiceLocation {
	_, _, serviceNamespace, serviceName, err := utils.SplitDiscoveredEndpointsName(discoveredEndpoints.Name)
	if err != nil {
		return nil
	}

	localSvc, err := s.serviceLister.Services(serviceNamespace).Get(serviceName)
	if err != nil {
		return nil
	}

	return &plugin.ServiceLocation{
		Address: plugin.ServiceAddress{
			IP: localSvc.Spec.ClusterIP,
		},
		Hosts: []string{fmt.Sprintf("%s.%s", serviceName, serviceNamespace)},
		ClusterInfo: plugin.ClusterInfo{
			Name:   s.clusterName,
			Zone:   s.clusterZone,
			Region: s.clusterRegion,
		},
	}
}
