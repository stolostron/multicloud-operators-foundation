// licensed Materials - Property of IBM
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package aggregator

import (
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/aggregator/v1alpha1"
	"k8s.io/client-go/informers"
	kubeclientset "k8s.io/client-go/kubernetes"
)

type Getters struct {
	V1alpha1InfoGetters *v1alpha1.InfoGetters
	InfoGetters         *InfoGetters
	client              kubeclientset.Interface
}

func NewGetters(o *Options, client kubeclientset.Interface) *Getters {
	return &Getters{
		V1alpha1InfoGetters: v1alpha1.NewInfoGetters(o.V1alpha1Options, client),
		InfoGetters:         NewInfoGetters(client),
		client:              client,
	}
}

func (g *Getters) Run(informerFactory informers.SharedInformerFactory, stopCh <-chan struct{}) {
	controller := NewController(g.client, informerFactory, g.InfoGetters, stopCh)
	go controller.Run()
}
