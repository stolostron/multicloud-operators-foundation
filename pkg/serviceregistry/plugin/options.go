// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package plugin

import "github.com/spf13/pflag"

// IstioPluginOptions the options for istio plugin
type IstioPluginOptions struct {
	IstioIngressGateway           string
	ServiceEntryRegistryNamespace string
}

// NewIstioPluginOptions returns istio plugin options
func NewIstioPluginOptions() *IstioPluginOptions {
	return &IstioPluginOptions{
		IstioIngressGateway:           "istio-system/istio-ingressgateway",
		ServiceEntryRegistryNamespace: "default",
	}
}

// AddFlags add flags to kube-service plugin options
func (o *IstioPluginOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&o.IstioIngressGateway,
		"istio-ingressgateway",
		o.IstioIngressGateway,
		"The name of Istio ingressgateway, format with namespace/name",
	)
	fs.StringVar(
		&o.ServiceEntryRegistryNamespace,
		"istio-serviceentry-registry-namespace",
		o.ServiceEntryRegistryNamespace,
		"The registry namespace of Istio ServiceEntry",
	)
}
