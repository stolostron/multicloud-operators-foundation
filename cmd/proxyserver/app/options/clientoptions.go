// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"github.com/spf13/pflag"
)

// ClientOptions is the options for agent client
type ClientOptions struct {
	ProxyServiceCAFile string
	ProxyServiceName   string
	ProxyServicePort   string
}

// NewClientOptions creates a new agent ClientOptions object with default values.
func NewClientOptions() *ClientOptions {
	s := &ClientOptions{
		ProxyServiceCAFile: "/var/run/clusterproxy/service-ca.crt",
		ProxyServiceName:   "cluster-proxy-addon-user",
		ProxyServicePort:   "9092",
	}

	return s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *ClientOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.ProxyServiceName, "proxy-service-name", s.ProxyServiceName, ""+
		"Proxy service name")
	fs.StringVar(&s.ProxyServicePort, "proxy-service-port", s.ProxyServicePort, ""+
		"Proxy service port")
	fs.StringVar(&s.ProxyServiceCAFile, "proxy-service-cafile", s.ProxyServiceCAFile, ""+
		"Proxy service CA file name")
}
