package controllers

import (
	"context"
	"fmt"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	routefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/klusterlet/agent"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"strings"
	"testing"
)

var bTrue = true

func newDeployment(name, namespace string) *v1.Deployment {
	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func newConfigIngress() *configv1.Ingress {
	return &configv1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.IngressSpec{
			Domain: "mydomain.com",
		},
	}
}
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
					KubeVendor: v1beta1.KubeVendorOpenShift,
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
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
					KubeVendor: v1beta1.KubeVendorOpenShift,
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
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
					KubeVendor: v1beta1.KubeVendorAKS,
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
			name:           "mciroshift",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorMicroShift,
				},
			},
		},
		{
			name:           "non-ocp update service",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorAKS,
				},
			},
			agentService: newAgentLBService(
				"klusterlet-addon-workmgr",
				"myns",
				metav1.OwnerReference{}),
		},
		{
			name:           "ocp update route",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "myns",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorOpenShift,
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
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
		{
			name:           "ocp update route with hash",
			agentName:      "klusterlet-addon-workmgr",
			agentNamespace: "open-cluster-management-cluster1-addon-workmanager",
			managedClusterInfo: &v1beta1.ManagedClusterInfo{
				ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "c1"},
				Status: v1beta1.ClusterInfoStatus{
					KubeVendor: v1beta1.KubeVendorOpenShift,
					DistributionInfo: v1beta1.DistributionInfo{
						Type: "OCP",
						OCP:  v1beta1.OCPDistributionInfo{},
					},
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
				agentPort:               443,
				agent:                   nil,
				managementClusterClient: kubefake.NewSimpleClientset(kubeObjs...),
				configV1Client:          configfake.NewSimpleClientset(newConfigIngress()),
				routeV1Client:           routefake.NewSimpleClientset(),
			}
			if test.agentRoute != nil {
				syncer.routeV1Client = routefake.NewSimpleClientset(test.agentRoute)
			}
			syncer.agent = agent.NewAgent("c1", syncer.managementClusterClient)

			syncer.sync(context.TODO(), test.managedClusterInfo)

			expectLoggingPort := corev1.EndpointPort{
				Name:     "https",
				Protocol: corev1.ProtocolTCP,
				Port:     443,
			}
			assert.Equal(t, expectLoggingPort, test.managedClusterInfo.Status.LoggingPort)

			svr, err := syncer.managementClusterClient.CoreV1().Services(test.agentNamespace).Get(context.TODO(), test.agentName, metav1.GetOptions{})

			if test.managedClusterInfo.Status.KubeVendor == "" || test.managedClusterInfo.Status.KubeVendor == v1beta1.KubeVendorMicroShift {
				assert.Equal(t, true, errors.IsNotFound(err))
				return
			}

			if err != nil {
				t.Errorf("failed to get service")
			}

			if test.managedClusterInfo.Status.DistributionInfo.Type == v1beta1.DistributionTypeOCP {
				if test.managedClusterInfo.Name == "local-cluster" {
					assert.Equal(t, test.managedClusterInfo.Status.LoggingEndpoint.Hostname,
						fmt.Sprintf("%s.%s.svc", test.agentName, test.agentNamespace))
				} else {
					route, err := syncer.routeV1Client.RouteV1().Routes(test.agentNamespace).Get(context.TODO(), test.agentName, metav1.GetOptions{})
					if err != nil {
						t.Errorf("failed to get route")
					}
					assert.Equal(t, test.managedClusterInfo.Status.LoggingEndpoint.Hostname, route.Spec.Host)
					subNames := strings.Split(route.Spec.Host, ".")
					if len(subNames[0]) > 40 {
						t.Errorf("the length of subName %s of route host cannot exceed 40", subNames[0])
					}
				}
				assert.Equal(t, svr.Spec.Type, corev1.ServiceTypeClusterIP)
			} else {
				assert.Equal(t, svr.Spec.Type, corev1.ServiceTypeLoadBalancer)
				if len(svr.Status.LoadBalancer.Ingress) != 0 {
					assert.Equal(t, test.managedClusterInfo.Status.LoggingEndpoint.IP, svr.Status.LoadBalancer.Ingress[0].IP)
					assert.Equal(t, test.managedClusterInfo.Status.LoggingEndpoint.Hostname, svr.Status.LoadBalancer.Ingress[0].Hostname)
				}
			}

		})
	}
}
