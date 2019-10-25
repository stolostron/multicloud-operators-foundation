// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"strings"

	"k8s.io/klog"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/controller"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// PluginFactory is factory to start plugin
type PluginFactory struct {
	memberKubeClient   kubernetes.Interface
	hubKubeClient      kubernetes.Interface
	hubInformerFactory informers.SharedInformerFactory
	options            *options.SvcRegistryOptions
	plugins            map[string]plugin.Plugin
}

// NewPluginFactory returns a plugin factory
func NewPluginFactory(memberKubeClient, hubKubeClient kubernetes.Interface,
	hubInformerFactory informers.SharedInformerFactory,
	options *options.SvcRegistryOptions) *PluginFactory {
	return &PluginFactory{
		memberKubeClient:   memberKubeClient,
		hubKubeClient:      hubKubeClient,
		hubInformerFactory: hubInformerFactory,
		options:            options,
		plugins:            make(map[string]plugin.Plugin),
	}
}

// Start initializes all requested plugin
func (f *PluginFactory) Start(stopCh <-chan struct{}) {
	enabledPlugins := []plugin.Plugin{}
	// start to each enabled plugins
	for _, pluginType := range f.options.EnabledPlugins {
		enabledPlugin, ok := f.plugins[strings.TrimSpace(pluginType)]
		if !ok {
			klog.Warningf("the plugin (%s) does not register, ignore it", strings.TrimSpace(pluginType))
			continue
		}

		ctrl := controller.NewPluginController(
			f.hubKubeClient,
			f.hubInformerFactory,
			f.options.ClusterNamespace,
			f.options.ClusterName,
			enabledPlugin,
			f.options.SyncPeriod,
			stopCh)
		go ctrl.Run()

		enabledPlugins = append(enabledPlugins, enabledPlugin)
	}

	if len(enabledPlugins) == 0 {
		klog.Warningf("there are no enabled plugins")
		return
	}

	ctrl := controller.NewEndpointsController(
		f.memberKubeClient,
		f.hubInformerFactory,
		plugin.ClusterInfo{
			Name: f.options.ClusterName,
		},
		f.options.ClusterNamespace,
		f.options.DNSConfigmap,
		f.options.DNSSuffix,
		enabledPlugins,
		f.options.SyncPeriod,
		stopCh)
	go ctrl.Run()
}

// Register registers a plugin to factory
func (f *PluginFactory) Register(plugin plugin.Plugin) {
	if plugin == nil {
		return
	}
	f.plugins[plugin.GetType()] = plugin
}
