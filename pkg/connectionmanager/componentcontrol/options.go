// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package componentcontrol

import (
	"github.com/spf13/pflag"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ControlOptions for the component controller.
type ControlOptions struct {
	KlusterletSecret string
	KlusterletLabels string
}

// NewControlOptions creates a new ControlOptions object with default values.
func NewControlOptions() *ControlOptions {
	s := ControlOptions{
		KlusterletSecret: "",
		KlusterletLabels: "",
	}

	return &s
}

// AddFlags adds flags for ServerRunOptions fields to be specified via FlagSet.
func (s *ControlOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.KlusterletLabels, "klusterlet-labels", "",
		"Klusterlet labels to be managed")
	fs.StringVar(&s.KlusterletSecret, "klusterlet-secret", "",
		"Klusterlet secret name in the format of namespace/name")
}

// ComponentControl returns a controller
func (s *ControlOptions) ComponentControl(kubeclient kubernetes.Interface) *Controller {
	controller := &Controller{
		kubeclient:       kubeclient,
		klusterletLabels: s.KlusterletLabels,
	}

	if s.KlusterletSecret != "" {
		namespace, secname, err := cache.SplitMetaNamespaceKey(s.KlusterletSecret)
		if err == nil {
			controller.klusterletSecretName = secname
			controller.klusterletSecretNamespace = namespace
		}
	}

	return controller
}
