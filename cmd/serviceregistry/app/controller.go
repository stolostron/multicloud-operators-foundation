// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app/options"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/clients"
	istioplugin "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin/istio"
	kubeingressplugin "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin/kubeingress"
	kubeserviceplugin "github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/serviceregistry/plugin/kubeservice"
	"k8s.io/client-go/informers"
)

// NewCommand return the service registry command
func NewCommand() *cobra.Command {
	commandOptions := options.GetSvcRegistryOptions()
	rootCmd := &cobra.Command{
		Use:   "mcm-service-registry-plugin",
		Short: "MCM Service Registry Plugin",
		Long:  "MCM Service Registry Plugin",
		Run: func(cmd *cobra.Command, args []string) {
			start()
		},
	}
	commandOptions.AddFlags(rootCmd.Flags())
	return rootCmd
}

func start() {
	stopCh := make(chan struct{})
	defer close(stopCh)

	commandOptions := options.GetSvcRegistryOptions()

	memberKubeClient := clients.MemberKubeClient()
	memberInformerFactory := informers.NewSharedInformerFactory(memberKubeClient, time.Minute*10)

	hubKubeClient := clients.HubKubeClient(commandOptions)
	hubInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		hubKubeClient,
		time.Minute*10,
		informers.WithNamespace(commandOptions.ClusterNamespace),
	)

	pluginFactory := NewPluginFactory(memberKubeClient, hubKubeClient, hubInformerFactory, commandOptions)

	// register plugins
	pluginFactory.Register(kubeserviceplugin.NewKubeServicePlugin(memberKubeClient, memberInformerFactory, commandOptions))
	pluginFactory.Register(kubeingressplugin.NewKubeIngressPlugin(memberInformerFactory, commandOptions))
	pluginFactory.Register(istioplugin.NewIstioPlugin(memberKubeClient, clients.MemberDynamicKubeClient(), memberInformerFactory, commandOptions))

	// start factories
	pluginFactory.Start(stopCh)
	memberInformerFactory.Start(stopCh)
	hubInformerFactory.Start(stopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}
