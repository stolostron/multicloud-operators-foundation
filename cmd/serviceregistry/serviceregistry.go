// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/serviceregistry/app"
	"github.com/spf13/pflag"

	"k8s.io/component-base/logs"
)

func main() {
	command := app.NewCommand()

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	if err := flag.CommandLine.Parse([]string{}); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
