// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package componentcontrol

import (
	"bytes"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Controller is the controller to manager mcm component
type Controller struct {
	kubeclient                kubernetes.Interface
	klusterletSecretNamespace string
	klusterletSecretName      string
	klusterletLabels          string
}

// RestartKlusterlet restart all klusterlet compoenents
func (c *Controller) RestartKlusterlet() error {
	if c.klusterletLabels != "" {
		podList, err := c.kubeclient.CoreV1().Pods(metav1.NamespaceAll).List(metav1.ListOptions{
			LabelSelector: c.klusterletLabels,
		})

		if err != nil {
			return err
		}

		for _, pod := range podList.Items {
			err = c.kubeclient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			if err != nil {
				klog.Errorf("failed to delete pod %s: %v", pod.Name, err)
			}
		}
	}

	return nil
}

// UpdateKlusterletSecret update klusterlet secret, if this secret should be updated, return true
func (c *Controller) UpdateKlusterletSecret(data []byte) (bool, error) {
	secret, err := c.kubeclient.CoreV1().Secrets(c.klusterletSecretNamespace).Get(c.klusterletSecretName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, err
		}

		_, createErr := c.kubeclient.CoreV1().Secrets(c.klusterletSecretNamespace).Create(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   c.klusterletSecretName,
				Labels: map[string]string{"compnent": "operator"},
			},
			Data: map[string][]byte{common.HubConfigSecretKey: data},
		})

		if createErr == nil {
			klog.V(4).Infof("Klusterlet secret [%s/%s] created", c.klusterletSecretNamespace, c.klusterletSecretName)
		}

		return true, createErr
	}
	if bytes.Equal(secret.Data[common.HubConfigSecretKey], data) {
		return false, nil
	}
	secret.Data[common.HubConfigSecretKey] = data
	_, err = c.kubeclient.CoreV1().Secrets(c.klusterletSecretNamespace).Update(secret)
	if err != nil {
		return true, err
	}

	klog.V(4).Infof("Klusterlet secret [%s/%s] updated", c.klusterletSecretNamespace, c.klusterletSecretName)
	return true, nil
}

// GetKlusterletSecret return klusterlet secret if exists
func (c *Controller) GetKlusterletSecret() (*corev1.Secret, error) {
	secret, err := c.kubeclient.CoreV1().Secrets(c.klusterletSecretNamespace).Get(c.klusterletSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return secret, nil
}
