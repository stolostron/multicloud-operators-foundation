// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"github.com/spf13/pflag"
	kubeclientset "k8s.io/client-go/kubernetes"
)

// SearchOptions is the options for aggregator client
type SearchOptions struct {
	Connection *ConnectionOption
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *SearchOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Connection.Secret, "aggregator-secret", s.Connection.Secret, ""+
		"Secret file for search aggregator connection")
	fs.StringVar(&s.Connection.Host, "aggregator-host", s.Connection.Host, ""+
		"Aggregator host Name")
}

// NewGetter returns a new NewInfoGetters
func (s *SearchOptions) NewGetter(client kubeclientset.Interface) *ConnectionInfoGetter {
	return NewConnectionInfoGetter(s.Connection, client, "/aggregator/clusters/")
}
