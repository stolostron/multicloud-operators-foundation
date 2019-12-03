// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package apiserverreloader

import (
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newClientset() *fake.Clientset {
	items := make([]v1.Pod, 2)
	for i := 0; i < len(items); i++ {
		items[i] = v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod%d", i+1),
				Namespace: namespace,
				Labels: map[string]string{
					"component": "mcm-apiserver",
				},
			},
		}
	}

	return fake.NewSimpleClientset(&v1.PodList{
		Items: items,
	}, &v1.ConfigMap{
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

func TestReloadAPIServer(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	clientset := newClientset()

	pods, _ := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("component=%s", componentName),
	})

	if len(pods.Items) != 2 {
		t.Errorf("Expect 2 pods, but found %d", len(pods.Items))
	}

	reloader := NewReloader(clientset, stopChan)
	go reloader.Run()

	// wait for reloader starting
	time.Sleep(5 * time.Second)

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
			clientCAFileKey:              "1",
			requestHeaderClientCAFileKey: "1",
		},
	})

	// wait for reloader deleting pods
	time.Sleep(10 * time.Second)
	pods, _ = clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("component=%s", componentName),
	})

	if len(pods.Items) != 0 {
		t.Errorf("Expect 0 pod, but found %d", len(pods.Items))
	}
}

func TestReloaderWithoutCAUpdate(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	clientset := newClientset()

	pods, _ := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("component=%s", componentName),
	})

	if len(pods.Items) != 2 {
		t.Errorf("Expect 2 pods, but found %d", len(pods.Items))
	}

	reloader := NewReloader(clientset, stopChan)
	go reloader.Run()

	// wait for reloader starting
	time.Sleep(5 * time.Second)

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

	// wait for reloader to handle change
	time.Sleep(5 * time.Second)
	pods, _ = clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("component=%s", componentName),
	})

	if len(pods.Items) != 2 {
		t.Errorf("Expect 2 pod, but found %d", len(pods.Items))
	}
}
