package controllers

import (
	"context"
	"crypto/md5" // #nosec
	"fmt"
	"io"

	routev1 "github.com/openshift/api/route/v1"
	openshiftclientset "github.com/openshift/client-go/config/clientset/versioned"
	routeclient "github.com/openshift/client-go/route/clientset/versioned"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/clusterclaim"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type loggingInfoSyncer struct {
	clusterName             string
	agentName               string
	agentNamespace          string
	agentPort               int32
	agent                   *agent.Agent
	managementClusterClient kubernetes.Interface
	configV1Client          openshiftclientset.Interface
	routeV1Client           routeclient.Interface
}

func (s *loggingInfoSyncer) sync(ctx context.Context, clusterInfo *clusterv1beta1.ManagedClusterInfo) error {
	// refresh logging server CA if CA is changed
	s.refreshAgentServer(clusterInfo)

	clusterInfo.Status.LoggingPort = corev1.EndpointPort{
		Name:     "https",
		Protocol: corev1.ProtocolTCP,
		Port:     s.agentPort,
	}

	err := s.setLoggingEndAddr(ctx, clusterInfo)
	if err != nil {
		return fmt.Errorf("failed to get logging agent config. error %v", err)
	}
	return nil
}

func (s *loggingInfoSyncer) refreshAgentServer(clusterInfo *clusterv1beta1.ManagedClusterInfo) {
	select {
	case s.agent.RunServer <- *clusterInfo:
		klog.Info("Signal agent server to start")
	default:
	}

	s.agent.RefreshServerIfNeeded(clusterInfo)
}

func (s *loggingInfoSyncer) setLoggingEndAddr(ctx context.Context, clusterInfo *clusterv1beta1.ManagedClusterInfo) error {
	var serviceType = corev1.ServiceTypeClusterIP
	if clusterInfo.Status.DistributionInfo.Type != clusterv1beta1.DistributionTypeOCP {
		serviceType = corev1.ServiceTypeLoadBalancer
	}

	isOCP311 := false
	if clusterInfo.Status.DistributionInfo.OCP.Version == clusterclaim.OCP3Version {
		isOCP311 = true
	}

	// when we delete addon agent deployment, we expect the route is deleted too.
	owner, err := s.getOwnerRef(ctx, isOCP311)
	if err != nil {
		return fmt.Errorf("failed to get owner. error: %v", err)
	}
	newAgentService := s.newService(serviceType, *owner)

	agentService, err := s.managementClusterClient.CoreV1().Services(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		_, err = s.managementClusterClient.CoreV1().Services(s.agentNamespace).Create(ctx, newAgentService, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create agent service. error:%v", err)
		}
	case err != nil:
		return fmt.Errorf("failed to get agent service. error:%v", err)
	case needUpdateService(agentService, newAgentService):
		_, err = s.managementClusterClient.CoreV1().Services(s.agentNamespace).Update(ctx, agentService, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create agent service. error:%v", err)
		}
	}

	if s.clusterName == "local-cluster" {
		clusterInfo.Status.LoggingEndpoint.Hostname = fmt.Sprintf("%s.%s.svc", s.agentName, s.agentNamespace)
		return nil
	}

	if clusterInfo.Status.DistributionInfo.Type != clusterv1beta1.DistributionTypeOCP {
		endPoint, err := s.getEndpointAddressFromService(ctx)
		if err != nil {
			return fmt.Errorf("failed to get EndpointAddress from service. error %v", err)
		}
		clusterInfo.Status.LoggingEndpoint = *endPoint
		return nil
	}

	// for OCP, get hostName from route
	endPoint, err := s.getEndpointAddressFromRoute(ctx, isOCP311)
	if err != nil {
		return fmt.Errorf("failed to get EndpointAddress from service. error %v", err)
	}
	clusterInfo.Status.LoggingEndpoint = *endPoint
	return nil

}

func (s *loggingInfoSyncer) getEndpointAddressFromService(ctx context.Context) (*corev1.EndpointAddress, error) {
	klSvc, err := s.managementClusterClient.CoreV1().Services(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if klSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil, fmt.Errorf("agent service should be in type of loadbalancer")
	}

	if len(klSvc.Status.LoadBalancer.Ingress) == 0 {
		return nil, fmt.Errorf("agent load balancer service does not have valid ip address")
	}

	// Only get the first IP and host name
	endpoint := &corev1.EndpointAddress{
		IP:       klSvc.Status.LoadBalancer.Ingress[0].IP,
		Hostname: klSvc.Status.LoadBalancer.Ingress[0].Hostname,
	}

	return endpoint, nil
}

