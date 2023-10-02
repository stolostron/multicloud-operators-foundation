// Copyright (c) 2020 Red Hat, Inc.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/pflag"
	"github.com/stolostron/multicloud-operators-foundation/cmd/proxyserver/app"
	"github.com/stolostron/multicloud-operators-foundation/cmd/proxyserver/app/options"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"k8s.io/klog/v2"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	opts := options.NewOptions()
	opts.AddFlags(pflag.CommandLine)

	klog.InitFlags(nil)
	flag.InitFlags()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := app.Run(opts, wait.NeverStop); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
