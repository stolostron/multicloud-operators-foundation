// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"testing"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func Test_NewGetters(t *testing.T) {
	opt := NewOptions()
	opt.AddFlags(pflag.CommandLine)

	objects := []runtime.Object{
		newSecret("default", "finding-ca"),
	}
	kubeClient := k8sfake.NewSimpleClientset(objects...)
	NewGetters(opt, kubeClient)
}
