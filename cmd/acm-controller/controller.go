// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller/app"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-controller/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/signals"

	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"github.com/spf13/pflag"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	s := options.NewControllerRunOptions()
	s.AddFlags(pflag.CommandLine)
	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	stopCh := signals.SetupSignalHandler()
	if err := app.Run(s, stopCh); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
