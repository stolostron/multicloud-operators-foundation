/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	equalutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/equals"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1beta1 "github.com/open-cluster-management/multicloud-operators-foundation/pkg/apis/cluster/v1beta1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"
)

// ClusterInfoReconciler reconciles a ClusterInfo object
type ClusterInfoReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	KubeClient        *kubernetes.Clientset
	RouteV1Client     routev1.Interface
	MasterAddresses   string
	KlusterletAddress string
	KlusterletIngress string
	KlusterletRoute   string
	KlusterletService string
	KlusterletPort    int32
}

func (r *ClusterInfoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("clusterinfo", req.NamespacedName)

	var clusterInfo *clusterv1beta1.ClusterInfo
	err := r.Get(ctx, req.NamespacedName, clusterInfo)
	if err != nil {
		log.Error(err, "unable to fetch work")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update cluster info here.
	shouldUpdateStatus := false
	masterAddresses, _, clusterURL := r.getMasterAddresses()

	// Config endpoint
	if !equalutils.EqualEndpointAddresses(masterAddresses, clusterInfo.Spec.MasterAddresses) {
		clusterInfo.Spec.MasterAddresses = masterAddresses
		shouldUpdateStatus = true
	}
	// Config console url
	if clusterInfo.Spec.ConsoleURL != clusterURL {
		clusterInfo.Spec.ConsoleURL = clusterURL
		shouldUpdateStatus = true
	}
	// Config klusterlet endpoint
	klusterletEndpoint, klusterletPort, err := r.readKlusterletConfig()
	if err == nil {
		if !reflect.DeepEqual(klusterletEndpoint, &clusterInfo.Spec.KlusterletEndpoint) {
			clusterInfo.Spec.KlusterletEndpoint = *klusterletEndpoint
			shouldUpdateStatus = true
		}
		if !reflect.DeepEqual(klusterletPort, &clusterInfo.Spec.KlusterletPort) {
			clusterInfo.Spec.KlusterletPort = *klusterletPort
			shouldUpdateStatus = true
		}
	} else {
		log.Error(err, "Failed to get klusterlet server config")
	}

	// Get version
	version := r.getVersion()
	if version != clusterInfo.Spec.Version {
		clusterInfo.Spec.Version = version
		shouldUpdateStatus = true
	}

	if shouldUpdateStatus {
		err = r.Client.Status().Update(ctx, clusterInfo)
		if err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *ClusterInfoReconciler) getMasterAddresses() ([]corev1.EndpointAddress, []corev1.EndpointPort, string) {
	//for Openshift
	masterAddresses, masterPorts, clusterURL, notEmpty, err := r.getMasterAddressesFromConsoleConfig()
	if err == nil && notEmpty {
		return masterAddresses, masterPorts, clusterURL
	}

	if r.MasterAddresses == "" {
		kubeEndpoints, serviceErr := r.KubeClient.CoreV1().Endpoints("default").Get("kubernetes", metav1.GetOptions{})
		if serviceErr == nil && len(kubeEndpoints.Subsets) > 0 {
			masterAddresses = kubeEndpoints.Subsets[0].Addresses
			masterPorts = kubeEndpoints.Subsets[0].Ports
		}
	} else {
		masterAddresses = append(masterAddresses, corev1.EndpointAddress{IP: r.MasterAddresses})
	}

	return masterAddresses, masterPorts, clusterURL
}

//for OpenShift, read endpoint address from console-config in openshift-console
func (r *ClusterInfoReconciler) getMasterAddressesFromConsoleConfig() ([]corev1.EndpointAddress, []corev1.EndpointPort, string, bool, error) {
	masterAddresses := []corev1.EndpointAddress{}
	masterPorts := []corev1.EndpointPort{}
	clusterURL := ""

	cfg, err := r.KubeClient.CoreV1().ConfigMaps("openshift-console").Get("console-config", metav1.GetOptions{})
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
				euArray := strings.Split(strings.Trim(eu, "htps:/"), ":")
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
	return masterAddresses, masterPorts, clusterURL, cfg != nil && cfg.Data != nil, err
}

func (r *ClusterInfoReconciler) readKlusterletConfig() (*corev1.EndpointAddress, *corev1.EndpointPort, error) {
	endpoint := &corev1.EndpointAddress{}
	port := &corev1.EndpointPort{
		Name:     "https",
		Protocol: corev1.ProtocolTCP,
		Port:     r.KlusterletPort,
	}
	// set endpoint by user flag at first, if it is not set, use IP in ingress status
	ip := net.ParseIP(r.KlusterletAddress)
	if ip != nil {
		endpoint.IP = r.KlusterletAddress
	} else {
		endpoint.Hostname = r.KlusterletAddress
	}

	if r.KlusterletIngress != "" {
		if err := r.setEndpointAddressFromIngress(endpoint); err != nil {
			return nil, nil, err
		}
	}

	// Only use klusterlet service when neither ingress nor address is set
	if r.KlusterletService != "" && endpoint.IP == "" && endpoint.Hostname == "" {
		if err := r.setEndpointAddressFromService(endpoint, port); err != nil {
			return nil, nil, err
		}
	}

	if r.KlusterletRoute != "" && endpoint.IP == "" && endpoint.Hostname == "" {
		if err := r.setEndpointAddressFromRoute(endpoint); err != nil {
			return nil, nil, err
		}
	}

	return endpoint, port, nil
}

func (r *ClusterInfoReconciler) setEndpointAddressFromIngress(endpoint *corev1.EndpointAddress) error {
	log := r.Log.WithName("SetEndpoint")
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(r.KlusterletIngress)
	if err != nil {
		log.Error(err, "Failed do parse ingress resource:")
		return err
	}
	klIngress, err := r.KubeClient.ExtensionsV1beta1().Ingresses(klNamespace).Get(klName, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "Failed do get ingress resource: %v")
		return err
	}

	if endpoint.IP == "" && len(klIngress.Status.LoadBalancer.Ingress) > 0 {
		endpoint.IP = klIngress.Status.LoadBalancer.Ingress[0].IP
	}

	if endpoint.Hostname == "" && len(klIngress.Spec.Rules) > 0 {
		endpoint.Hostname = klIngress.Spec.Rules[0].Host
	}
	return nil
}

func (r *ClusterInfoReconciler) setEndpointAddressFromService(endpoint *corev1.EndpointAddress, port *corev1.EndpointPort) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(r.KlusterletService)
	if err != nil {
		return err
	}

	klSvc, err := r.KubeClient.CoreV1().Services(klNamespace).Get(klName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if klSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return fmt.Errorf("klusterlet service should be in type of loadbalancer")
	}

	if len(klSvc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf("klusterlet load balancer service does not have valid ip address")
	}

	if len(klSvc.Spec.Ports) == 0 {
		return fmt.Errorf("klusterlet load balancer service does not have valid port")
	}

	// Only get the first IP and port
	endpoint.IP = klSvc.Status.LoadBalancer.Ingress[0].IP
	endpoint.Hostname = klSvc.Status.LoadBalancer.Ingress[0].Hostname
	port.Port = klSvc.Spec.Ports[0].Port
	return nil
}

