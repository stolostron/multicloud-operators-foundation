// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.ibm.com/IBMPrivateCloud/multicloud-operators-foundation/cmd/serviceregistry/app"
)

func main() {
	command := app.NewCommand()

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	flag.CommandLine.Parse([]string{})

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
