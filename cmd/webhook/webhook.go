// Copyright (c) 2020 Red Hat, Inc.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app"
	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/webhook/app/options"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	opts := options.NewOptions()
	opts.AddFlags(pflag.CommandLine)
	flag.InitFlags()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := app.Run(opts, wait.NeverStop); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
