// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator/v1alpha1"
	"github.com/spf13/pflag"
)

// ClientOptions is the options for aggregator client
type Options struct {
	V1alpha1Options *v1alpha1.Options
}

// NewOptions creates a new NewOptions object with default values.
func NewOptions() *Options {
	return &Options{
		V1alpha1Options: v1alpha1.NewOptions(),
	}
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *Options) AddFlags(fs *pflag.FlagSet) {
	s.V1alpha1Options.AddFlags(fs)
}
