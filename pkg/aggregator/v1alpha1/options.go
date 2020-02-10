// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package v1alpha1

import (
	"github.com/spf13/pflag"
	kubeclientset "k8s.io/client-go/kubernetes"
)

// Options is the options for aggregator client
type Options struct {
	Metering *MeteringOptions
	Search   *SearchOptions
}

// NewOptions creates a new NewOptions object with default values.
func NewOptions() *Options {
	return &Options{
		Metering: &MeteringOptions{Connection: NewConnectionOption()},
		Search:   &SearchOptions{Connection: NewConnectionOption()},
	}
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *Options) AddFlags(fs *pflag.FlagSet) {
	s.Metering.AddFlags(fs)
	s.Search.AddFlags(fs)
}

// NewGetter returns a new NewInfoGetters
func (s *Options) NewGetter(client kubeclientset.Interface) *InfoGetters {
	return NewInfoGetters(s, client)
}

// ConnectionOption is the common connection options
type ConnectionOption struct {
	Secret string
	Host   string
}

// NewConnectionOption creates a new ConnectionOption object with default values.
func NewConnectionOption() *ConnectionOption {
	s := &ConnectionOption{
		Secret: "",
		Host:   "",
	}

	return s
}
