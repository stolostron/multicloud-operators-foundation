// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package api

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"k8s.io/klog"

	mcmv1alpha1 "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/apis/mcm/v1alpha1"
	hcmclientset "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/client/clientset_generated/clientset"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/clusterbootstrap"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/common"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	initWaitTime = 5 * time.Second
	maxWaitTime  = 120 * time.Second
)

// ServerConnection defines a hub server connection
type ServerConnection struct {
	bootstrapper     *clusterbootstrap.BootStrapper
	hcmclient        hcmclientset.Interface
	clusterName      string
	clusterNamespace string
	clientconfig     []byte
	clientCert       []byte
	clientKey        []byte
	hubName          string
	infoLock         sync.RWMutex
}

// NewServerConnection returns a hub ServerConnection
func NewServerConnection(
	bootstrapperconfig, clientconfig []byte, clientCert, clientKey []byte,
	clusterName, clusterNamespace, hubName string) (*ServerConnection, string, error) {
	conn := &ServerConnection{
		clientconfig:     clientconfig,
		clientCert:       clientCert,
		clientKey:        clientKey,
		hubName:          hubName,
		clusterName:      clusterName,
		clusterNamespace: clusterNamespace,
	}

	var host string
	if bootstrapperconfig != nil {
		btrestconfig, err := clientcmd.RESTConfigFromKubeConfig(bootstrapperconfig)
		if err != nil {
			return nil, "", err
		}

		host = btrestconfig.Host
		btclient, err := hcmclientset.NewForConfig(btrestconfig)
		if err != nil {
			return nil, "", err
		}

		bootstrapper := clusterbootstrap.NewBootStrapper(
			btclient, btrestconfig.Host, clusterNamespace, clusterName, clientKey, clientCert)

		conn.bootstrapper = bootstrapper
	} else if clientconfig != nil {
		cliconfig, err := clientcmd.RESTConfigFromKubeConfig(clientconfig)
		if err != nil {
			return nil, "", err
		}
		host = cliconfig.Host

		client, err := generateClient(clientconfig)
		if err != nil {
			return nil, "", err
		}
		conn.hcmclient = client
	}

	return conn, host, nil
}

// NewMyselfServerConnection returns a myself server connection
func NewMyselfServerConnection(hcmclient hcmclientset.Interface) *ServerConnection {
	return &ServerConnection{
		hcmclient: hcmclient,
	}
}

// Bootstrap initialze configuration and credential to hub
func (conn *ServerConnection) Bootstrap() error {
	conn.infoLock.Lock()
	defer conn.infoLock.Unlock()
	if conn.clientconfig == nil {
		if conn.bootstrapper == nil {
			return fmt.Errorf("bootstrapper is empty to hub")
		}
		waitTime := initWaitTime
		for {
			key, cert, config, err := conn.bootstrapper.LoadClientCert()
			if err != nil {
				if !errors.IsNotFound(err) &&
					!errors.IsServiceUnavailable(err) &&
					!errors.IsTimeout(err) &&
					!errors.IsInternalError(err) &&
					!errors.IsServerTimeout(err) {
					return err
				}
				klog.Infof("wait to hub (%s) approve cluster join request, %v", conn.hubName, err)
				time.Sleep(waitTime)
				if waitTime < maxWaitTime {
					waitTime += initWaitTime
				} else {
					waitTime = maxWaitTime
				}
				continue
			}

			conn.clientCert = cert
			conn.clientKey = key
			conn.clientconfig = config
			break
		}
	}

	client, err := generateClient(conn.clientconfig)
	if err != nil {
		return err
	}
	conn.hcmclient = client
	return nil
}

func generateClient(clientConfig []byte) (hcmclientset.Interface, error) {
	restconfig, err := clientcmd.RESTConfigFromKubeConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	client, err := hcmclientset.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// MonitorCert monitor certificate change on clusterjoinrequest and update
// cert on connection
func (conn *ServerConnection) MonitorCert(ctx context.Context, callback func()) {
	subject := clusterbootstrap.Subject(conn.clusterName, conn.clusterNamespace)
	clusterJoinName := clusterbootstrap.DigestedName(conn.clientKey, subject)
	fieldSelector := fields.OneTermEqualSelector("metadata.name", clusterJoinName).String()
	indexers := cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return conn.hcmclient.McmV1alpha1().ClusterJoinRequests().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return conn.hcmclient.McmV1alpha1().ClusterJoinRequests().Watch(options)
			},
		},
		&mcmv1alpha1.ClusterJoinRequest{},
		5*time.Minute,
		indexers,
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newRequest := obj.(*mcmv1alpha1.ClusterJoinRequest)
			conn.handleRequestUpdate(newRequest, callback)
		},
		UpdateFunc: func(old, new interface{}) {
			newRequest := new.(*mcmv1alpha1.ClusterJoinRequest)
			conn.handleRequestUpdate(newRequest, callback)
		},
	})

	go informer.Run(ctx.Done())
	klog.V(4).Infof("start monitoring certificate rotation on hub server")
}

// refresh cert and related config and client
func (conn *ServerConnection) handleRequestUpdate(request *mcmv1alpha1.ClusterJoinRequest, callback func()) {
	conn.infoLock.Lock()
	defer conn.infoLock.Unlock()
	certificate := request.Status.CSRStatus.Certificate
	if !reflect.DeepEqual(certificate, conn.clientCert) {
		klog.V(4).Infof("handle certification rotate request")
		conn.clientCert = certificate
		host, _, _, err := common.GetHostCerKeyFromClientConfig(conn.clientconfig)
		if err != nil {
			klog.Errorf("failed to get host from config")
			return
		}
		conn.clientconfig = common.NewClientConfig(host, conn.clientCert, conn.clientKey)
		client, err := generateClient(conn.clientconfig)
		if err != nil {
			klog.Errorf("failed to generate client")
			return
		}
		conn.hcmclient = client

		// call callback function
		if callback != nil {
			callback()
		}
	}
}

// Ping checks the availability of the hub server
func (conn *ServerConnection) Ping() error {
	if conn.hcmclient == nil {
		return fmt.Errorf("client is not correctly init")
	}

	_, err := conn.hcmclient.Discovery().ServerVersion()
	return err
}

// SetClient set clients for connection, just for testing
func (conn *ServerConnection) SetClient(client hcmclientset.Interface) {
	conn.hcmclient = client
}

// ConnInfo returns configuration to connect to hub
func (conn *ServerConnection) ConnInfo() []byte {
	conn.infoLock.RLock()
	defer conn.infoLock.RUnlock()
	return conn.clientconfig
}

// KeyCert returns key and cert
func (conn *ServerConnection) KeyCert() ([]byte, []byte) {
	conn.infoLock.RLock()
	defer conn.infoLock.RUnlock()
	return conn.clientKey, conn.clientCert
}
