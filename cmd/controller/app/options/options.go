// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"github.com/spf13/pflag"
)

// ControllerRunOptions for the hcm controller.
type ControllerRunOptions struct {
	KubeConfig           string
	CAFile               string
	EnableInventory      bool
	EnableLeaderElection bool
	EnableAddon          bool
	QPS                  float32
	Burst                int
}

// NewControllerRunOptions creates a new ServerRunOptions object with default values.
func NewControllerRunOptions() *ControllerRunOptions {
	return &ControllerRunOptions{
		KubeConfig:           "",
		CAFile:               "/var/run/agent/ca.crt",
		EnableInventory:      true,
		EnableLeaderElection: true,
		EnableAddon:          false,
		QPS:                  100.0,
		Burst:                200,
	}
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (o *ControllerRunOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfig, "kubeconfig", "",
		"The kubeconfig to connect to cluster to watch/apply resources.")
	fs.StringVar(&o.CAFile, "agent-cafile", o.CAFile, ""+
		"Agent CA file.")
	fs.BoolVar(&o.EnableInventory, "enable-inventory", o.EnableInventory,
		"enable multi-cluster inventory")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.BoolVar(&o.EnableAddon, "enable-addon-registration", o.EnableAddon,
		"Enable addon registration.")
	fs.Float32Var(&o.QPS, "max-qps", o.QPS,
		"Maximum QPS to the hub server from this controller.")
	fs.IntVar(&o.Burst, "max-burst", o.Burst,
		"Maximum burst for throttle.")
}
