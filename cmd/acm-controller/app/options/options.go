// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package options

import (
	"github.com/spf13/pflag"
)

// ControllerRunOptions for the hcm controller.
type ControllerRunOptions struct {
	APIServerConfigFile string
	EnableInventory     bool
	LeaderElect         bool
	QPS                 float32
	Burst               int
}

// NewControllerRunOptions creates a new ServerRunOptions object with default values.
func NewControllerRunOptions() *ControllerRunOptions {
	s := ControllerRunOptions{
		APIServerConfigFile: "",
		EnableInventory:     false,
		QPS:                 100.0,
		Burst:               200,
	}

	return &s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *ControllerRunOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.APIServerConfigFile, "apiserver-config-file", "",
		"Klusterlet configuration file to connect to hcm api-server")
	fs.BoolVar(&s.EnableInventory, "enable-inventory", s.EnableInventory,
		"enable multi-cluster inventory")
	fs.Float32Var(&s.QPS, "max-qps", s.QPS,
		"Maximum QPS to the hub server from this controller")
	fs.IntVar(&s.Burst, "max-burst", s.Burst,
		"Maximum burst for throttle")
}
