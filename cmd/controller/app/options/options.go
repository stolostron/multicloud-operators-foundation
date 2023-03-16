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
	fs.DurationVar(&o.LeaseDuration, "leader-election-lease-duration", 15*time.Second, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	fs.DurationVar(&o.RenewDeadline, "leader-election-renew-deadline", 10*time.Second, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	fs.DurationVar(&o.RetryPeriod, "leader-election-retry-period", 2*time.Second, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
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