func (s *loggingInfoSyncer) getEndpointAddressFromRoute(ctx context.Context, isOCP311 bool) (*corev1.EndpointAddress, error) {
	var (
		err      error
		endpoint = &corev1.EndpointAddress{}
	)

	// when we delete addon agent deployment, we expect the route is deleted too.
	owner, err := s.getOwnerRef(ctx, isOCP311)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner. error: %v", err)
	}
	newRoute := s.newRoute(*owner)
	// there is no configIngress in OCP 311,only get domain from configIngress in OCP4.
	// for OCP311 the host is empty and will be generated by OCP.
	if !isOCP311 {
		ingressDomain, err := s.getIngressDomain(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get ingress domain.error %v", err)
		}
		hostName := subHostName(s.agentNamespace)
		newRoute.Spec.Host = fmt.Sprintf("%s.%s", hostName, ingressDomain)
	}

	route, err := s.routeV1Client.RouteV1().Routes(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		route, err = s.routeV1Client.RouteV1().Routes(s.agentNamespace).Create(ctx, newRoute, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create route.error %v", err)
		}
	case err != nil:
		return nil, fmt.Errorf("failed to get route. error %v", err)
	case needUpdateRoute(route, newRoute):
		route, err = s.routeV1Client.RouteV1().Routes(s.agentNamespace).Update(ctx, route, metav1.UpdateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to update route. error %v", err)
		}
	}

	endpoint.Hostname = route.Spec.Host
	return endpoint, nil
}

func (s *loggingInfoSyncer) getOwnerRef(ctx context.Context, ocp311 bool) (*metav1.OwnerReference, error) {
	if ocp311 {
		deploy, err := s.managementClusterClient.ExtensionsV1beta1().Deployments(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get work-manager addon agent deployment.error %v", err)
		}
		return metav1.NewControllerRef(deploy, schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "Deployment"}), nil

	}

	deploy, err := s.managementClusterClient.AppsV1().Deployments(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get work-manager addon agent deployment.error %v", err)
	}
	return metav1.NewControllerRef(deploy, schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}), nil
}

func (s *loggingInfoSyncer) getIngressDomain(ctx context.Context) (string, error) {
	clusterIngress, err := s.configV1Client.ConfigV1().Ingresses().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get configv1.ingress. error %v", err)
	}

	if clusterIngress.Spec.AppsDomain != "" {
		return clusterIngress.Spec.AppsDomain, nil
	}

	if clusterIngress.Spec.Domain == "" {
		return "", fmt.Errorf("ingress domain not found or empty in Ingress")
	}
	return clusterIngress.Spec.Domain, nil
}

func (s *loggingInfoSyncer) newRoute(owner metav1.OwnerReference) *routev1.Route {
	weight := (int32)(100)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.agentName,
			Namespace: s.agentNamespace,
			Labels:    map[string]string{"component": "work-manager"},
			OwnerReferences: []metav1.OwnerReference{
				owner,
			},
		},
		Spec: routev1.RouteSpec{
			Host: "",
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   s.agentName,
				Weight: &weight,
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationPassthrough,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}
}

func (s *loggingInfoSyncer) newService(serviceType corev1.ServiceType, owner metav1.OwnerReference) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.agentName,
			Namespace: s.agentNamespace,
			Labels:    map[string]string{"component": "work-manager"},
			OwnerReferences: []metav1.OwnerReference{
				owner,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "app",
					Protocol:   corev1.ProtocolTCP,
					Port:       s.agentPort,
					TargetPort: intstr.FromInt(4443),
				},
			},
			Selector: map[string]string{"component": "work-manager"},
			Type:     serviceType,
		},
	}
}

func needUpdateRoute(old, new *routev1.Route) bool {
	needUpdate := false
	if !equality.Semantic.DeepEqual(old.OwnerReferences, new.OwnerReferences) {
		old.OwnerReferences = new.OwnerReferences
		needUpdate = true
	}

	if !equality.Semantic.DeepEqual(old.Spec, new.Spec) {
		old.Spec = new.Spec
		needUpdate = true
	}
	return needUpdate
}

func needUpdateService(old, new *corev1.Service) bool {
	needUpdate := false
	if old.Spec.Type != new.Spec.Type {
		old.Spec.Type = new.Spec.Type
		needUpdate = true
	}
	if !equality.Semantic.DeepEqual(old.Spec.Ports, new.Spec.Ports) {
		old.Spec.Ports = new.Spec.Ports
		needUpdate = true
	}
	if old.Spec.Selector["component"] != "work-manager" {
		old.Spec.Selector["component"] = "work-manager"
		needUpdate = true
	}
	if !equality.Semantic.DeepEqual(old.OwnerReferences, new.OwnerReferences) {
		old.OwnerReferences = new.OwnerReferences
		needUpdate = true
	}
	return needUpdate
}

func hashOfString(s string) string {
	// #nosec
	h := md5.New()
	if _, err := io.WriteString(h, s); err != nil {
		klog.Errorf("failed to hash object: %v", err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func subHostName(agentNamespace string) string {
	subName := fmt.Sprintf("workmgr-%s", agentNamespace)
	// Please be aware that instead of the standardized 63 characters long maximum length,
	// the domain names generated from the ExternalDNS sources have the following limits:
	// for the CNAME record type: 44 characters
	// 42 characters for wildcard records on AzureDNS (OCPBUGS-819)
	// for the A record type: 48 characters
	// 46 characters for wildcard records on AzureDNS (OCPBUGS-819)
	if len(subName) < 40 {
		return subName
	}

	// the length of hash is 32, `workmgr-<hash of sub name>` is 40
	return fmt.Sprintf("workmgr-%s", hashOfString(subName))
}