func (r *ClusterInfoReconciler) setEndpointAddressFromRoute(endpoint *corev1.EndpointAddress) error {
	klNamespace, klName, err := cache.SplitMetaNamespaceKey(r.KlusterletRoute)
	if err != nil {
		log.Error(err, "Route name input error")
		return err
	}

	route, err := r.RouteV1Client.RouteV1().Routes(klNamespace).Get(klName, metav1.GetOptions{})

	if err != nil {
		log.Error(err, "Failed to get the route")
		return err
	}

	endpoint.Hostname = route.Spec.Host
	return nil
}

func (r *ClusterInfoReconciler) getVersion() string {
	serverVersionString := ""
	serverVersion, err := r.KubeClient.ServerVersion()
	if err == nil {
		serverVersionString = serverVersion.String()
	}

	var isOpenShift = false
	serverGroups, err := r.KubeClient.ServerGroups()
	if err != nil {
		return serverVersionString
	}
	for _, apiGroup := range serverGroups.Groups {
		if apiGroup.Name == "project.openshift.io" {
			isOpenShift = true
			break
		}
	}

	var isAKS = false
	_, err = r.KubeClient.CoreV1().ServiceAccounts("kube-system").Get("omsagent", metav1.GetOptions{})
	if err == nil {
		isAKS = true
	}

	if isOpenShift {
		if strings.Contains(serverVersionString, "+") {
			serverVersionString += ".rhos"
		} else {
			serverVersionString += "+rhos"
		}
	}
	if isAKS {
		if strings.Contains(serverVersionString, "+") {
			serverVersionString += ".aks"
		} else {
			serverVersionString += "+aks"
		}
	}

	return serverVersionString
}

func (r *ClusterInfoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1beta1.ClusterInfo{}).
		Complete(r)
}
