// Copyright (c) 2020 Red Hat, Inc.

package apiserverreloader

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newClientset() *fake.Clientset {
	return fake.NewSimpleClientset(&v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			clientCAFileKey:              "0",
			requestHeaderClientCAFileKey: "0",
		},
	})
}

func TestReloaderWithoutCAUpdate(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	clientset := newClientset()
	reloader := NewReloader(clientset, stopChan)
	go reloader.Run()

	// wait for reloader starting
	time.Sleep(1 * time.Second)

	// update configmap then
	clientset.CoreV1().ConfigMaps(namespace).Update(&v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			clientCAFileKey:              "0",
			requestHeaderClientCAFileKey: "0",
			"others":                     "1",
		},
	})
}
