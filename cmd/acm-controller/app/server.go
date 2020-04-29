// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package app

import (
	"fmt"
	"os"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/acm-controller/inventory"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/signals"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	componentKlusterlet = "controller"
)

// NewControllerCommand creates a *cobra.Command object with default parameters
func NewControllerCommand() *cobra.Command {
	s := options.NewControllerRunOptions()
	cmd := &cobra.Command{
		Use:  componentKlusterlet,
		Long: ``,
		Run: func(cmd *cobra.Command, args []string) {
			stopCh := signals.SetupSignalHandler()
			if err := Run(s, stopCh); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	s.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the specified klusterlet.  It only returns if stopCh is closed
// or one of the ports cannot be listened on initially.
func Run(s *options.ControllerRunOptions, stopCh <-chan struct{}) error {
	err := RunController(s, stopCh)
	if err != nil {
		klog.Fatalf("Error run controller: %s", err.Error())
	}
	return nil
}

// RunController start a hcm controller server
func RunController(s *options.ControllerRunOptions, stopCh <-chan struct{}) error {
	hcmCfg, err := clientcmd.BuildConfigFromFlags("", s.APIServerConfigFile)
	if err != nil {
		klog.Fatalf("Error building config to connect to api: %s", err.Error())
	}

	// Configure qps and maxburst
	hcmCfg.QPS = s.QPS
	hcmCfg.Burst = s.Burst

	if s.EnableInventory {
		go inventory.Run(hcmCfg, stopCh)
	}

	return nil
}
