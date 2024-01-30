package controllers

import (
	"context"
	routeclient "github.com/openshift/client-go/route/clientset/versioned"
	clusterv1beta1 "github.com/stolostron/cluster-lifecycle-api/clusterinfo/v1beta1"
	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type loggingInfoSyncer struct {
	clusterName             string
	agentName               string
	agentNamespace          string
	managementClusterClient kubernetes.Interface
	routeV1Client           routeclient.Interface
}

// the log is got by the cluster-proxy and managedserviceaccount addons.
// in ACM 2.10:
// 1. clean up the log info in the status of clusterinfo.
// 2. delete the route and service of log.
// in ACM 2.11:
// delete the sync.
func (s *loggingInfoSyncer) sync(ctx context.Context, clusterInfo *clusterv1beta1.ManagedClusterInfo) error {
	clusterInfo.Status.LoggingEndpoint = corev1.EndpointAddress{}
	clusterInfo.Status.LoggingPort = corev1.EndpointPort{}
	if s.clusterName == "local-cluster" {
		return nil
	}

	errs := []error{}
	_, err := s.managementClusterClient.CoreV1().Services(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
	if err == nil {
		err = s.managementClusterClient.CoreV1().Services(s.agentNamespace).Delete(ctx, s.agentName, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	} else if !errors.IsNotFound(err) {
		errs = append(errs, err)
	}

	if clusterInfo.Status.DistributionInfo.Type != clusterv1beta1.DistributionTypeOCP {
		return utils.NewMultiLineAggregate(errs)
	}

	_, err = s.routeV1Client.RouteV1().Routes(s.agentNamespace).Get(ctx, s.agentName, metav1.GetOptions{})
	if err == nil {
		err = s.routeV1Client.RouteV1().Routes(s.agentNamespace).Delete(ctx, s.agentName, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, err)
		}
	} else if !errors.IsNotFound(err) {
		errs = append(errs, err)
	}

	return utils.NewMultiLineAggregate(errs)
}
