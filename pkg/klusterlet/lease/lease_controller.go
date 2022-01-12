//###############################################################################
//# Copyright (c) 2020 Red Hat, Inc.
//###############################################################################

package controllers

import (
	"context"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/pkg/utils"
	coordinationv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

var (
	agentLabel = map[string]string{"app": "work-manager"}
)

// LeaseReconciler reconciles a Secret object
type LeaseReconciler struct {
	KubeClient           kubernetes.Interface
	LeaseName            string
	LeaseNamespace       string
	HubKubeConfigPath    string
	LeaseDurationSeconds int32
	cachedKubeConfig     []byte
	componentNamespace   string
	hubClient            kubernetes.Interface
}

func (r *LeaseReconciler) Reconcile(ctx context.Context) {
	curKubeConfig, err := ioutil.ReadFile(r.HubKubeConfigPath)
	if err != nil || len(curKubeConfig) == 0 {
		klog.Errorf("Failed to get hub kubeconfig. path: %v", r.HubKubeConfigPath)
		return
	}
	if !reflect.DeepEqual(r.cachedKubeConfig, curKubeConfig) {
		if len(r.componentNamespace) == 0 {
			r.componentNamespace, err = utils.GetComponentNamespace()
			if err != nil {
				klog.Errorf("failed to get pod namespace use. error:%v", err)
			}
		}
		if len(r.cachedKubeConfig) != 0 {
			//If kubeconfig changed, restart agent pods
			labelSelector := labels.FormatLabels(agentLabel)
			err = r.KubeClient.CoreV1().Pods(r.componentNamespace).DeleteCollection(context.TODO(), metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labelSelector})
			if err != nil {
				klog.Errorf("failed to restart pod. error:%v", err)
			}
			return
		}

		//update cached kubeconfig and hub client
		r.hubClient, err = utils.BuildKubeClient(r.HubKubeConfigPath)
		if err != nil {
			klog.Errorf("failed to build hub client. error:%v", err)
			return
		}
		r.cachedKubeConfig = curKubeConfig
	}

	lease, err := r.hubClient.CoordinationV1().Leases(r.LeaseNamespace).Get(context.TODO(), r.LeaseName, metav1.GetOptions{})
	switch {
	case errors.IsNotFound(err):
		//create lease
		lease := &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.LeaseName,
				Namespace: r.LeaseNamespace,
			},
			Spec: coordinationv1.LeaseSpec{
				LeaseDurationSeconds: &r.LeaseDurationSeconds,
				RenewTime: &metav1.MicroTime{
					Time: time.Now(),
				},
			},
		}
		if _, err := r.hubClient.CoordinationV1().Leases(r.LeaseNamespace).Create(context.TODO(), lease, metav1.CreateOptions{}); err != nil {
			klog.Errorf("unable to create addon lease %q/%q on hub cluster. error:%v", r.LeaseNamespace, r.LeaseName, err)
		}
		return
	case err != nil:
		klog.Errorf("unable to get addon lease %q/%q on hub cluster. error:%v", r.LeaseNamespace, r.LeaseName, err)
		return
	default:
		//update lease
		lease.Spec.RenewTime = &metav1.MicroTime{Time: time.Now()}
		if _, err = r.hubClient.CoordinationV1().Leases(r.LeaseNamespace).Update(context.TODO(), lease, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("unable to update cluster lease %q/%q on hub cluster. error:%v", r.LeaseNamespace, r.LeaseName, err)
		}
		return
	}
}
