// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	"fmt"
	"strings"

	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const kubeIngressPluginType = "kube-ingress"
const ingressHostsAnnotation = "mcm.ibm.com/ingress-hosts"

// KubeIngressPlugin kube-ingress plugin
type KubeIngressPlugin struct {
	clusterName      string
	clusterNamespace string
	informer         cache.SharedIndexInformer
	ingressLister    listers.IngressLister
}

// NewKubeIngressPlugin returns the instance of kube-ingress plugin
func NewKubeIngressPlugin(memberInformerFactory informers.SharedInformerFactory,
	options *options.SvcRegistryOptions) *KubeIngressPlugin {
	return &KubeIngressPlugin{
		clusterName:      options.ClusterName,
		clusterNamespace: options.ClusterNamespace,
		informer:         memberInformerFactory.Extensions().V1beta1().Ingresses().Informer(),
		ingressLister:    memberInformerFactory.Extensions().V1beta1().Ingresses().Lister(),
	}
}

// GetType always returns kube-ingress
func (i *KubeIngressPlugin) GetType() string {
	return kubeIngressPluginType
}

// RegisterAnnotatedResouceHandler registers annotated k8s ingress as an endpoints
func (i *KubeIngressPlugin) RegisterAnnotatedResouceHandler(registryHandler cache.ResourceEventHandlerFuncs) {
	i.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoints := i.toEndpoints(obj)
			if endpoints != nil {
				registryHandler.OnAdd(endpoints)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldEndpoints := i.toEndpoints(old)
			newEndpoints := i.toEndpoints(new)
			if oldEndpoints != nil && newEndpoints != nil {
				registryHandler.OnUpdate(oldEndpoints, newEndpoints)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deletionEndpoints := i.toEndpoints(obj)
			if deletionEndpoints != nil {
				registryHandler.OnDelete(deletionEndpoints)
			}
		},
	})
}

// SyncRegisteredEndpoints synchronizes annotated k8s ingress with plugin registered endpoints
func (i *KubeIngressPlugin) SyncRegisteredEndpoints(
	registeredEdpoints []*v1.Endpoints) (toCreate, toDelete, toUpdate []*v1.Endpoints) {
	toCreate = []*v1.Endpoints{}
	toUpdate = []*v1.Endpoints{}
	toDelete = []*v1.Endpoints{}
	for _, ep := range registeredEdpoints {
		_, namespace, name, err := utils.SplitRegisteredEndpointsName(ep.Name)
		if err != nil {
			continue
		}
		ingress, err := i.ingressLister.Ingresses(namespace).Get(name)
		if err != nil {
			// the endpoints corresponding service does not exist
			toDelete = append(toDelete, ep)
			continue
		}
		endpoints := i.toEndpoints(ingress)
		if endpoints == nil {
			// ingress may be changed or the annotation may be removed
			toDelete = append(toDelete, ep)
			continue
		}
		if !utils.NeedToUpdateEndpoints(ep, endpoints) {
			// service changed
			toUpdate = append(toUpdate, ep)
		}
	}

	ingresses, err := i.ingressLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("failed to get kube ingresses list %+v", err)
		return toCreate, toDelete, toUpdate
	}

	for _, ingress := range ingresses {
		_, annotated := i.isAnnotatedIngress(ingress)
		if annotated && !i.hasBeenRegisterd(registeredEdpoints, ingress) {
			// ingress does not register
			toCreate = append(toCreate, i.toEndpoints(ingress))
		}
	}
	return toCreate, toDelete, toUpdate
}

// DiscoveryRequired always returns true
func (i *KubeIngressPlugin) DiscoveryRequired() bool {
	return true
}

// SyncDiscoveredResouces returns the current hub discovered service locations
func (i *KubeIngressPlugin) SyncDiscoveredResouces(discoveredEndpoints []*v1.Endpoints) (bool, []*plugin.ServiceLocation) {
	locations := []*plugin.ServiceLocation{}
	for _, endpoints := range discoveredEndpoints {
		location, err := i.toIngressLocation(endpoints)
		if err != nil {
			klog.Errorf("failed to sync discoverd endpoints (%s/%s), %v", endpoints.Namespace, endpoints.Name, err)
			continue
		}
		locations = append(locations, location)
	}
	return true, locations
}

