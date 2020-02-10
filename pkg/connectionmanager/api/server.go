// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package api

import (
	"fmt"

	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/connectionmanager/common"
	corev1 "k8s.io/api/core/v1"
)

// ServerInfo maintains the info for a hub server
type ServerInfo struct {
	name string
	host string
	conn *ServerConnection
}

// NewServerInfo return a server info
func NewServerInfo(name, host string, conn *ServerConnection) *ServerInfo {
	return &ServerInfo{
		name: name,
		host: host,
		conn: conn,
	}
}

// SecretGetter defines an interface for looking up a secret by name
type SecretGetter interface {
	Get(namespace, name string) (*corev1.Secret, error)
}

// SecretGetterFunc allows implementing SecretGetter with a function
type SecretGetterFunc func(namespace, name string) (*corev1.Secret, error)

// Get defins a secret getter function
func (f SecretGetterFunc) Get(namespace, name string) (*corev1.Secret, error) {
	return f(namespace, name)
}

// LoadBootstrapServerInfo load bootstrap server info
func LoadBootstrapServerInfo(
	bootstrapSecret *corev1.Secret, scgetter SecretGetter, clusterName, clusterNamespace string) (string, *ServerInfo, error) {
	bootstrapConfig := bootstrapSecret.Data[common.HubConfigSecretKey]
	if bootstrapConfig == nil {
		return "", nil, fmt.Errorf("bootstrap config is not found")
	}
	conn, host, err := NewServerConnection(bootstrapConfig, nil, nil, nil, clusterName, clusterNamespace, "")
	if err != nil {
		return "", nil, err
	}
	return host, NewServerInfo("", host, conn), nil
}

// SetName set server name
func (s *ServerInfo) SetName(name string) {
	s.name = name
}

// Name return the identity of the server
func (s *ServerInfo) Name() string {
	return s.name
}

// Conn return server conn
func (s *ServerInfo) Conn() *ServerConnection {
	return s.conn
}

// Host return apiserver host
func (s *ServerInfo) Host() string {
	return s.host
}
