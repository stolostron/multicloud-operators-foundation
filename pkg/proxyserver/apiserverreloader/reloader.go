// Copyright (c) 2020 Red Hat, Inc.

package apiserverreloader

import (
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	namespace                    = "kube-system"
	configmapName                = "extension-apiserver-authentication"
	clientCAFileKey              = "client-ca-file"
	requestHeaderClientCAFileKey = "requestheader-client-ca-file"
	retryInterval                = 5
)

type Reloader struct {
	kubeclient kubernetes.Interface
	watch      watch.Interface
	configmap  *v1.ConfigMap
	stopCh     <-chan struct{}
}

func NewReloader(kubeclient kubernetes.Interface, stopCh <-chan struct{}) *Reloader {
	return &Reloader{
		kubeclient: kubeclient,
		stopCh:     stopCh,
	}
}

func (r *Reloader) Run() {
	for {
		select {
		case <-r.stopCh:
			if r.watch != nil {
				r.watch.Stop()
			}
		default:
			configmap, err := r.watchConfigmap()
			if err != nil {
				klog.Warningf("Failed to watch configmap [%v]: %+v", configmapName, err)
				time.Sleep(retryInterval * time.Second)
			} else {
				r.configmap = configmap
				for event := range r.watch.ResultChan() {
					if event.Type == watch.Deleted || event.Type == watch.Error {
						continue
					}

					newConfigmap := event.Object.(*v1.ConfigMap)
					if !isCAChanged(r.configmap, newConfigmap) {
						continue
					}

					klog.Infof("CA change in configmap [%v] detected, try to reload myself", configmapName)
					os.Exit(0)
				}
			}
		}
	}
}

func (r *Reloader) watchConfigmap() (*v1.ConfigMap, error) {
	configmap, err := r.kubeclient.CoreV1().ConfigMaps(namespace).Get(configmapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	watch, err := r.kubeclient.CoreV1().ConfigMaps(namespace).Watch(metav1.SingleObject(metav1.ObjectMeta{
		Name: configmapName,
	}))

	if err != nil {
		return nil, err
	}

	klog.Infof("Start watching changes on configmap [%v]", configmapName)
	r.watch = watch
	return configmap, nil
}

func isCAChanged(m1, m2 *v1.ConfigMap) bool {
	if m1.Data[clientCAFileKey] != m2.Data[clientCAFileKey] {
		return true
	}

	if m1.Data[requestHeaderClientCAFileKey] != m2.Data[requestHeaderClientCAFileKey] {
		return true
	}

	return false
}
