// Copyright (c) 2020 Red Hat, Inc.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app"
	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app/options"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

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
