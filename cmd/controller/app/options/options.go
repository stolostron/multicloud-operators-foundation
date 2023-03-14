// Copyright (c) 2020 Red Hat, Inc.

package options

import (
	"time"

	"github.com/spf13/pflag"
)

// ControllerRunOptions for the hcm controller.
type ControllerRunOptions struct {
	KubeConfig            string
	CAFile                string
	LogCertSecret         string
	EnableInventory       bool
	EnableLeaderElection  bool
	LeaseDuration         time.Duration
	RenewDeadline         time.Duration
	RetryPeriod           time.Duration
	EnableAddonDeploy     bool
	AddonImage            string
	AddonInstallNamespace string
	QPS                   float32
	Burst                 int
}

// NewControllerRunOptions creates a new ServerRunOptions object with default values.
func NewControllerRunOptions() *ControllerRunOptions {
	return &ControllerRunOptions{
		KubeConfig:            "",
		CAFile:                "/var/run/agent/ca.crt",
		LogCertSecret:         "ocm-klusterlet-self-signed-secrets",
		EnableInventory:       true,
		EnableLeaderElection:  true,
		EnableAddonDeploy:     false,
		QPS:                   100.0,
		Burst:                 200,
		AddonImage:            "quay.io/stolostron/multicloud-manager:latest",
		AddonInstallNamespace: "open-cluster-management-agent-addon",
	}
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (o *ControllerRunOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfig, "kubeconfig", "",
		"The kubeconfig to connect to cluster to watch/apply resources.")
	fs.StringVar(&o.CAFile, "agent-cafile", o.CAFile, ""+
		"Agent CA file.")
	fs.StringVar(&o.LogCertSecret, "log-cert-secret", o.LogCertSecret,
		"log cert secret name.")
	fs.BoolVar(&o.EnableInventory, "enable-inventory", o.EnableInventory,
		"enable multi-cluster inventory")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", o.EnableLeaderElection,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	fs.DurationVar(&o.LeaseDuration, "lease-duration", o.LeaseDuration,
		"Non-leader candidates will wait to force acquire leadership.")
	fs.DurationVar(&o.RenewDeadline, "renew-deadline", o.RenewDeadline,
		"Acting controlplane will retry refreshing leadership before giving up.")
	fs.DurationVar(&o.RetryPeriod, "retry-period", o.RetryPeriod,
		"LeaderElector clients should wait between tries of actions.")
	fs.BoolVar(&o.EnableAddonDeploy, "enable-agent-deploy", o.EnableAddonDeploy,
		"Enable deploy addon agent.")
	fs.StringVar(&o.AddonImage, "agent-addon-image", o.AddonImage,
		"image of the addon agent to deploy.")
	fs.StringVar(&o.AddonInstallNamespace, "agent-addon-install-namespace", o.AddonInstallNamespace,
		"namespace on the managed cluster for the addon agent to install.")
	fs.Float32Var(&o.QPS, "max-qps", o.QPS,
		"Maximum QPS to the hub server from this controller.")
	fs.IntVar(&o.Burst, "max-burst", o.Burst,
		"Maximum burst for throttle.")
}
