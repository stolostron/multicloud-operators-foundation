// Copyright (c) 2020 Red Hat, Inc.

package main

import (
	"fmt"
	"os"

	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app"
	"github.com/stolostron/multicloud-operators-foundation/cmd/controller/app/options"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	"github.com/spf13/pflag"

	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	fmt.Println("Test PR: Starting controller with test log")
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	s := options.NewControllerRunOptions()
	s.AddFlags(pflag.CommandLine)
	klog.InitFlags(nil)
	flag.InitFlags()

	logs.InitLogs()
	defer logs.FlushLogs()

	ctx := signals.SetupSignalHandler()
	if err := app.Run(s, ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
