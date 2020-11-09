//###############################################################################
//# Copyright (c) 2020 Red Hat, Inc.
//###############################################################################

package controllers

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	leaseName      = "lease"
	leaseNamespace = "lease-ns"
	podName        = "pod"
	agentNs        = "open-cluster-management-agent"
)

var ns = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: leaseNamespace,
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
	kubeconfigData := []byte("default kubeconfig")

	//init kubeconfig
	tempdir, err := ioutil.TempDir("", "kube")
	if err != nil {
		return
	}
	defer os.RemoveAll(tempdir)
	if err := ioutil.WriteFile(path.Join(tempdir, "kubeconfig"), kubeconfigData, 0600); err != nil {
		t.Errorf("Failed to generator kubeconfig. error: %v", err)
	}

	leaseReconciler := &LeaseReconciler{
		KubeClient:           kubeClient,
		LeaseName:            leaseName,
		LeaseNamespace:       leaseNamespace,
		LeaseDurationSeconds: 1,
		cachedKubeConfig:     kubeconfigData,
		HubKubeConfigPath:    path.Join(tempdir, "kubeconfig"),
		hubClient:            kubeClient,
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
		LeaseNamespace:       leaseNamespace,
		componentNamespace:   "",
		LeaseDurationSeconds: 1,
		cachedKubeConfig:     nil,
		HubKubeConfigPath:    path.Join(tempdir, "kubeconfig"),
		hubClient:            kubeClient,
	}
	leaseReconciler2.Reconcile(context.TODO())
	//kubeconfig can not work, so cached kubeconfig should also be nil
	if leaseReconciler2.cachedKubeConfig != nil {
		t.Errorf("cached kubeconfig should be nil")
	}
	if leaseReconciler2.componentNamespace == "" {
		t.Errorf("failed to update component namespace")
	}

	//delete pods
	leaseReconciler3 := &LeaseReconciler{
		KubeClient:           kubeClient,
		LeaseName:            leaseName,
		LeaseNamespace:       leaseNamespace,
		componentNamespace:   "",
		LeaseDurationSeconds: 1,
		cachedKubeConfig:     []byte("cached data"),
		HubKubeConfigPath:    path.Join(tempdir, "kubeconfig"),
		hubClient:            kubeClient,
	}
	leaseReconciler3.Reconcile(context.TODO())
	if !actionExist(kubeClient, "delete-collection") {
		t.Errorf("failed to delete pods")
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
