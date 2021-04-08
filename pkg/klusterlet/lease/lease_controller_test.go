//###############################################################################
//# Copyright (c) 2020 Red Hat, Inc.
//###############################################################################

package controllers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	leaseName = "lease"
	podName   = "pod"
	agentNs   = "open-cluster-management-agent"
)

var ns = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: agentNs,
	},
}
var pod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      podName,
		Namespace: agentNs,
		Labels:    agentLabel,
	},
	Status: corev1.PodStatus{
		Phase: corev1.PodRunning,
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		},
	},
}

func TestLeaseReconciler_Reconcile(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Namespace{})
	kubeClient := kubefake.NewSimpleClientset(ns, pod)

	leaseReconciler := &LeaseReconciler{
		KubeClient:           kubeClient,
		LeaseName:            leaseName,
		LeaseDurationSeconds: 1,
		componentNamespace:   agentNs,
	}

	//create lease
	leaseReconciler.Reconcile(context.TODO())
	if !actionExist(kubeClient, "create") {
		t.Errorf("failed to create lease")
	}

	//update lease
	leaseReconciler.Reconcile(context.TODO())
	if !actionExist(kubeClient, "update") {
		t.Errorf("failed to update lease")
	}

	//update componentnamespace, cached kubeconfig and client
	leaseReconciler2 := &LeaseReconciler{
		KubeClient:           kubeClient,
		LeaseName:            leaseName,
		componentNamespace:   "",
		LeaseDurationSeconds: 1,
	}
	leaseReconciler2.Reconcile(context.TODO())
	//kubeconfig can not work, so cached kubeconfig should also be nil
	if leaseReconciler2.componentNamespace == "" {
		t.Errorf("failed to update component namespace")
	}
}

func actionExist(kubeClient *kubefake.Clientset, existAction string) bool {
	for _, action := range kubeClient.Actions() {
		if action.GetVerb() == existAction {
			return true
		}
	}
	return false
}
