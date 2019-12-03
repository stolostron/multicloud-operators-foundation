// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package manager

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	operatorapi "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/common"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/componentcontrol"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/genericoptions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Operator is to define an operator to manage mcm components
type Operator struct {
	kubeclient          kubernetes.Interface
	clusterNamespace    string
	clusterName         string
	componentController *componentcontrol.Controller

	server          *operatorapi.ServerInfo
	bootstrapServer *operatorapi.ServerInfo
	stopCh          <-chan struct{}
}

// NewOperator start a operator function
func NewOperator(
	genericConfig *genericoptions.GenericConfig,
	clusterName, clusterNamespace string,
	stopCh <-chan struct{}) *Operator {
	kubeclient := genericConfig.Kubeclient
	scgetter := operatorapi.SecretGetterFunc(func(namespace, name string) (*corev1.Secret, error) {
		return kubeclient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	})

	btsec := genericConfig.BootstrapSecret
	_, bootstrapServer, err := operatorapi.LoadBootstrapServerInfo(btsec, scgetter, clusterName, clusterNamespace)
	if err != nil {
		klog.Errorf("failed to load bootstrap server: %s", err)
	}

	operator := &Operator{
		kubeclient:          kubeclient,
		bootstrapServer:     bootstrapServer,
		clusterName:         clusterName,
		clusterNamespace:    clusterNamespace,
		componentController: genericConfig.ComponentControl,
		stopCh:              stopCh,
	}

	return operator
}

// Run is the main run loop of kluster server
func (o *Operator) Run() error {
	defer utilruntime.HandleCrash()

	o.bootstrapAll()

	go wait.Until(o.refreshKlusterletSecret, 5*time.Second, o.stopCh)

	<-o.stopCh
	klog.Info("Shutting operator")

	return nil
}

func (o *Operator) bootstrapAll() {
	secret, err := o.componentController.GetKlusterletSecret()
	if err != nil {
		klog.Fatalf("failed to get klusterlet secret. %+v", err)
	}

	if secret == nil {
		if o.bootstrapServer == nil {
			klog.Fatalf("bootstrap server is nil, cannot start")
		}

		klog.Infof("start to bootstrap")
		err = o.bootstrapServer.Conn().Bootstrap()
		if err != nil {
			klog.Fatalf("failed to bootstrap: %v", err)
		}

		created, err := o.componentController.UpdateKlusterletSecret(o.bootstrapServer.Conn().ConnInfo())
		if err != nil {
			klog.Fatalf("failed to create klusterlet secret: %v", err)
		} else if created {
			klog.Infof("klusterlet secret created")
			err = o.componentController.RestartKlusterlet()
			if err != nil {
				klog.Errorf("failed to restart klusterlet: %v", err)
			} else {
				klog.Infof("klusterlet restarted")
			}
		}

		o.server = o.bootstrapServer
	} else {
		// reconnect to server with klusterlet secret if exists
		klog.Infof("already had connection to server, try to reconnect")
		if err = o.reconnectToServer(secret); err != nil {
			klog.Fatalf("failed to reconnect to server: %v. try to re-bootstrap", err)
		}
	}

	if err = o.server.Conn().MonitorCert(o.stopCh); err != nil {
		klog.Fatalf("failed to monitor cert on server: %v", err)
	}

	klog.Infof("server connected")
}

func (o *Operator) reconnectToServer(secret *corev1.Secret) error {
	kubeConfig := secret.Data[common.HubConfigSecretKey]
	if kubeConfig == nil {
		return fmt.Errorf("%v is not found in klusterlet secret", common.HubConfigSecretKey)
	}

	_, key, cert, err := common.GetHostCerKeyFromClientConfig(kubeConfig)
	if err != nil {
		return err
	}

	conn, host, err := operatorapi.NewServerConnection(nil, kubeConfig, cert, key, o.clusterName, o.clusterNamespace, "")
	if err != nil {
		return err
	}

	o.server = operatorapi.NewServerInfo("", host, conn)
	return nil
}

func (o *Operator) refreshKlusterletSecret() {
	// update klusterlet secret
	updated, err := o.componentController.UpdateKlusterletSecret(o.server.Conn().ConnInfo())
	if err != nil {
		klog.Errorf("failed to update klusterlet secret: %v", err)
	} else if updated {
		klog.Infof("klusterlet secret updated")
		err = o.componentController.RestartKlusterlet()
		if err != nil {
			klog.Errorf("failed to restart klusterlet: %v", err)
		} else {
			klog.Infof("klusterlet restarted")
		}
	}
}
