// Copyright (c) 2020 Red Hat, Inc.

package app

import (
	"strings"
	"time"

	"k8s.io/client-go/dynamic"

	apilabels "k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/open-cluster-management/multicloud-operators-foundation/cmd/acm-proxyserver/app/options"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/apiserverreloader"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/controller"
	"github.com/open-cluster-management/multicloud-operators-foundation/pkg/proxyserver/getter"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func Run(s *options.Options, stopCh <-chan struct{}) error {
	if err := s.SetDefaults(); err != nil {
		return err
	}

	if errs := s.Validate(); len(errs) != 0 {
		return utilerrors.NewAggregate(errs)
	}

	clusterCfg, err := clientcmd.BuildConfigFromFlags("", s.KubeConfigFile)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(clusterCfg)
	if err != nil {
		return err
	}

	configMapLabels, err := apilabels.ConvertSelectorToLabelsMap(strings.TrimSuffix(s.ConfigMapLabels, ","))
	if err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(kubeClient, 10*time.Minute)
	proxyGetter := getter.NewProxyServiceInfoGetter()
	ctrl := controller.NewProxyServiceInfoController(kubeClient, configMapLabels, informerFactory, proxyGetter, stopCh)
	go ctrl.Run()
	informerFactory.Start(stopCh)

	reloader := apiserverreloader.NewReloader(kubeClient, stopCh)
	go reloader.Run()

	apiServerConfig, err := s.APIServerConfig()
	if err != nil {
		return err
	}

	logGetter, err := getter.NewLogConnectionInfoGetter(s.ClientOptions.Config(dynamicClient))
	if err != nil {
		return nil
	}

	proxyServer, err := NewProxyServer(informerFactory, apiServerConfig, proxyGetter, logGetter)
	if err != nil {
		return err
	}
	return proxyServer.Run(stopCh)
}
