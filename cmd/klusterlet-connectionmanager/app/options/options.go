// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package options

import (
	"github.com/spf13/pflag"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/pkg/connectionmanager/genericoptions"
)

// RunOptions for the klusterlet operator.
type RunOptions struct {
	Cluster     string
	Generic     *genericoptions.GenericOptions
	LeaderElect bool
}

// NewRunOptions creates a new RunOptions object with default values.
func NewRunOptions() *RunOptions {
	s := RunOptions{
		Generic: genericoptions.NewGenericOptions(),
		Cluster: "",
	}

	return &s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *RunOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.Cluster, "cluster", "",
		"cluster in the format of namespace/name")
	fs.BoolVar(&s.LeaderElect, "leader-elect", false, "Enable a leader client to gain leadership before executing the main loop")
	s.Generic.AddFlags(fs)
}