func (i *KubeIngressPlugin) toEndpoints(obj interface{}) *v1.Endpoints {
	ingress, ok := i.isAnnotatedIngress(obj)
	if !ok {
		return nil
	}

	// labels
	labels := make(map[string]string)
	labels[utils.ServiceTypeLabel] = kubeIngressPluginType
	labels[utils.ClusterLabel] = i.clusterName
	for k, v := range ingress.Labels {
		if strings.HasPrefix(k, utils.DeployablePrefix) {
			labels[fmt.Sprintf("%s.%s", utils.ServiceDiscoveryPrefix, k)] = v
		} else {
			labels[k] = v
		}
	}

	// annotations
	annotations := make(map[string]string)
	for k, v := range ingress.Annotations {
		if strings.HasPrefix(k, utils.DeployablePrefix) {
			annotations[fmt.Sprintf("%s.%s", utils.ServiceDiscoveryPrefix, k)] = v
		} else {
			annotations[k] = v
		}
	}

	hosts := []string{}
	for _, rule := range ingress.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		hosts = append(hosts, rule.Host)
	}
	annotations[ingressHostsAnnotation] = strings.Join(hosts, ",")

	// address
	address := v1.EndpointAddress{IP: ingress.Status.LoadBalancer.Ingress[0].IP}
	if address.IP == "" {
		address.IP = "255.255.255.255"
		// put the load balancer hostname to annotaion, because the endpoints address hostname must consist of lower case
		// alphanumeric characters or '-', and must start and end with an alphanumeric character
		annotations[utils.LoadBalancerAnnotation] = ingress.Status.LoadBalancer.Ingress[0].Hostname
	}

	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s.%s.%s", kubeIngressPluginType, ingress.Namespace, ingress.Name),
			Namespace:   i.clusterNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Subsets: []v1.EndpointSubset{v1.EndpointSubset{
			Addresses: []v1.EndpointAddress{address},
			Ports: []v1.EndpointPort{{
				Port:     80,
				Protocol: v1.ProtocolTCP,
			}},
		}},
	}
}

func (i *KubeIngressPlugin) isAnnotatedIngress(obj interface{}) (*v1beta1.Ingress, bool) {
	ingress, ok := obj.(*v1beta1.Ingress)
	if !ok {
		return nil, false
	}

	// ignore the ingress without service-discovery annotation
	_, ok = ingress.Annotations[utils.ServiceDiscoveryAnnotation]
	if !ok {
		return nil, false
	}

	// ignore the ingress without rules
	if ingress.Spec.Backend != nil || len(ingress.Spec.Rules) == 0 {
		return nil, false
	}

	// ignore the ingress whoes status is not ready
	if !i.ingressStatusIsReady(ingress) {
		return nil, false
	}

	return ingress, true
}

func (i *KubeIngressPlugin) ingressStatusIsReady(ingress *v1beta1.Ingress) bool {
	if len(ingress.Status.LoadBalancer.Ingress) != 0 {
		return true
	}
	klog.V(5).Infof("the ingress (%s/%s) is not ready", ingress.Namespace, ingress.Name)
	return false
}

func (i *KubeIngressPlugin) hasBeenRegisterd(registeredEdpoints []*v1.Endpoints, ingress *v1beta1.Ingress) bool {
	for _, ep := range registeredEdpoints {
		if ep.Name == fmt.Sprintf("%s.%s.%s", kubeIngressPluginType, ingress.Namespace, ingress.Name) {
			return true
		}
	}
	return false
}

func (i *KubeIngressPlugin) toIngressLocation(discoveredEndpoints *v1.Endpoints) (*plugin.ServiceLocation, error) {
	clusterName, _, _, _, err := utils.SplitDiscoveredEndpointsName(discoveredEndpoints.Name)
	if err != nil {
		return nil, err
	}
	address := plugin.ServiceAddress{
		IP: discoveredEndpoints.Subsets[0].Addresses[0].IP,
	}
	if address.IP == "255.255.255.255" {
		address.IP = ""
		address.Hostname = discoveredEndpoints.Annotations[utils.LoadBalancerAnnotation]
	}

	hosts := strings.Split(discoveredEndpoints.Annotations[ingressHostsAnnotation], ",")
	host, err := utils.GetDNSPrefix(discoveredEndpoints.Annotations[utils.ServiceDiscoveryAnnotation])
	if err != nil {
		return nil, err
	}
	if host != "" {
		hosts = append([]string{host}, hosts...)
	}

	// TODO: zone and region

	return &plugin.ServiceLocation{
		Address: address,
		Hosts:   hosts,
		ClusterInfo: plugin.ClusterInfo{
			Name:   clusterName,
			Zone:   "",
			Region: "",
		},
	}, nil
}
