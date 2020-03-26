// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package apiserverreloader

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

const (
	namespace                    = "kube-system"
	configmapName                = "extension-apiserver-authentication"
	appName                      = "mcm-apiserver"
	clientCAFileKey              = "client-ca-file"
	requestHeaderClientCAFileKey = "requestheader-client-ca-file"
	maxRetryNumber               = 3
	retryInterval                = 5
	reloadingInterval            = 5
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
	var cancel context.CancelFunc
	for {
		select {
		case <-r.stopCh:
			if r.watch != nil {
				r.watch.Stop()
			}

			if cancel != nil {
				cancel()
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
					r.configmap = newConfigmap

					klog.Infof("CA change in configmap [%v] detected, try to reload API server", configmapName)
					if cancel != nil {
						cancel()
					}

					var ctx context.Context
					ctx, cancel = context.WithCancel(context.Background())
					go r.reload(ctx)
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

func (r *Reloader) reload(ctx context.Context) {
	done := false
	for i := 0; i < maxRetryNumber; i++ {
		err := r.reloadAPIServer(ctx)
		if err == nil {
			done = true
			break
		}

		if i != maxRetryNumber-1 {
			time.Sleep(retryInterval * time.Second)
		}
	}

	if !done {
		klog.Errorf("Failed to reloade API server after %d retries", maxRetryNumber)
	}
}

func (r *Reloader) reloadAPIServer(ctx context.Context) error {
	// get all pods of api server
	pods, err := r.kubeclient.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})

	if err != nil {
		klog.Errorf("Caught error when pod list of API server: %v", err)
		return err
	}

	nPods := len(pods.Items)

	for i := 0; i < nPods; i++ {
		select {
		case <-ctx.Done():
			klog.Infof("Reloading of API server is cancelled")
			return nil
		default:
			// delete pod
			podNamespace := pods.Items[i].GetNamespace()
			podName := pods.Items[i].GetName()
			err := r.kubeclient.CoreV1().Pods(podNamespace).Delete(podName, &metav1.DeleteOptions{})
			if err != nil {
				klog.Errorf("Caught error when deleting pod [%v] of API server: %v", podName, err)
				return err
			}
			klog.V(4).Infof("%d/%d pod(s) of API server deleted", i+1, nPods)
			if i != nPods-1 {
				time.Sleep(reloadingInterval * time.Second)
			}
		}
	}

	klog.Infof("API server is reloaded successfully")
	return nil
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
