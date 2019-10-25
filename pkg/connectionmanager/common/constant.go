// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package common

const (
	// BootStrapSuffix is the suffix of the key pointing to the bootstrap secret
	BootStrapSuffix = "-bootstrap-secret"
	// BootStrapConfig is the key of bootstrap secret
	BootStrapConfig = "bootstrap-secret"
	// HubConfigSuffix is the suffix of the key pointing to the hub config secret
	HubConfigSuffix = "-config-secret"
	// HubConfigSecretKey is the key of the hub kubeconfig in secret
	HubConfigSecretKey = "kubeconfig"
	// ClientCertFileName is the file name of tls cert
	ClientCertFileName = "tls.crt"
	// ClientKeyFileName is the file name of tls key
	ClientKeyFileName = "tls.key"
)
