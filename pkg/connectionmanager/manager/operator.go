// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package manager

import (
	"bytes"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	hcmclientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	operatorapi "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/api"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/common"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/componentcontrol"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/genericoptions"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

// Operator is to define an operator to manage mcm components
type Operator struct {
	kubeclient                kubernetes.Interface
	clusterNamespace          string
	clusterName               string
	klusterletSecretNamespace string
	klusterletSecretName      string
	componentController       *componentcontrol.Controller

	server          *operatorapi.ServerInfo
	bootstrapServer *operatorapi.ServerInfo
	certManager     operatorapi.CertManager

	// fields for klusterlet secrets (ks)
	ksinformer  cache.SharedIndexInformer
	ksstore     cache.Store
	ksworkqueue workqueue.RateLimitingInterface

	stopCh <-chan struct{}
}

type queueHandlerFunc func(key string) error

// NewOperator start a operator function
func NewOperator(
	genericConfig *genericoptions.GenericConfig,
	clusterName, clusterNamespace string,
	klusterletSecretNamespace, klusterletSecretName string,
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

	fieldSelector := fields.OneTermEqualSelector("metadata.name", klusterletSecretName).String()
	indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return kubeclient.CoreV1().Secrets(klusterletSecretNamespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return kubeclient.CoreV1().Secrets(klusterletSecretNamespace).Watch(options)
			},
		},
		&corev1.Secret{},
		5*time.Minute,
		indexers,
	)

	operator := &Operator{
		kubeclient:                kubeclient,
		bootstrapServer:           bootstrapServer,
		clusterName:               clusterName,
		clusterNamespace:          clusterNamespace,
		klusterletSecretNamespace: klusterletSecretNamespace,
		klusterletSecretName:      klusterletSecretName,
		componentController:       genericConfig.ComponentControl,
		ksinformer:                informer,
		ksstore:                   informer.GetStore(),
		ksworkqueue:               workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		stopCh:                    stopCh,
	}

	store := &klusterletSecretStore{operator.componentController}
	operator.certManager = operatorapi.NewCertManager(clusterNamespace, clusterName, store, operator.getHcmClient)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			operator.enqueue(obj, operator.ksworkqueue)
		},
		UpdateFunc: func(old, new interface{}) {
			operator.enqueue(new, operator.ksworkqueue)
		},
	})

	return operator
}

// Run is the main run loop of kluster server
func (o *Operator) Run() {
	defer utilruntime.HandleCrash()
	defer o.ksworkqueue.ShutDown()
	defer o.certManager.Stop()

	go o.ksinformer.Run(o.stopCh)
	o.bootstrap()

	go wait.Until(o.runKlusterletSecretWorker, time.Second, o.stopCh)

	<-o.stopCh
	klog.Info("Shutting operator")
}

func (o *Operator) bootstrap() {
	secret, err := o.componentController.GetKlusterletSecret()
	if err != nil {
		klog.Fatalf("failed to get klusterlet secret. %+v", err)
	}

	if secret == nil {
		if o.bootstrapServer == nil {
			klog.Fatalf("bootstrap server is nil, cannot start")
		}

		klog.Infof("Start to bootstrap")
		err = o.bootstrapServer.Conn().Bootstrap()
		if err != nil {
			klog.Fatalf("failed to bootstrap: %v", err)
		}

		_, err := o.componentController.UpdateKlusterletSecret(o.bootstrapServer.Conn().ConnInfo())
		if err != nil {
			klog.Fatalf("failed to create klusterlet secret: %v", err)
		}
		klog.Infof("Bootstrap completed")
	} else {
		klog.Infof("Klusterlet secret exists. Skip bootstrap")

		// reconnect to hub server with kubeconfig in secret
		server, err := o.setupServer(secret)
		if err != nil {
			klog.Errorf("failed to reconnect to hub server: %v", err)
			return
		}

		o.server = server
		klog.Infof("Hub server (%s) connected", o.server.Host())

		// start cert rotation
		o.certManager.Start()
	}
}

func (o *Operator) handleKlusterletSecretChange(key string) error {
	obj, exists, err := o.ksstore.GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		// The secrect resource may no longer exist, in which case we stop processing.
		utilruntime.HandleError(fmt.Errorf("work '%s' in work queue no longer exists", key))
		return nil
	}

	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return fmt.Errorf("expected secret but got %#v", obj)
	}

	// do nothing if the current server exists and uses the same secret
	if o.server != nil && bytes.Equal(secret.Data[common.HubConfigSecretKey], o.server.Conn().ConnInfo()) {
		return nil
	} else if o.server != nil {
		klog.Infof("Klusterlet secret changed. Try to connect to hub server with new configuration")
	}

	// otherwise, setup a server
	server, err := o.setupServer(secret)
	if err != nil {
		return err
	}

	o.server = server
	klog.Infof("Hub server (%s) connected", o.server.Host())

	// restart cert rotation
	o.certManager.Restart()

	// restart other components, like klusterlet
	err = o.componentController.RestartKlusterlet()
	if err != nil {
		klog.Errorf("failed to restart klusterlet: %v", err)
	} else {
		klog.Infof("klusterlet restarted")
	}

	return nil
}

func (o *Operator) setupServer(secret *corev1.Secret) (*operatorapi.ServerInfo, error) {
	kubeConfig := secret.Data[common.HubConfigSecretKey]
	if kubeConfig == nil {
		return nil, fmt.Errorf("%v is not found in klusterlet secret", common.HubConfigSecretKey)
	}

	_, key, cert, err := common.GetHostCerKeyFromClientConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	conn, host, err := operatorapi.NewServerConnection(nil, kubeConfig, cert, key, o.clusterName, o.clusterNamespace, "")
	if err != nil {
		return nil, err
	}

	return operatorapi.NewServerInfo("", host, conn), nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (o *Operator) runKlusterletSecretWorker() {
	for o.processNextWorkItem(o.ksworkqueue, o.handleKlusterletSecretChange) {
	}
}

// enqueue takes a resource and converts it into a name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Work.
func (o *Operator) enqueue(obj interface{}, queue workqueue.RateLimitingInterface) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	queue.AddRateLimited(key)
}

func (o *Operator) processNextWorkItem(queue workqueue.RateLimitingInterface, fn queueHandlerFunc) bool {
	obj, shutdown := queue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer queue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			queue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := fn(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		queue.Forget(obj)
		klog.V(4).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (o *Operator) getHcmClient() hcmclientset.Interface {
	if o.server == nil {
		return nil
	}

	return o.server.Conn().GetHcmClient()
}

type klusterletSecretStore struct {
	componentController *componentcontrol.Controller
}

func (s *klusterletSecretStore) Current() (*clientcmdapi.Config, error) {
	secret, err := s.componentController.GetKlusterletSecret()
	if err != nil {
		return nil, err
	}

	config, ok := secret.Data[common.HubConfigSecretKey]
	if !ok {
		return nil, fmt.Errorf("no key [%s] found in secret", common.HubConfigSecretKey)
	}

	clientConfigObj, err := runtime.Decode(clientcmdlatest.Codec, config)
	if err != nil {
		return nil, err
	}

	clientConfig, ok := clientConfigObj.(*clientcmdapi.Config)
	if !ok {
		return nil, fmt.Errorf("wrong type of client config data")
	}

	return clientConfig, nil
}

func (s *klusterletSecretStore) Update(config []byte) (bool, error) {
	return s.componentController.UpdateKlusterletSecret(config)
}
