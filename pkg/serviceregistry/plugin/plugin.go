// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// ClusterInfo represents a cluster information
type ClusterInfo struct {
	Name   string
	Zone   string
	Region string
}

// ServiceAddress represents a service address
type ServiceAddress struct {
	// The IP of this service
	IP string
	// The load balancer hostname of this service
	Hostname string
}

// ServiceLocation represents a service location
type ServiceLocation struct {
	// The address of this service
	Address ServiceAddress
	// The host domains of this service
	Hosts []string
	// The cluster information of this service
	ClusterInfo ClusterInfo
}

// Plugin registers annotated resource as an endpoints to hub cluster and (or) discovers other member cluster's resouces from hub
// cluster as an endpoints
type Plugin interface {
	// GetType returns the type of current plugin
	GetType() string

	// RegisterAnnotatedResouceHandler generates resource creation, update and deletion events (as an endpoints) and dispatches
	// them to registry handler to register them to hub cluster
	RegisterAnnotatedResouceHandler(registryHandler cache.ResourceEventHandlerFuncs)

	// SyncRegisteredEndpoints synchronizes the annotated resouces and their corresponding registered endpoints and returns the
	// changed endpoints to create, delete or update
	SyncRegisteredEndpoints(registeredEndpoints []*v1.Endpoints) (toCreate, toDelete, toUpdate []*v1.Endpoints)

	// DiscoveryRequired returns a flag to mark the plugin needs to handle the hub cluster auto discovered endpoints
	DiscoveryRequired() bool

	// SyncDiscoveredResouces synchronizes the hub discovered resources (as endpoints), if plugin want to record these resource to
	// CoreDNS, this method must set dnsRequired with true and returns the resource locations
	SyncDiscoveredResouces(discoveredEndpoints []*v1.Endpoints) (dnsRequired bool, locations []*ServiceLocation)
}
