package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"

	routev1 "github.com/openshift/api/route/v1"
	routefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"testing"
)

var bTrue = true

func newAgentLBService(name, namespace string, owner metav1.OwnerReference) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"component": "work-manager"},
			OwnerReferences: []metav1.OwnerReference{
				owner,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:       "app",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(4443),
				},
			},
			Selector: map[string]string{"component": "work-manager"},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP:       "10.0.0.1",
						Hostname: "myhost.com",
					},
				},
			},
		},
	}
}

func newAgentIPService(name, namespace string, owner metav1.OwnerReference) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				owner,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					TargetPort: intstr.FromInt(80),
				},
			},
			Selector: map[string]string{"component": "work-manager"},
		},
	}
}

func newRoute(name, namespace string, owner metav1.OwnerReference) *routev1.Route {
	weight := (int32)(100)
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"component": "work-manager"},
			OwnerReferences: []metav1.OwnerReference{
				owner,
			},
		},
		Spec: routev1.RouteSpec{
			Host: "",
			To: routev1.RouteTargetReference{
				Kind:   "Service",
				Name:   name,
				Weight: &weight,
			},
			TLS: &routev1.TLSConfig{
				Termination: routev1.TLSTerminationPassthrough,
			},
			WildcardPolicy: routev1.WildcardPolicyNone,
		},
	}
}

func Test_LoggingInfo_syncer(t *testing.T) {
	tests := []struct {
		name                      string
		managedClusterInfo        *v1beta1.ManagedClusterInfo
		agentName, agentNamespace string
		agentService              *corev1.Service
		agentRoute                *routev1.Route
	}{
		{
			name:           "no service,no route",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
					LoggingEndpoint: corev1.EndpointAddress{Hostname: "abc"},
					LoggingPort:     corev1.EndpointPort{Port: 443},
				},
			},
		},
		{
			name:           "local-cluster",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "local-cluster", Namespace: "local-cluster"},
				Status: v1beta1.ClusterInfoStatus{
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
					LoggingEndpoint: corev1.EndpointAddress{Hostname: "abc"},
					LoggingPort:     corev1.EndpointPort{Port: 443},
				},
			},
		},
		{
			name:           "non-ocp",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					LoggingEndpoint: corev1.EndpointAddress{Hostname: "abc"},
					LoggingPort:     corev1.EndpointPort{Port: 443},
				},
			},
			agentService: newAgentLBService(
				"klusterlet-addon-workmgr",
				"myns",
				metav1.OwnerReference{
					APIVersion:         "app/v1",
					Kind:               "Deployment",
					Name:               "klusterlet-addon-workmgr",
					UID:                "",
					Controller:         &bTrue,
					BlockOwnerDeletion: &bTrue,
				}),
		},

		{
			name:           "ocp delete service and route",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
					LoggingEndpoint: corev1.EndpointAddress{Hostname: "abc"},
					LoggingPort:     corev1.EndpointPort{Port: 443},
				},
			},
			agentService: newAgentIPService(
				"klusterlet-addon-workmgr",
				"myns",
				metav1.OwnerReference{
					APIVersion:         "app/v1",
					Kind:               "Deployment",
					Name:               "klusterlet-addon-workmgr",
					UID:                "",
					Controller:         &bTrue,
					BlockOwnerDeletion: &bTrue,
				}),
			agentRoute: newRoute("klusterlet-addon-workmgr",
				"myns", metav1.OwnerReference{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubeObjs := []runtime.Object{
				newDeployment(test.agentName, test.agentNamespace),
			}
			if test.agentService != nil {
				kubeObjs = append(kubeObjs, test.agentService)
			}

			syncer := loggingInfoSyncer{
				clusterName:             test.managedClusterInfo.Name,
				agentName:               test.agentName,
				agentNamespace:          test.agentNamespace,
				managementClusterClient: kubefake.NewSimpleClientset(kubeObjs...),
				routeV1Client:           routefake.NewSimpleClientset(),
			}
			if test.agentRoute != nil {
				syncer.routeV1Client = routefake.NewSimpleClientset(test.agentRoute)
			}

			syncer.sync(context.TODO(), test.managedClusterInfo)

			expectLoggingPort := corev1.EndpointPort{}
			expectLoggingEndpoint := corev1.EndpointAddress{}
			assert.Equal(t, expectLoggingPort, test.managedClusterInfo.Status.LoggingPort)
			assert.Equal(t, expectLoggingEndpoint, test.managedClusterInfo.Status.LoggingEndpoint)

			_, err := syncer.managementClusterClient.CoreV1().Services(test.agentNamespace).Get(context.TODO(), test.agentName, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Errorf("unexpected err: %v", err)
			}

			_, err = syncer.routeV1Client.RouteV1().Routes(test.agentNamespace).Get(context.TODO(), test.agentName, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Errorf("unexpected err: %v", err)
			}
		})
	}
}
